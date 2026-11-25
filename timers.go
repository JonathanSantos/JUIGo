package juigo

import (
	"time"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
)

// After agenda fn para rodar UMA vez na main thread, após d — o timer
// público das aplicações (o mesmo relógio dos widgets, virtual no uitest).
// Devolve a função de cancelamento. Sem aplicação ou harness em execução,
// nada é agendado e o cancelamento é inócuo.
func After(d time.Duration, fn func()) (cancel func()) {
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
func Every(d time.Duration, fn func()) (cancel func()) {
	stopped := false
	var current func()
	var tick func()
	tick = func() {
		if stopped {
			return
		}
		fn()
		if !stopped {
			current = hooks.ScheduleAfter(d, tick)
		}
	}
	current = hooks.ScheduleAfter(d, tick)
	return func() {
		stopped = true
		if current != nil {
			current()
		}
	}
}
