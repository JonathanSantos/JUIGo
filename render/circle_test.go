package render

import (
	"bytes"
	"image"
	"testing"
)

// TestFillCircleForma: miolo e cardeais sólidos, cantos do quadrado
// circundante intocados, e simetria exata nos dois eixos.
func TestFillCircleForma(t *testing.T) {
	tela := novaTela(image.Rect(0, 0, 48, 48))
	fundo := novaTela(image.Rect(0, 0, 48, 48))
	c := image.Pt(24, 24)
	const r = 10
	FillCircle(tela, c, r, corTeste)

	for _, p := range []image.Point{{24, 24}, {24, 24 - r + 1}, {24, 24 + r - 2}, {24 - r + 1, 24}} {
		if got := tela.RGBAAt(p.X, p.Y); got != corTeste {
			t.Errorf("pixel %v deveria ser sólido; veio %v", p, got)
		}
	}
	for _, p := range []image.Point{{24 - r, 24 - r}, {24 + r, 24 + r}, {24 - r - 2, 24}} {
		if got, want := tela.RGBAAt(p.X, p.Y), fundo.RGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v deveria ficar intocado; veio %v", p, got)
		}
	}

	uniforme := image.NewRGBA(image.Rect(0, 0, 48, 48))
	FillCircle(uniforme, c, r, corTeste)
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			ex, ey := 2*c.X-x-1, 2*c.Y-y-1
			if a, b := uniforme.RGBAAt(x, y), uniforme.RGBAAt(ex, y); a != b {
				t.Fatalf("assimetria horizontal em (%d,%d): %v != %v", x, y, a, b)
			}
			if a, b := uniforme.RGBAAt(x, y), uniforme.RGBAAt(x, ey); a != b {
				t.Fatalf("assimetria vertical em (%d,%d): %v != %v", x, y, a, b)
			}
		}
	}
}

// TestStrokeCircleAnel: o anel pinta os cardeais, poupa o miolo e espessura
// maior ou igual ao raio equivale ao círculo cheio.
func TestStrokeCircleAnel(t *testing.T) {
	tela := novaTela(image.Rect(0, 0, 48, 48))
	fundo := novaTela(image.Rect(0, 0, 48, 48))
	c := image.Pt(24, 24)
	StrokeCircle(tela, c, 10, 2, corTeste)

	for _, p := range []image.Point{{24, 24 - 9}, {24, 24 + 8}, {24 - 9, 24}, {24 + 8, 24}} {
		if got := tela.RGBAAt(p.X, p.Y); got != corTeste {
			t.Errorf("anel em %v deveria ser sólido; veio %v", p, got)
		}
	}
	for _, p := range []image.Point{{24, 24}, {24 + 3, 24 - 3}, {24 - 10, 24 - 10}} {
		if got, want := tela.RGBAAt(p.X, p.Y), fundo.RGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v deveria ficar intocado; veio %v", p, got)
		}
	}

	cheio := novaTela(image.Rect(0, 0, 48, 48))
	anelGrosso := novaTela(image.Rect(0, 0, 48, 48))
	FillCircle(cheio, c, 8, corTeste)
	StrokeCircle(anelGrosso, c, 8, 8, corTeste)
	if !bytes.Equal(cheio.Pix, anelGrosso.Pix) {
		t.Error("espessura igual ao raio deveria equivaler ao círculo cheio")
	}
}

// TestCirculoRecorteEAlocacao: desenhar nas bordas do buffer recorta sem
// pânico e as primitivas não alocam.
func TestCirculoRecorteEAlocacao(t *testing.T) {
	tela := image.NewRGBA(image.Rect(0, 0, 32, 32))
	FillCircle(tela, image.Pt(0, 0), 8, corTeste)
	FillCircle(tela, image.Pt(31, 31), 8, corTeste)
	StrokeCircle(tela, image.Pt(0, 31), 8, 2, corTeste)

	if n := testing.AllocsPerRun(50, func() {
		FillCircle(tela, image.Pt(16, 16), 10, corTeste)
		StrokeCircle(tela, image.Pt(16, 16), 12, 2, corTeste)
	}); n != 0 {
		t.Errorf("círculos alocaram %.0f vez(es) por chamada", n)
	}
}
