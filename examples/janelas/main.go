// Janelas — o exemplo MULTI-JANELA do JUIGo. A janela principal abre
// janelas adicionais com App.NewWindow: cada uma tem tema e Session
// próprios (a escura prova o tema por janela), mas todas compartilham os
// MESMOS States — mexer no slider de uma atualiza todas ao vivo, e o dano
// fica restrito à janela de cada widget (dano com identidade de janela).
// Fechar uma janela dispara OnClose (aqui, um toast); a aplicação termina
// quando todas fecham.
package main

import (
	"fmt"
	"log"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
)

func main() {
	app, err := juigo.New("Janelas — multi-janela em JUIGo", 460, 300)
	if err != nil {
		log.Fatal(err)
	}

	// Estado COMPARTILHADO entre todas as janelas.
	volume := juigo.NewState(0.4)
	rotulo := juigo.Map(volume, func(v float64) string {
		return fmt.Sprintf("volume: %.0f%%", v*100)
	})

	// painel monta a mesma vista reativa usada em todas as janelas.
	painel := func(titulo string) juigo.Widget {
		return juigo.NewVBox(
			juigo.NewText(titulo),
			juigo.NewSlider(0, 1).BindValue(volume),
			juigo.NewText("").BindText(rotulo),
		).Pad(16).Gap(12)
	}

	contador := 0
	abrir := func(escura bool) {
		contador++
		titulo := fmt.Sprintf("Janela %d", contador)
		w, err := app.NewWindow(titulo, 360, 220)
		if err != nil {
			log.Println("janelas:", err)
			return
		}
		if escura {
			if th, err := juigo.DarkTheme(); err == nil {
				if err := w.SetTheme(th); err != nil {
					log.Println("janelas:", err)
				}
			}
		}
		w.SetRoot(painel(titulo + " — o mesmo State"))
		w.OnClose(func() { quick.Toast(titulo + " fechada") })
	}

	app.SetRoot(juigo.NewVBox(
		juigo.NewText("O slider é o MESMO State em todas as janelas."),
		painel("Principal"),
		juigo.NewHBox(
			juigo.NewButton("Nova janela", func() { abrir(false) }),
			juigo.NewButton("Nova janela escura", func() { abrir(true) }),
		).Gap(8),
	).Pad(16).Gap(8))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
