// O launcher dos 7GUIs: uma tela inicial com um botão por GUI; abrir troca
// a raiz da aplicação (App.SetRoot) e "← Voltar" restaura o menu. Cada GUI
// vive no próprio pacote, com testes de uitest — rode `go test ./...` para
// vê-los.
package main

import (
	"log"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/7guis/cells"
	"github.com/JonathanSantos/JUIGo/examples/7guis/circles"
	"github.com/JonathanSantos/JUIGo/examples/7guis/counter"
	"github.com/JonathanSantos/JUIGo/examples/7guis/crud"
	"github.com/JonathanSantos/JUIGo/examples/7guis/flightbooker"
	"github.com/JonathanSantos/JUIGo/examples/7guis/tempconv"
	"github.com/JonathanSantos/JUIGo/examples/7guis/timer"
)

func main() {
	app, err := juigo.New("7GUIs em JUIGo", 680, 560)
	if err != nil {
		log.Fatal(err)
	}

	var inicio juigo.Widget
	abre := func(nome string, monta func() juigo.Widget) *juigo.Button {
		return juigo.NewButton(nome, func() {
			app.SetRoot(juigo.NewVBox(
				juigo.NewHBox(
					juigo.NewButton("← Voltar", func() { app.SetRoot(inicio) }),
					juigo.Grow(juigo.NewText(nome).Center(), 1),
				),
				juigo.Grow(monta(), 1),
			).Pad(8))
		})
	}

	inicio = juigo.NewVBox(
		juigo.NewText("7GUIs — sete GUIs clássicas em JUIGo").Center(),
		abre("1 · Counter", counter.UI),
		abre("2 · Temperature Converter", tempconv.UI),
		abre("3 · Flight Booker", flightbooker.UI),
		abre("4 · Timer", timer.UI),
		abre("5 · CRUD", crud.UI),
		abre("6 · Circle Drawer", circles.UI),
		abre("7 · Cells", cells.UI),
	).Pad(24).Gap(8)

	app.SetRoot(inicio)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
