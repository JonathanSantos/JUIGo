// Demo básica do JUIGo: uma janela com título (Text), um campo de texto
// (Input) e um botão (Button) que, ao ser clicado — ou acionado com Enter/
// Espaço quando focado —, atualiza o título com o conteúdo do campo.
//
// Interações para experimentar:
//   - digitar com acentos (ã, é, ç…) e mover o cursor com setas/Home/End;
//   - Tab para alternar o foco entre o Input e o Button;
//   - pressionar o botão, arrastar o cursor para fora e soltar (não dispara).
package main

import (
	"log"

	"juigo"
)

func main() {
	app, err := juigo.New("JUIGo — demo básico", 480, 220)
	if err != nil {
		log.Fatal(err)
	}
	th := app.Theme()

	title := juigo.NewText(th, "Digite algo e clique em Enviar")
	title.Align = juigo.AlignCenter

	input := juigo.NewInput(th, "Digite aqui…")

	btn := juigo.NewButton(th, "Enviar", nil)
	btn.OnClick = func() {
		title.SetText("Você digitou: " + input.Text())
	}

	root := juigo.NewVBox(th.SpacingPx(), 2*th.PaddingPx())
	root.Add(title, input, btn)
	app.SetRoot(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
