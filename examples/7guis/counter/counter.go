// Package counter é o 7GUIs nº 1: um contador. O caso mínimo de
// reatividade — um State[int] exibido por um Text derivado e incrementado
// por um Button.
package counter

import (
	"strconv"

	"juigo"
)

// UI monta a tela do contador.
func UI() juigo.Widget {
	n := juigo.NewState(0)
	return juigo.NewVBox(
		juigo.NewHBox(
			juigo.Grow(juigo.NewText("").BindText(juigo.Map(n, strconv.Itoa)).Center(), 1),
			juigo.NewButton("Contar", func() { n.Set(n.Get() + 1) }),
		),
	).Pad(16)
}
