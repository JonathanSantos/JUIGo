// Demo básica do JUIGo.
//
// Passo 6: Input com foco, cursor e edição por runes; Button ecoa o texto.
package main

import (
	"image"
	"log"

	"juigo"
)

func main() {
	app, err := juigo.New("JUIGo — demo básico", 480, 240)
	if err != nil {
		log.Fatal(err)
	}
	th := app.Theme()

	title := juigo.NewText(th, "JUIGo — árvore de widgets com acentuação")
	title.Align = juigo.AlignCenter
	title.Layout(image.Rect(0, 16, 480, 16+th.LineHeight()))

	input := juigo.NewInput(th, "Digite aqui…")
	ip := input.PreferredSize()
	input.Layout(image.Rect(40, 56, 40+ip.X, 56+ip.Y))

	btn := juigo.NewButton(th, "Enviar", nil)
	bp := btn.PreferredSize()
	btn.Layout(image.Rect(40, 56+ip.Y+8, 40+bp.X, 56+ip.Y+8+bp.Y))
	btn.OnClick = func() {
		title.SetText("Você digitou: " + input.Text())
	}

	root := juigo.NewContainer()
	root.Add(title, input, btn)
	app.SetRoot(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
