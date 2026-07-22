// Demo básica do JUIGo.
//
// Passo 5: Button interativo (hover/pressed/OnClick) atualiza o título.
package main

import (
	"fmt"
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

	clicks := 0
	btn := juigo.NewButton(th, "Enviar", func() {
		clicks++
		title.SetText(fmt.Sprintf("Cliques: %d", clicks))
	})
	pref := btn.PreferredSize()
	btn.Layout(image.Rect(40, 56, 40+pref.X, 56+pref.Y))

	root := juigo.NewContainer()
	root.Add(title, btn)
	app.SetRoot(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
