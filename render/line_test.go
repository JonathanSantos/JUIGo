package render

import (
	"image"
	"image/color"
	"testing"
)

var tinta = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}

// TestStrokeLineRetasNitidas: com coordenadas em centros de pixel, uma
// linha horizontal (e uma vertical) de espessura 1 acende exatamente a sua
// linha, cheia, sem vazar para as vizinhas.
func TestStrokeLineRetasNitidas(t *testing.T) {
	fundo := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	dst := solid(image.Rect(0, 0, 20, 20), fundo)
	StrokeLine(dst, image.Pt(3, 10), image.Pt(16, 10), 1, tinta)
	for x := 3; x <= 16; x++ {
		if got := dst.RGBAAt(x, 10); got != tinta {
			t.Fatalf("pixel (%d,10) deveria ser cheio; veio %v", x, got)
		}
		if got := dst.RGBAAt(x, 9); got != fundo {
			t.Fatalf("pixel (%d,9) não deveria ser tocado; veio %v", x, got)
		}
		if got := dst.RGBAAt(x, 11); got != fundo {
			t.Fatalf("pixel (%d,11) não deveria ser tocado; veio %v", x, got)
		}
	}

	dst2 := solid(image.Rect(0, 0, 20, 20), fundo)
	StrokeLine(dst2, image.Pt(10, 3), image.Pt(10, 16), 1, tinta)
	if got := dst2.RGBAAt(10, 8); got != tinta {
		t.Fatalf("vertical: (10,8) deveria ser cheio; veio %v", got)
	}
	if got := dst2.RGBAAt(11, 8); got != fundo {
		t.Fatalf("vertical: (11,8) não deveria ser tocado; veio %v", got)
	}
}

// TestStrokeLineDiagonalSuave: a diagonal de 45° passa pelos centros dos
// pixels do caminho (cheios) e deixa rampa de AA nos vizinhos.
func TestStrokeLineDiagonalSuave(t *testing.T) {
	fundo := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	dst := solid(image.Rect(0, 0, 20, 20), fundo)
	StrokeLine(dst, image.Pt(4, 4), image.Pt(15, 15), 1, tinta)

	if got := dst.RGBAAt(9, 9); got != tinta {
		t.Fatalf("(9,9) está no caminho; deveria ser cheio, veio %v", got)
	}
	// Vizinho lateral: parcialmente coberto (nem fundo, nem cheio).
	viz := dst.RGBAAt(9, 8)
	if viz == fundo || viz == tinta {
		t.Fatalf("(9,8) deveria ter rampa de AA; veio %v", viz)
	}
	// Longe da linha: intocado.
	if got := dst.RGBAAt(4, 15); got != fundo {
		t.Fatalf("(4,15) está longe; veio %v", got)
	}
}

// TestStrokePolylineCobreJuntas: os segmentos emendam sem buracos.
func TestStrokePolylineCobreJuntas(t *testing.T) {
	fundo := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	dst := solid(image.Rect(0, 0, 30, 30), fundo)
	StrokePolyline(dst, []image.Point{{4, 20}, {12, 8}, {20, 16}, {26, 6}}, 2, tinta)
	for _, p := range []image.Point{{4, 20}, {12, 8}, {20, 16}, {26, 6}} {
		if got := dst.RGBAAt(p.X, p.Y); got == fundo {
			t.Fatalf("vértice %v deveria estar pintado", p)
		}
	}
}

// TestStrokeLineSemAlocacao garante o contrato de zero alocações.
func TestStrokeLineSemAlocacao(t *testing.T) {
	dst := solid(image.Rect(0, 0, 128, 128), color.RGBA{A: 0xFF})
	pts := []image.Point{{4, 100}, {40, 20}, {80, 90}, {120, 10}}
	allocs := testing.AllocsPerRun(20, func() {
		StrokePolyline(dst, pts, 2, tinta)
	})
	if allocs != 0 {
		t.Fatalf("StrokePolyline alocou %v vezes por chamada; esperava 0", allocs)
	}
}
