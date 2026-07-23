// Package quick é a camada de conveniência do JUIGo: monta em poucas linhas
// os rituais que mais aparecem em aplicações — formulários validados,
// diálogos (confirmação, alerta, pergunta) e linhas rótulo+campo — compondo
// os widgets normais de juigo/widget e a validação de juigo/form.
//
// A regra da rampa: cada helper ACEITA widgets comuns onde tem conteúdo e
// DEVOLVE widgets comuns. Quando o padrão pronto deixar de servir, desça um
// nível naquele ponto (monte o Grid, o Modal ou o form à mão) sem reescrever
// o resto da tela — não há um segundo dialeto para abandonar.
//
//	quick.Form(
//	    quick.Text("Nome:", nome).Required("Informe o nome"),
//	    quick.Text("E-mail:", mail).Email("E-mail inválido"),
//	).Submit("Salvar", salvar)
package quick

import "github.com/JonathanSantos/JUIGo/widget"

// Labeled monta a linha rótulo+campo: o rótulo à esquerda, na largura
// preferida e centralizado na vertical, e o campo crescendo até a borda.
func Labeled(label string, w widget.Widget) *widget.HBox {
	return widget.NewHBox(
		widget.Centered(widget.NewText(label)),
		widget.Grow(w, 1),
	)
}

// Buttons monta a barra de ações alinhada à direita — o rodapé clássico de
// formulários e diálogos (um Spacer empurra as ações para o fim).
func Buttons(actions ...widget.Widget) *widget.HBox {
	children := make([]widget.Widget, 0, len(actions)+1)
	children = append(children, widget.NewSpacer())
	children = append(children, actions...)
	return widget.NewHBox(children...)
}
