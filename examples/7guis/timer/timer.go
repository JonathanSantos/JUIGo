// Package timer é o 7GUIs nº 4: cronômetro com duração ajustável AO VIVO.
// O tempo decorrido é um State[float64] dirigido por anim.Tween linear
// (1 segundo de animação = 1 segundo de relógio); mexer no slider de
// duração re-mira o tween a partir do ponto atual, e Reset reinicia.
//
// Limitação registrada (ver ../GAPS.md): aplicações não têm acesso direto
// aos timers do loop (internal/hooks) — anim.Tween cobre este caso com
// elegância, mas um "App.Every/After" público faria falta em timers que não
// são interpolação.
package timer

import (
	"fmt"
	"time"

	"juigo"
	"juigo/anim"
)

// UI monta o cronômetro.
func UI() juigo.Widget {
	decorrido := juigo.NewState(0.0) // segundos
	duracao := juigo.NewState(15.0)  // segundos (slider 1..30)

	// liga (re)mira o tween: do ponto atual até a duração, no tempo REAL
	// restante — linear, então decorrido avança 1s por segundo.
	liga := func() {
		resta := duracao.Get() - decorrido.Get()
		if resta <= 0 {
			return
		}
		anim.Tween(decorrido, duracao.Get(), time.Duration(resta*float64(time.Second)), anim.Linear)
	}
	duracao.Watch(func(float64) { liga() })
	liga()

	fracao := juigo.Combine(func() float64 {
		if duracao.Get() <= 0 {
			return 1
		}
		return decorrido.Get() / duracao.Get()
	}, decorrido, duracao)

	return juigo.NewVBox(
		juigo.NewGrid(2,
			juigo.NewText("Decorrido:"), juigo.NewProgressBar(0, 1).BindValue(fracao),
			juigo.NewText(""), juigo.NewText("").BindText(juigo.Map(decorrido, func(v float64) string {
				return fmt.Sprintf("%.1fs", v)
			})),
			juigo.NewText("Duração:"), juigo.Grow(juigo.NewSlider(1, 30).BindValue(duracao), 1),
		),
		juigo.NewButton("Reset", func() {
			decorrido.Set(0)
			liga()
		}),
	).Pad(16)
}
