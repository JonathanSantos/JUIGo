package state

// History guarda um valor com pilhas de desfazer/refazer — a infraestrutura
// do padrão undo/redo sem cerimônia:
//
//	hist := state.NewHistory([]Forma{})
//	hist.Commit(comNovaForma)                       // vira um ponto de undo
//	anterior, ok := hist.Undo()
//	juigo.BindDisabled(desfazer, state.Not(hist.CanUndo()))
//
// O History NÃO copia valores: com tipos mutáveis (slices, maps), entregue
// sempre um valor novo a Commit — ou use CommitFrom com a cópia do estado
// anterior quando a edição mutou o valor no lugar (ajustes ao vivo).
type History[T any] struct {
	current      T
	past, future []T
	canUndo      *State[bool]
	canRedo      *State[bool]
}

// NewHistory cria o histórico com o valor inicial (que não conta como ponto
// de undo).
func NewHistory[T any](initial T) *History[T] {
	return &History[T]{current: initial, canUndo: New(false), canRedo: New(false)}
}

// Get devolve o valor atual.
func (h *History[T]) Get() T {
	return h.current
}

// Replace troca o valor atual SEM registrar ponto de undo — para ajustes ao
// vivo que só viram história no fim (ver CommitFrom).
func (h *History[T]) Replace(v T) {
	h.current = v
}

// Commit registra o valor atual como ponto de undo, passa a v e descarta o
// refazer.
func (h *History[T]) Commit(v T) {
	h.past = append(h.past, h.current)
	h.current = v
	h.future = nil
	h.sync()
}

// CommitFrom registra "anterior" como ponto de undo do valor ATUAL — para
// edições que mutaram o valor no lugar: guarde a cópia antes de editar e
// entregue-a aqui ao concluir.
func (h *History[T]) CommitFrom(anterior T) {
	h.past = append(h.past, anterior)
	h.future = nil
	h.sync()
}

// Undo volta um passo e devolve o novo valor atual; false se não há o que
// desfazer.
func (h *History[T]) Undo() (T, bool) {
	if len(h.past) == 0 {
		var zero T
		return zero, false
	}
	h.future = append(h.future, h.current)
	h.current = h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	h.sync()
	return h.current, true
}

// Redo repete o passo desfeito e devolve o novo valor atual; false se não
// há o que refazer.
func (h *History[T]) Redo() (T, bool) {
	if len(h.future) == 0 {
		var zero T
		return zero, false
	}
	h.past = append(h.past, h.current)
	h.current = h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.sync()
	return h.current, true
}

// CanUndo devolve o State que informa se há o que desfazer — pronto para
// prender botões (combine com Not para BindDisabled).
func (h *History[T]) CanUndo() *State[bool] {
	return h.canUndo
}

// CanRedo devolve o State que informa se há o que refazer.
func (h *History[T]) CanRedo() *State[bool] {
	return h.canRedo
}

// sync espelha as pilhas nos estados observáveis.
func (h *History[T]) sync() {
	if v := len(h.past) > 0; h.canUndo.Get() != v {
		h.canUndo.Set(v)
	}
	if v := len(h.future) > 0; h.canRedo.Get() != v {
		h.canRedo.Set(v)
	}
}

// Not deriva o State booleano inverso — o complemento natural de
// BindDisabled ("desabilitado quando NÃO pode").
func Not(s *State[bool]) *State[bool] {
	return Map(s, func(v bool) bool { return !v })
}
