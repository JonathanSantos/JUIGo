package quick

import (
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/widget"
)

// Os diálogos abrem imediatamente (Show) e devolvem o Modal — feche-o com
// Close ou deixe o usuário decidir. São de uso único: para reabrir, chame o
// helper de novo. Precisam dos hooks de overlay ativos (App ou uitest).

// Confirm abre um diálogo de confirmação com os botões Cancel e OK e chama
// onResult exatamente uma vez quando ele fechar: true no OK; false no
// Cancel, no Escape ou no clique fora.
func Confirm(title string, onResult func(ok bool)) *widget.Modal {
	ok := false
	var m *widget.Modal
	m = widget.NewModal(widget.NewVBox(
		widget.NewText(title),
		Buttons(
			widget.NewButton("Cancel", func() { m.Close() }),
			widget.NewButton("OK", func() { ok = true; m.Close() }),
		),
	)).OnClose(func() {
		if onResult != nil {
			onResult(ok)
		}
	})
	m.Show()
	return m
}

// Alert abre um aviso com a mensagem e um botão OK. O botão abre focado:
// Enter dispensa o alerta (Escape também funciona).
func Alert(message string) *widget.Modal {
	var m *widget.Modal
	m = widget.NewModal(widget.NewVBox(
		widget.NewText(message),
		Buttons(widget.NewButton("OK", func() { m.Close() })),
	))
	m.Show()
	return m
}

// Prompt abre uma pergunta com um campo de texto — já focado, pronto para
// digitar — e chama onSubmit com o valor no OK ou no Enter; Cancel, Escape e
// clique fora descartam sem chamar.
func Prompt(title, placeholder string, onSubmit func(value string)) *widget.Modal {
	value := state.New("")
	accepted := false
	var m *widget.Modal
	accept := func() { accepted = true; m.Close() }
	m = widget.NewModal(widget.NewVBox(
		widget.NewText(title),
		widget.NewInput(placeholder).BindValue(value).OnSubmit(accept),
		Buttons(
			widget.NewButton("Cancel", func() { m.Close() }),
			widget.NewButton("OK", accept),
		),
	)).OnClose(func() {
		if accepted && onSubmit != nil {
			onSubmit(value.Get())
		}
	})
	m.Show()
	return m
}
