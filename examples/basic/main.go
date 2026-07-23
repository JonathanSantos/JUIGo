// Demo do JUIGo: um formulário reativo completo.
//
//   - Input com seleção/clipboard/rolagem vinculado a State (BindValue);
//   - contador e volume derivados com Map, atualizando ao vivo;
//   - Dropdown, Checkbox, Slider→ProgressBar e Radios (grupo = State
//     compartilhado) todos reativos;
//   - TextArea multilinha (Enter quebra linha, setas navegam, Ctrl/Cmd+C/V
//     preservam quebras);
//   - botão "Sobre…" abre um Modal (Escape ou clique fora fecham);
//   - lista rolável no rodapé; Tab/Shift+Tab navegam o foco; tooltips ao
//     pausar o mouse.
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
	prioridade := juigo.NewState("Média")
	plano := juigo.NewState("free")

	contador := juigo.Map(valor, func(s string) string {
		return fmt.Sprintf("%d caracteres", len([]rune(s)))
	})
	volTxt := juigo.Map(volume, func(v float64) string {
		return fmt.Sprintf("Volume: %.0f%%", v*100)
	})

	sobre := juigo.NewModal(juigo.NewVBox(
		juigo.NewText("JUIGo — biblioteca de GUI em Go").Center(),
		juigo.NewText("Renderização por software, reatividade com State,"),
		juigo.NewText("HiDPI, overlay e tema centralizado."),
	))

	lista := juigo.NewVBox().Gap(4).Pad(0)
	for i := 1; i <= 25; i++ {
		lista.Add(juigo.NewText(fmt.Sprintf("Item %02d da lista rolável — use a roda do mouse", i)))
	}

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
			juigo.NewSpacer(),
			juigo.NewRadio("Grátis", "free").BindValue(plano),
			juigo.NewRadio("Pro", "pro").BindValue(plano),
		),
		juigo.NewCheckbox("Notificações").BindChecked(notif),
		juigo.NewSlider(0, 1).BindValue(volume),
		juigo.NewProgressBar(0, 1).BindValue(volume),
		juigo.NewText("").BindText(volTxt).Right(),
		juigo.NewTextArea("Observações (Enter quebra linha)…"),
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.Tooltip(juigo.NewButton("Sobre…", sobre.Show), "Abre o diálogo modal"),
		),
		juigo.Grow(juigo.NewScroll(lista), 1),
	).Pad(16)

	if err := juigo.Run("JUIGo — demo", 520, 640, ui); err != nil {
		log.Fatal(err)
	}
}
