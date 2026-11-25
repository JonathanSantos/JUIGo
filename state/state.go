package state

import (
	"weak"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
)

// State é um valor observável tipado — a base da reatividade do JUIGo.
// Widgets podem ser vinculados a um State (Text.BindText, Input.BindValue);
// chamar Set notifica os observadores e agenda um redesenho da interface
// automaticamente, sem Invalidate manual.
//
// Como todo o JUIGo, State é single-threaded: use apenas na main thread.
type State[T any] struct {
	value    T
	watchers []watcher[T]
	nextID   uint64
	// notifying conta os Set em andamento (um observador pode chamar Set de
	// novo); cancelamentos durante a notificação só marcam a entrada e a
	// compactação fica para o fim do Set mais externo.
	notifying int
	stale     bool
}

// watcher é uma assinatura: fn nil marca a entrada como cancelada.
type watcher[T any] struct {
	id uint64
	fn func(T)
}

// New cria um State com o valor inicial dado.
func New[T any](v T) *State[T] {
	return &State[T]{value: v}
}

// Get devolve o valor atual.
func (s *State[T]) Get() T {
	return s.value
}

// Set define o valor, notifica os observadores sincronamente (na ordem de
// registro) e agenda um frame. Não deduplica: definir o mesmo valor notifica
// de novo. Observadores registrados durante a notificação só passam a valer
// no próximo Set; os cancelados durante ela deixam de ser chamados na hora.
//
// Repintura: os bindings e setters de widgets reportam as REGIÕES afetadas
// (dano parcial); se nenhum observador tocar a interface, o frame é pulado.
// Widgets personalizados que leem estados diretamente no Draw devem se
// invalidar num Watch (Watch(func(T){ w.Invalidate() })); mutar campos
// públicos por fora dos setters exige App.Invalidate().
func (s *State[T]) Set(v T) {
	s.value = v
	s.notifying++
	n := len(s.watchers) // os registrados durante a notificação ficam de fora
	for i := 0; i < n && i < len(s.watchers); i++ {
		if fn := s.watchers[i].fn; fn != nil {
			fn(v)
		}
	}
	s.notifying--
	if s.notifying == 0 && s.stale {
		s.compact()
	}
	hooks.RequestFrame()
}

// Watch registra fn para ser chamada a cada Set, com o novo valor, e devolve
// a função que CANCELA a assinatura (idempotente). Ignorar o retorno mantém
// a assinatura pelo tempo de vida do State — o caso comum de estados e
// observadores que vivem tanto quanto a aplicação.
//
// O State guarda fn com referência FORTE: tudo o que fn captura vive
// enquanto a assinatura existir. Para observar em nome de um objeto de vida
// curta (um widget de uma lista reconstruída), use WatchWeak.
func (s *State[T]) Watch(fn func(T)) (cancel func()) {
	s.nextID++
	id := s.nextID
	s.watchers = append(s.watchers, watcher[T]{id: id, fn: fn})
	return func() { s.unwatch(id) }
}

// unwatch cancela a assinatura id; fora de uma notificação a entrada sai da
// lista na hora.
func (s *State[T]) unwatch(id uint64) {
	for i := range s.watchers {
		if s.watchers[i].id != id {
			continue
		}
		if s.notifying > 0 {
			s.watchers[i].fn = nil
			s.stale = true
			return
		}
		copy(s.watchers[i:], s.watchers[i+1:])
		last := len(s.watchers) - 1
		s.watchers[last] = watcher[T]{} // solta a closure
		s.watchers = s.watchers[:last]
		return
	}
}

// compact remove as entradas canceladas durante uma notificação.
func (s *State[T]) compact() {
	kept := s.watchers[:0]
	for _, w := range s.watchers {
		if w.fn != nil {
			kept = append(kept, w)
		}
	}
	clear(s.watchers[len(kept):])
	s.watchers = kept
	s.stale = false
}

// Watchers devolve quantas assinaturas ativas o State tem — útil em testes
// e para diagnosticar vazamentos de observadores.
func (s *State[T]) Watchers() int {
	n := 0
	for _, w := range s.watchers {
		if w.fn != nil {
			n++
		}
	}
	return n
}

// WatchWeak registra fn para ser chamada a cada Set de s EM NOME de owner,
// sem manter owner vivo: o State guarda só uma referência fraca, e quando
// owner é coletado pelo GC a assinatura se cancela sozinha no Set seguinte.
// É a base de todo Bind* dos widgets — um widget descartado (lista
// reconstruída com Clear+Add) não fica preso ao State nem recebe mais
// notificações depois de coletado.
//
// fn recebe owner de volta e NÃO deve capturá-lo (nem nada que o referencie)
// por fora do parâmetro — isso recriaria a referência forte. Devolve a
// função que cancela a assinatura antes disso.
func WatchWeak[T, O any](s *State[T], owner *O, fn func(owner *O, v T)) (cancel func()) {
	ref := weak.Make(owner)
	var stop func()
	stop = s.Watch(func(v T) {
		o := ref.Value()
		if o == nil {
			stop()
			return
		}
		fn(o, v)
	})
	return stop
}

// Observable é o aspecto sem tipo de um State: permite observar mudanças
// sem conhecer o valor — a base de Combine.
type Observable interface {
	// WatchChange registra fn para ser chamada a cada mudança e devolve a
	// função que cancela a assinatura.
	WatchChange(fn func()) (cancel func())
}

// WatchChange registra fn para ser chamada a cada Set, sem o valor, e
// devolve a função que cancela a assinatura.
func (s *State[T]) WatchChange(fn func()) (cancel func()) {
	return s.Watch(func(T) { fn() })
}

// Combine deriva um State de VÁRIAS fontes: compute é reavaliada a cada
// mudança de qualquer uma delas (e uma vez na criação). Trate o derivado
// como somente-leitura.
//
//	total := state.Combine(func() int { return a.Get() + b.Get() }, a, b)
//
// As fontes mantêm o derivado vivo enquanto elas viverem: crie derivados
// junto dos estados (uma vez), não a cada reconstrução de interface.
func Combine[T any](compute func() T, sources ...Observable) *State[T] {
	d := New(compute())
	for _, src := range sources {
		src.WatchChange(func() {
			d.Set(compute())
		})
	}
	return d
}

// Map deriva um State de outro: o derivado recebe f(valor) a cada Set do
// original. Trate o derivado como somente-leitura — Sets diretos nele serão
// sobrescritos pela próxima atualização do original.
//
// O original mantém o derivado vivo enquanto ele próprio viver: crie
// derivados junto dos estados (uma vez), não a cada reconstrução de
// interface.
func Map[A, B any](s *State[A], f func(A) B) *State[B] {
	d := New(f(s.Get()))
	s.Watch(func(v A) {
		d.Set(f(v))
	})
	return d
}
