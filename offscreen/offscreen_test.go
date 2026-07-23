package offscreen_test

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"juigo"
	"juigo/offscreen"
)

func demoUI() juigo.Widget {
	campo := juigo.NewInput("Digite…")
	campo.SetText("Olá, ação!")
	return juigo.NewVBox(
		juigo.NewText("Título").Center(),
		juigo.NewHBox(
			juigo.Grow(campo, 1),
			juigo.NewButton("Enviar", nil),
		),
		juigo.NewCheckbox("Opção"),
	).Pad(16)
}

func TestRenderDeterministico(t *testing.T) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}

	a := offscreen.Render(demoUI(), th, 480, 240)
	b := offscreen.Render(demoUI(), th, 480, 240)

	// Determinismo byte a byte — a base dos golden tests.
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("duas renderizações da mesma árvore deveriam ser idênticas")
	}

	// Algo além do fundo foi pintado.
	bg := th.Background
	painted := false
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != bg.R || a.Pix[i+1] != bg.G || a.Pix[i+2] != bg.B {
			painted = true
			break
		}
	}
	if !painted {
		t.Fatal("Render não pintou nenhum widget")
	}
}

func TestRenderNilSeguro(t *testing.T) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}
	if img := offscreen.Render(nil, th, 10, 10); img.Bounds().Dx() != 10 {
		t.Fatal("Render(nil, ...) deveria devolver buffer vazio do tamanho pedido")
	}
	if img := offscreen.Render(demoUI(), nil, 10, 10); img.Bounds().Dy() != 10 {
		t.Fatal("Render(..., nil) deveria devolver buffer vazio do tamanho pedido")
	}
}

func TestSavePNG(t *testing.T) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}
	img := offscreen.Render(demoUI(), th, 320, 160)

	path := filepath.Join(t.TempDir(), "captura.png")
	if err := offscreen.SavePNG(path, img); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("abrir PNG: %v", err)
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decodificar PNG: %v", err)
	}
	if decoded.Bounds() != img.Bounds() {
		t.Fatalf("PNG decodificado com bounds %v, esperado %v", decoded.Bounds(), img.Bounds())
	}
}
