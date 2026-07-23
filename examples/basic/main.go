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
	"juigo/anim"
	"juigo/form"
)

func main() {
	app, err := juigo.New("JUIGo — demo", 520, 640)
	if err != nil {
		log.Fatal(err)
	}

	temaEscuro, err := juigo.DarkTheme()
	if err != nil {
		log.Fatal(err)
	}
	temaClaro := app.Theme()

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

	// Modal de cadastro com validação declarativa (juigo/form): erros
	// aparecem no blur ou no Submit; Salvar só passa com tudo válido.
	nome := juigo.NewState("")
	mail := juigo.NewState("")
	cadastro := form.New(
		form.Field(nome, form.Required("Informe o nome"), form.MinRunes(3, "Mínimo de 3 caracteres")),
		form.Field(mail, form.Required("Informe o e-mail"), form.Email("E-mail inválido")),
	)
	erroDe := func(chave any) *juigo.Text {
		t := juigo.NewText("").BindText(cadastro.ErrorOf(chave))
		t.Danger()
		return t
	}
	campoNome := juigo.NewInput("Nome").BindValue(nome)
	campoNome.OnBlur(func() { cadastro.Touch(nome) })
	campoMail := juigo.NewInput("E-mail").BindValue(mail)
	campoMail.OnBlur(func() { cadastro.Touch(mail) })

	var sobre *juigo.Modal
	sobre = juigo.NewModal(juigo.NewVBox(
		juigo.NewText("Cadastro").Center(),
		juigo.NewGrid(2,
			juigo.NewText("Nome:"), juigo.Grow(campoNome, 1),
			juigo.NewText(""), erroDe(nome),
			juigo.NewText("E-mail:"), juigo.Grow(campoMail, 1),
			juigo.NewText(""), erroDe(mail),
		),
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.NewButton("Salvar", func() {
				cadastro.Submit(func() {
					sobre.Close()
					eco.Set("Cadastrado: " + nome.Get())
				})
			}),
		),
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

	// Tema claro ↔ escuro ao vivo: o Watch troca o tema da aplicação e o
	// mount re-propaga a nova paleta pela árvore no próximo frame.
	escuro := juigo.NewState(false)
	escuro.Watch(func(v bool) {
		t := temaClaro
		if v {
			t = temaEscuro
		}
		if err := app.SetTheme(t); err != nil {
			log.Print(err)
		}
	})

	// Lista VIRTUALIZADA: 500 itens, mas só as linhas visíveis existem como
	// widgets (pool reciclado na rolagem).
	lista := juigo.NewList(500,
		func() *juigo.Text { return juigo.NewText("") },
		func(row *juigo.Text, i int) {
			row.SetText(fmt.Sprintf("Item %03d da lista virtualizada — role com a roda", i+1))
		},
	)

	// Rolagem suave: o botão anima o offset do Scroll até o topo (o Tween
	// dirige um State que o Watch aplica ao ScrollTo).
	rolagem := juigo.NewScroll(lista)
	aoTopo := juigo.Tooltip(juigo.NewButton("↑ Topo", func() {
		pos := juigo.NewState(float64(rolagem.Offset()))
		pos.Watch(func(v float64) { rolagem.ScrollTo(int(v)) })
		anim.Tween(pos, 0, 400*time.Millisecond, anim.EaseOut)
	}), "Rola a lista suavemente até o topo")

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
		juigo.NewHBox(
			juigo.NewCheckbox("Notificações").BindChecked(notif),
			juigo.NewSpacer(),
			juigo.NewCheckbox("Tema escuro").BindChecked(escuro),
		),
		juigo.NewSlider(0, 1).BindValue(volume),
		juigo.NewProgressBar(0, 1).BindValue(volume),
		juigo.NewText("").BindText(volTxt).Right(),
		juigo.NewTextArea("Observações (Enter quebra linha)…"),
		juigo.NewHBox(
			juigo.NewSpacer(),
			aoTopo,
			juigo.Tooltip(juigo.NewButton("Cadastrar…", sobre.Show), "Abre o formulário validado"),
		),
		juigo.Grow(rolagem, 1),
	).Pad(16)

	app.SetRoot(ui)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
