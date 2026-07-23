package juigo

import (
	"time"

	"juigo/internal/hooks"
)

// After agenda fn para rodar UMA vez na main thread, após d — o timer
// público das aplicações (o mesmo relógio dos widgets, virtual no uitest).
// Devolve a função de cancelamento. Sem aplicação ou harness em execução,
// nada é agendado e o cancelamento é inócuo.
func After(d time.Duration, fn func()) (cancelar func()) {
	c := hooks.ScheduleAfter(d, fn)
	return func() {
		if c != nil {
			c()
		}
	}
}

// Every agenda fn para rodar na main thread a cada d, até ser cancelada.
// Primeira execução após d (não imediata). Devolve a função de
// cancelamento; sem aplicação em execução, nada é agendado.
func Every(d time.Duration, fn func()) (cancelar func()) {
	parado := false
	var atual func()
	var tique func()
	tique = func() {
		if parado {
			return
		}
		fn()
		if !parado {
			atual = hooks.ScheduleAfter(d, tique)
		}
	}
	atual = hooks.ScheduleAfter(d, tique)
	return func() {
		parado = true
		if atual != nil {
			atual()
		}
	}
}
