// Demo básica do JUIGo.
//
// Passo 3: Button visual estático posicionado em um Container absoluto.
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

	btn := juigo.NewButton(th, "Enviar", nil)
	pref := btn.PreferredSize()
	btn.Layout(image.Rect(40, 40, 40+pref.X, 40+pref.Y))

	root := juigo.NewContainer()
	root.Add(btn)
	app.SetRoot(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
