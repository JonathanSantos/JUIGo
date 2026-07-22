// Demo básica do JUIGo, mostrando a DX reativa:
//
//   - o campo é vinculado em duas vias a um State (BindValue);
//   - o contador de caracteres deriva desse State com Map e atualiza ao
//     vivo enquanto você digita — sem callback nem Invalidate;
//   - o botão ecoa o valor no título via um segundo State (BindText).
//
// Interações para experimentar: digitar com acentos, setas/Home/End, Tab
// para alternar o foco, Enter/Espaço no botão focado.
package main

import (
	"fmt"
	"log"

	"juigo"
)

func main() {
	valor := juigo.NewState("")
	eco := juigo.NewState("Digite algo e clique em Enviar")

	contador := juigo.Map(valor, func(s string) string {
		return fmt.Sprintf("%d caracteres", len([]rune(s)))
	})

	ui := juigo.NewVBox(
		juigo.NewText("").BindText(eco).Center(),
		juigo.NewInput("Digite aqui…").BindValue(valor),
		juigo.NewText("").BindText(contador).Right(),
		juigo.NewButton("Enviar", func() {
			eco.Set("Você digitou: " + valor.Get())
		}),
	).Pad(16)

	if err := juigo.Run("JUIGo — demo básico", 480, 240, ui); err != nil {
		log.Fatal(err)
	}
}
