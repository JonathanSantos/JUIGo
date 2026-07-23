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
	h := uitest.New(t, v.Raiz, 640, 360)
	alvo := cartaoPorTitulo(t, h, "Revisar o texto")
	h.MoveTo(centro(alvo))
	h.Session().PointerDown(centro(alvo), juigo.MouseButtonLeft)
	h.Session().PointerMove(centro(v.colunas[1]).Add(image.Pt(0, 20)))
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
