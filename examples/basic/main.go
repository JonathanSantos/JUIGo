// Demo básica do JUIGo, mostrando a DX reativa:
//
//   - o campo é vinculado em duas vias a um State (BindValue) e suporta
//     seleção (arraste ou Shift+setas) e Ctrl/Cmd+A/C/X/V;
//   - o contador de caracteres deriva desse State com Map e atualiza ao
//     vivo enquanto você digita — sem callback nem Invalidate;
//   - o checkbox e o slider também são vinculados a States, com o texto de
//     volume derivado por Map;
//   - o botão ecoa o valor no título via um segundo State (BindText).
//
// Interações para experimentar: digitar com acentos, selecionar e colar
// texto, Tab/Shift+Tab para navegar o foco, setas no slider focado.
package main

import (
	"fmt"
	"log"

	"juigo"
)

func main() {
	valor := juigo.NewState("")
	eco := juigo.NewState("Digite algo e clique em Enviar")
	notif := juigo.NewState(true)
	volume := juigo.NewState(0.3)

	contador := juigo.Map(valor, func(s string) string {
		return fmt.Sprintf("%d caracteres", len([]rune(s)))
	})
	volTxt := juigo.Map(volume, func(v float64) string {
		return fmt.Sprintf("Volume: %.0f%%", v*100)
	})

	lista := juigo.NewVBox().Gap(4).Pad(0)
	for i := 1; i <= 25; i++ {
		lista.Add(juigo.NewText(fmt.Sprintf("Item %02d da lista rolável — use a roda do mouse", i)))
	}

	prioridade := juigo.NewState("Média")

	ui := juigo.NewVBox(
		juigo.NewText("").BindText(eco).Center(),
		juigo.NewHBox(
			juigo.Grow(juigo.NewInput("Digite aqui…").BindValue(valor), 1),
			juigo.Tooltip(juigo.NewButton("Enviar", func() {
				eco.Set("Você digitou: " + valor.Get())
			}), "Atualiza o título com o texto digitado"),
		),
		juigo.NewText("").BindText(contador).Right(),
		juigo.NewHBox(
			juigo.NewText("Prioridade:"),
			juigo.NewDropdown("Baixa", "Média", "Alta").BindValue(prioridade),
			juigo.NewText("").BindText(juigo.Map(prioridade, func(p string) string {
				return "→ " + p
			})),
		),
		juigo.NewCheckbox("Notificações").BindChecked(notif),
		juigo.NewSlider(0, 1).BindValue(volume),
		juigo.NewText("").BindText(volTxt).Right(),
		juigo.Grow(juigo.NewScroll(lista), 1),
	).Pad(16)

	if err := juigo.Run("JUIGo — demo básico", 480, 460, ui); err != nil {
		log.Fatal(err)
	}
}
