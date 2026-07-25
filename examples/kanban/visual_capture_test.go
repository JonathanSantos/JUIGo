package main

import (
	"image"
	"os"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCapturaVisual salva um frame do quadro em pleno arrasto quando a
// variável KANBAN_CAPTURA aponta um caminho — só para inspeção manual.
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("KANBAN_CAPTURA")
	if caminho == "" {
		t.Skip("defina KANBAN_CAPTURA para salvar o frame")
	}
	v := nova(exemplo())
	// Captura em ESCALA 2 (retina): o PNG sai nítido como o App real.
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatal(err)
	}
	if err := th.SetScale(2); err != nil {
		t.Fatal(err)
	}
	h := uitest.NewWithTheme(t, v.Raiz, th, 1280, 720)
	alvo := cartaoPorTitulo(t, h, "Revisar o texto")
	h.MoveTo(centro(alvo))
	h.Session().PointerDown(centro(alvo), juigo.MouseButtonLeft)
	h.Session().PointerMove(centro(v.colunas[1]).Add(image.Pt(0, 20)))
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
