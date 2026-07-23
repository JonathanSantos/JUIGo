package state

import "juigo/internal/hooks"

// State é um valor observável tipado — a base da reatividade do JUIGo.
// Widgets podem ser vinculados a um State (Text.BindText, Input.BindValue);
// chamar Set notifica os observadores e agenda um redesenho da interface
// automaticamente, sem Invalidate manual.
//
// Como todo o JUIGo, State é single-threaded: use apenas na main thread.
type State[T any] struct {
	value    T
	watchers []func(T)
}

// NewState cria um State com o valor inicial dado.
func New[T any](v T) *State[T] {
	return &State[T]{value: v}
}

// Get devolve o valor atual.
func (s *State[T]) Get() T {
	return s.value
}

// Set define o valor, notifica os observadores sincronamente (na ordem de
// registro) e agenda um frame. Não deduplica: definir o mesmo valor notifica
// de novo.
//
// Repintura: os bindings e setters de widgets reportam as REGIÕES afetadas
// (dano parcial); se nenhum observador tocar a interface, o frame é pulado.
// Widgets personalizados que leem estados diretamente no Draw devem se
// invalidar num Watch (Watch(func(T){ w.Invalidate() })); mutar campos
// públicos por fora dos setters exige App.Invalidate().
func (s *State[T]) Set(v T) {
	s.value = v
	for _, fn := range s.watchers {
		fn(v)
	}
	hooks.RequestFrame()
}

// Watch registra fn para ser chamada a cada Set, com o novo valor.
func (s *State[T]) Watch(fn func(T)) {
	s.watchers = append(s.watchers, fn)
}

// Observable é o aspecto sem tipo de um State: permite observar mudanças
// sem conhecer o valor — a base de Combine.
type Observable interface {
	// WatchChange registra fn para ser chamada a cada mudança.
	WatchChange(fn func())
}

// WatchChange registra fn para ser chamada a cada Set, sem o valor.
func (s *State[T]) WatchChange(fn func()) {
	s.Watch(func(T) { fn() })
}

// Combine deriva um State de VÁRIAS fontes: compute é reavaliada a cada
// mudança de qualquer uma delas (e uma vez na criação). Trate o derivado
// como somente-leitura.
//
//	total := state.Combine(func() int { return a.Get() + b.Get() }, a, b)
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
func Map[A, B any](s *State[A], f func(A) B) *State[B] {
	d := New(f(s.Get()))
	s.Watch(func(v A) {
		d.Set(f(v))
	})
	return d
}
