// Package tempconv é o 7GUIs nº 2: conversor de temperatura. Dois campos
// sincronizados em DUAS vias (Celsius ↔ Fahrenheit): editar um recalcula o
// outro; entrada inválida não propaga. O guarda `sincronizando` quebra o
// ciclo de eco — o mesmo padrão dos bindings da lib.
package tempconv

import (
	"strconv"

	"github.com/JonathanSantos/JUIGo"
)

// formata imprime a temperatura sem zeros supérfluos.
func formata(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// UI monta o conversor.
func UI() juigo.Widget {
	celsius := juigo.NewState("")
	fahrenheit := juigo.NewState("")

	sincronizando := false
	converte := func(destino *juigo.State[string], f func(float64) float64) func(string) {
		return func(s string) {
			if sincronizando {
				return
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return // inválido: não propaga (7GUIs)
			}
			sincronizando = true
			destino.Set(formata(f(v)))
			sincronizando = false
		}
	}
	celsius.Watch(converte(fahrenheit, func(c float64) float64 { return c*9/5 + 32 }))
	fahrenheit.Watch(converte(celsius, func(f float64) float64 { return (f - 32) * 5 / 9 }))

	return juigo.NewVBox(
		juigo.NewHBox(
			juigo.Grow(juigo.NewInput("Celsius").BindValue(celsius), 1),
			juigo.Centered(juigo.NewText("Celsius =")),
			juigo.Grow(juigo.NewInput("Fahrenheit").BindValue(fahrenheit), 1),
			juigo.Centered(juigo.NewText("Fahrenheit")),
		),
	).Pad(16)
}
