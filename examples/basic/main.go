// Demo básica do JUIGo.
//
// Passo 1: janela GLFW com o buffer RGBA preenchido com cor sólida,
// apresentado via textura OpenGL em um quad fullscreen.
package main

import (
	"log"

	"juigo"
)

func main() {
	app, err := juigo.New("JUIGo — demo básico", 480, 240)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
