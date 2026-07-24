package render

import (
	"image"
	"image/color"
	"testing"
)

// solid devolve um buffer com os bounds dados inteiramente na cor c.
func solid(b image.Rectangle, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(b)
	FillRect(img, b, c)
	return img
}

// TestCrossFadeExtremos verifica que t=0 copia a e t=1 copia b, byte a byte.
func TestCrossFadeExtremos(t *testing.T) {
	b := image.Rect(0, 0, 8, 8)
	a := solid(b, color.RGBA{200, 40, 10, 255})
	bb := solid(b, color.RGBA{20, 180, 240, 255})

	dst := image.NewRGBA(b)
	CrossFade(dst, a, bb, b, 0)
	for i := range dst.Pix {
		if dst.Pix[i] != a.Pix[i] {
			t.Fatalf("t=0: byte %d = %d, esperava %d (cópia de a)", i, dst.Pix[i], a.Pix[i])
		}
	}
	CrossFade(dst, a, bb, b, 1)
	for i := range dst.Pix {
		if dst.Pix[i] != bb.Pix[i] {
			t.Fatalf("t=1: byte %d = %d, esperava %d (cópia de b)", i, dst.Pix[i], bb.Pix[i])
		}
	}
}

// TestCrossFadeMeio verifica a mistura no ponto médio: cada canal fica no
// meio do caminho entre a e b (tolerância de 1 pelo arredondamento inteiro).
func TestCrossFadeMeio(t *testing.T) {
	b := image.Rect(0, 0, 4, 4)
	a := solid(b, color.RGBA{200, 40, 10, 255})
	bb := solid(b, color.RGBA{20, 180, 240, 255})

	dst := image.NewRGBA(b)
	CrossFade(dst, a, bb, b, 0.5)
	esperado := [4]int{110, 110, 125, 255}
	for ch := 0; ch < 4; ch++ {
		got := int(dst.Pix[ch])
		if diff := got - esperado[ch]; diff < -1 || diff > 1 {
			t.Fatalf("canal %d = %d, esperava ~%d", ch, got, esperado[ch])
		}
	}
}

// TestCrossFadeRecorte verifica que a mistura fica contida na região pedida
// e na interseção com os bounds das fontes.
func TestCrossFadeRecorte(t *testing.T) {
	full := image.Rect(0, 0, 10, 10)
	fundo := color.RGBA{1, 2, 3, 255}
	dst := solid(full, fundo)
	a := solid(full, color.RGBA{255, 0, 0, 255})
	bb := solid(full, color.RGBA{0, 0, 255, 255})

	r := image.Rect(2, 3, 6, 7)
	CrossFade(dst, a, bb, r, 0.5)
	for y := full.Min.Y; y < full.Max.Y; y++ {
		for x := full.Min.X; x < full.Max.X; x++ {
			dentro := image.Pt(x, y).In(r)
			c := dst.RGBAAt(x, y)
			if !dentro && c != fundo {
				t.Fatalf("pixel fora da região (%d,%d) foi tocado: %v", x, y, c)
			}
			if dentro && c == fundo {
				t.Fatalf("pixel dentro da região (%d,%d) não foi misturado", x, y)
			}
		}
	}

	// Fontes menores que a região: só a interseção é tocada.
	dst2 := solid(full, fundo)
	menor := solid(image.Rect(0, 0, 4, 4), color.RGBA{255, 255, 255, 255})
	CrossFade(dst2, menor, bb, full, 0.5)
	if c := dst2.RGBAAt(5, 5); c != fundo {
		t.Fatalf("pixel além dos bounds da fonte foi tocado: %v", c)
	}
}

// TestCrossFadeSemAlocacao garante o contrato de zero alocações.
func TestCrossFadeSemAlocacao(t *testing.T) {
	b := image.Rect(0, 0, 64, 64)
	a := solid(b, color.RGBA{200, 40, 10, 255})
	bb := solid(b, color.RGBA{20, 180, 240, 255})
	dst := image.NewRGBA(b)

	allocs := testing.AllocsPerRun(50, func() {
		CrossFade(dst, a, bb, b, 0.37)
	})
	if allocs != 0 {
		t.Fatalf("CrossFade alocou %v vezes por chamada; esperava 0", allocs)
	}
}
