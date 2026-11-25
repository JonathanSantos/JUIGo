package juigo

import "time"

// timerQueue é a fila de agendamentos do loop (hooks.Schedule): lógica pura,
// sem GLFW nem relógio próprio — o App informa o instante e executa o que
// vencer. Poucos timers vivem ao mesmo tempo (piscada, tooltip, animações),
// então a busca linear basta.
type timerQueue struct {
	items []queuedTimer
	seq   int
}

// queuedTimer é um agendamento pendente.
type queuedTimer struct {
	id int
	at time.Time
	fn func()
}

// add agenda fn para o instante at e devolve o identificador do timer.
func (q *timerQueue) add(at time.Time, fn func()) int {
	q.seq++
	q.items = append(q.items, queuedTimer{id: q.seq, at: at, fn: fn})
	return q.seq
}

// cancel remove o timer id, se ainda estiver pendente.
func (q *timerQueue) cancel(id int) {
	for i := range q.items {
		if q.items[i].id == id {
			q.removeAt(i)
			return
		}
	}
}

// removeAt tira o item i preservando a ordem e solta a closure da sobra.
func (q *timerQueue) removeAt(i int) {
	copy(q.items[i:], q.items[i+1:])
	last := len(q.items) - 1
	q.items[last] = queuedTimer{}
	q.items = q.items[:last]
}

// next devolve o vencimento mais próximo; ok é false com a fila vazia.
func (q *timerQueue) next() (at time.Time, ok bool) {
	if len(q.items) == 0 {
		return time.Time{}, false
	}
	at = q.items[0].at
	for _, t := range q.items[1:] {
		if t.at.Before(at) {
			at = t.at
		}
	}
	return at, true
}

// runDue executa, UM A UM e em ordem de vencimento, os timers vencidos até
// now. Cada timer sai da fila antes de rodar, então um callback pode
// cancelar outro timer do mesmo lote (ele não dispara mais) e agendar novos
// — os novos só rodam nesta chamada se também vencerem até now, o que um
// reagendamento com intervalo positivo nunca faz (sem laço infinito).
func (q *timerQueue) runDue(now time.Time) {
	for {
		best := -1
		for i := range q.items {
			if q.items[i].at.After(now) {
				continue
			}
			if best < 0 || q.items[i].at.Before(q.items[best].at) {
				best = i
			}
		}
		if best < 0 {
			return
		}
		fn := q.items[best].fn
		q.removeAt(best)
		fn()
	}
}
