// Demo básica do JUIGo: uma janela com título (Text), um campo de texto
// (Input) e um botão (Button) que, ao ser acionado, atualiza o título com o
// conteúdo do campo.
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
	title := juigo.NewText("Digite algo e clique em Enviar").Center()
	campo := juigo.NewInput("Digite aqui…")

	ui := juigo.NewVBox(
		title,
		campo,
		juigo.NewButton("Enviar", func() {
			title.SetText("Você digitou: " + campo.Text())
		}),
	).Pad(16)

	if err := juigo.Run("JUIGo — demo básico", 480, 220, ui); err != nil {
		log.Fatal(err)
	}
}
