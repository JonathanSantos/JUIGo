package juigo

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
func NewState[T any](v T) *State[T] {
	return &State[T]{value: v}
}

// Get devolve o valor atual.
func (s *State[T]) Get() T {
	return s.value
}

// Set define o valor, notifica os observadores sincronamente (na ordem de
// registro) e agenda um redesenho da interface. Não deduplica: definir o
// mesmo valor notifica de novo.
func (s *State[T]) Set(v T) {
	s.value = v
	for _, fn := range s.watchers {
		fn(v)
	}
	requestRepaint()
}

// Watch registra fn para ser chamada a cada Set, com o novo valor.
func (s *State[T]) Watch(fn func(T)) {
	s.watchers = append(s.watchers, fn)
}

// Map deriva um State de outro: o derivado recebe f(valor) a cada Set do
// original. Trate o derivado como somente-leitura — Sets diretos nele serão
// sobrescritos pela próxima atualização do original.
func Map[A, B any](s *State[A], f func(A) B) *State[B] {
	d := NewState(f(s.Get()))
	s.Watch(func(v A) {
		d.Set(f(v))
	})
	return d
}
