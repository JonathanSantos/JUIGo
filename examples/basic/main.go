// Demo básica do JUIGo.
//
// Passo 4: título (Text) e Button estático em um Container absoluto.
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

	btn := juigo.NewButton(th, "Enviar", nil)
	pref := btn.PreferredSize()
	btn.Layout(image.Rect(40, 56, 40+pref.X, 56+pref.Y))

	root := juigo.NewContainer()
	root.Add(title, btn)
	app.SetRoot(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
