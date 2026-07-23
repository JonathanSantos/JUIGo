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
	"time"

	"juigo"
)

func main() {
	app, err := juigo.New("JUIGo — demo", 520, 640)
	if err != nil {
		log.Fatal(err)
	}

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

	// Enviar: desabilitado enquanto o campo está vazio (reativo) e com
	// loading durante um "trabalho de rede" — a goroutine entrega o
	// resultado de volta à main thread com app.Post.
	var enviar *juigo.Button
	enviar = juigo.BindDisabled(juigo.Tooltip(juigo.NewButton("Enviar", func() {
		enviar.SetLoading(true)
		texto := valor.Get()
		go func() {
			time.Sleep(1200 * time.Millisecond) // simula rede
			app.Post(func() {
				enviar.SetLoading(false)
				eco.Set("Você digitou: " + texto)
			})
		}()
	}), "Atualiza o título com o texto digitado"),
		juigo.Map(valor, func(s string) bool { return s == "" }))

	lista := juigo.NewVBox().Gap(4).Pad(0)
	for i := 1; i <= 25; i++ {
		lista.Add(juigo.NewText(fmt.Sprintf("Item %02d da lista rolável — use a roda do mouse", i)))
	}

	ui := juigo.NewVBox(
		juigo.NewText("").BindText(eco).Center(),
		juigo.NewHBox(
			juigo.Grow(juigo.NewInput("Digite aqui…").BindValue(valor), 1),
			enviar,
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

	app.SetRoot(ui)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
