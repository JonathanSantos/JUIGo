// Package anim anima States do JUIGo: um Tween interpola um State[float64]
// do valor atual até um alvo, com easing, sobre os timers da aplicação —
// funciona no App real (WaitEventsTimeout) e, IDENTICAMENTE, no relógio
// virtual do uitest (h.Advance), sem sleeps nos testes.
//
// Como tudo é State, animação compõe com os bindings existentes: anime o
// State de um ProgressBar, um offset de rolagem via Watch, ou qualquer valor
// derivado. Iniciar um Tween sobre um State que JÁ está animando retarget-a:
// a animação anterior para e a nova parte do valor atual (sem saltos).
//
//	pos := state.New(float64(rolagem.Offset()))
//	pos.Watch(func(v float64) { rolagem.ScrollTo(int(v)) })
//	anim.Tween(pos, 0, 400*time.Millisecond, nil) // nil = EaseInOut
//
// O passo é fixo (~60 quadros/s) e o progresso conta o tempo AGENDADO, não o
// relógio de parede: determinístico nos testes; no App, quadros atrasados
// desaceleram a animação em vez de saltar. Como todo o JUIGo,
// single-threaded.
package anim

import (
	"time"

	"juigo/internal/hooks"
	"juigo/state"
)

// frame é o passo da animação (~60 quadros por segundo).
const frame = 16 * time.Millisecond

// Easing mapeia o progresso temporal t ∈ [0,1] para o progresso do valor.
type Easing func(t float64) float64

// Linear avança em velocidade constante.
func Linear(t float64) float64 {
	return t
}

// EaseIn (cúbica) começa devagar e acelera.
func EaseIn(t float64) float64 {
	return t * t * t
}

// EaseOut (cúbica) começa rápido e desacelera — o padrão de rolagens.
func EaseOut(t float64) float64 {
	u := 1 - t
	return 1 - u*u*u
}

// EaseInOut (cúbica) acelera e desacelera — o padrão de transições.
func EaseInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := -2*t + 2
	return 1 - u*u*u/2
}

// Animation é uma animação em andamento, devolvida por Tween.
type Animation struct {
	// OnDone é chamado quando a animação COMPLETA (não é chamado por Stop).
	// Defina logo após o Tween — o primeiro quadro só ocorre no próximo
	// timer. Exceção: com duração <= 0 ou sem aplicação em execução, a
	// animação completa dentro do próprio Tween e OnDone não dispara;
	// verifique Running().
	OnDone func()

	target  *state.State[float64]
	cancel  func()
	running bool
	from    float64
	to      float64
	elapsed time.Duration
	dur     time.Duration
	ease    Easing
}

// active rastreia a animação corrente de cada State, para retarget.
var active = map[*state.State[float64]]*Animation{}

// Tween anima s do valor ATUAL até to na duração dada. ease nil usa
// EaseInOut. Se s já estiver animando, a animação anterior para (retarget).
// Com d <= 0 — ou sem aplicação/harness em execução — o valor salta direto
// para o alvo.
func Tween(s *state.State[float64], to float64, d time.Duration, ease Easing) *Animation {
	if prev := active[s]; prev != nil {
		prev.Stop()
	}
	if ease == nil {
		ease = EaseInOut
	}
	a := &Animation{target: s, from: s.Get(), to: to, dur: d, ease: ease, running: true}
	active[s] = a
	if d <= 0 || hooks.Schedule == nil {
		a.finish()
		return a
	}
	a.cancel = hooks.ScheduleAfter(frame, a.tick)
	return a
}

// Stop interrompe a animação onde está (sem OnDone). Idempotente.
func (a *Animation) Stop() {
	if !a.running {
		return
	}
	a.running = false
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	if active[a.target] == a {
		delete(active, a.target)
	}
}

// Running informa se a animação ainda está em andamento.
func (a *Animation) Running() bool {
	return a.running
}

// tick avança um quadro: interpola, agenda o próximo ou completa.
func (a *Animation) tick() {
	if !a.running {
		return
	}
	a.elapsed += frame
	if a.elapsed >= a.dur {
		a.finish()
		return
	}
	t := float64(a.elapsed) / float64(a.dur)
	a.target.Set(a.from + (a.to-a.from)*a.ease(t))
	a.cancel = hooks.ScheduleAfter(frame, a.tick)
}

// finish leva o valor exatamente ao alvo e dispara OnDone.
func (a *Animation) finish() {
	a.running = false
	if active[a.target] == a {
		delete(active, a.target)
	}
	a.target.Set(a.to)
	if a.OnDone != nil {
		a.OnDone()
	}
}
