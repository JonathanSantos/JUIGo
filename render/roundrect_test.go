package render

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// novaTela cria um buffer com conteúdo determinístico e não uniforme, para
// que qualquer escrita (ou mistura) indevida apareça na comparação.
func novaTela(b image.Rectangle) *image.RGBA {
	img := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 13), B: uint8(x + y), A: 0xFF})
		}
	}
	return img
}

var (
	corTeste  = color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF}
	corTransl = color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x80}
)

// TestRaioZeroEquivaleAoRetanguloReto garante a base do visual clássico:
// com raio zero (ou negativo), as primitivas arredondadas produzem byte a
// byte o mesmo resultado das primitivas retas.
func TestRaioZeroEquivaleAoRetanguloReto(t *testing.T) {
	r := image.Rect(3, 5, 40, 25)
	for _, raio := range []int{0, -3} {
		reto := novaTela(image.Rect(0, 0, 48, 32))
		redondo := novaTela(image.Rect(0, 0, 48, 32))

		FillRect(reto, r, corTeste)
		FillRoundRect(redondo, r, raio, corTeste)
		if !bytes.Equal(reto.Pix, redondo.Pix) {
			t.Errorf("FillRoundRect com raio %d difere de FillRect", raio)
		}

		StrokeRect(reto, r, 2, corTeste)
		StrokeRoundRect(redondo, r, raio, 2, corTeste)
		if !bytes.Equal(reto.Pix, redondo.Pix) {
			t.Errorf("StrokeRoundRect com raio %d difere de StrokeRect", raio)
		}

		FillRectOver(reto, r, corTransl)
		FillRoundRectOver(redondo, r, raio, corTransl)
		if !bytes.Equal(reto.Pix, redondo.Pix) {
			t.Errorf("FillRoundRectOver com raio %d difere de FillRectOver", raio)
		}
	}
}

// TestFillRoundRectFormaBasica confere miolo e beiradas retas sólidas, e os
// pixels extremos dos cantos intocados (fora do arco).
func TestFillRoundRectFormaBasica(t *testing.T) {
	tela := novaTela(image.Rect(0, 0, 48, 32))
	fundo := novaTela(image.Rect(0, 0, 48, 32))
	r := image.Rect(4, 4, 44, 28)
	FillRoundRect(tela, r, 4, corTeste)

	solidos := []image.Point{
		{24, 16},          // centro
		{24, r.Min.Y},     // meio da beirada superior
		{24, r.Max.Y - 1}, // meio da beirada inferior
		{r.Min.X, 16},     // meio da beirada esquerda
		{r.Max.X - 1, 16}, // meio da beirada direita
	}
	for _, p := range solidos {
		if got := tela.RGBAAt(p.X, p.Y); got != corTeste {
			t.Errorf("pixel %v deveria ser sólido %v; veio %v", p, corTeste, got)
		}
	}

	intocados := []image.Point{
		{r.Min.X, r.Min.Y},
		{r.Max.X - 1, r.Min.Y},
		{r.Min.X, r.Max.Y - 1},
		{r.Max.X - 1, r.Max.Y - 1},
		{r.Min.X - 1, 16}, // fora do retângulo
	}
	for _, p := range intocados {
		if got, want := tela.RGBAAt(p.X, p.Y), fundo.RGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v deveria ficar intocado (%v); veio %v", p, want, got)
		}
	}
}

// TestFillRoundRectSimetria: os quatro cantos são espelhos exatos uns dos
// outros nos dois eixos.
func TestFillRoundRectSimetria(t *testing.T) {
	tela := image.NewRGBA(image.Rect(0, 0, 40, 30))
	r := image.Rect(2, 3, 38, 27)
	FillRoundRect(tela, r, 6, corTeste)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ex := r.Min.X + r.Max.X - 1 - x
			ey := r.Min.Y + r.Max.Y - 1 - y
			if a, b := tela.RGBAAt(x, y), tela.RGBAAt(ex, y); a != b {
				t.Fatalf("assimetria horizontal em (%d,%d): %v != %v", x, y, a, b)
			}
			if a, b := tela.RGBAAt(x, y), tela.RGBAAt(x, ey); a != b {
				t.Fatalf("assimetria vertical em (%d,%d): %v != %v", x, y, a, b)
			}
		}
	}
}

// TestStrokeRoundRectAnel: o contorno pinta as beiradas, poupa o miolo e não
// vaza nos cantos externos.
func TestStrokeRoundRectAnel(t *testing.T) {
	tela := novaTela(image.Rect(0, 0, 48, 32))
	fundo := novaTela(image.Rect(0, 0, 48, 32))
	r := image.Rect(4, 4, 44, 28)
	StrokeRoundRect(tela, r, 4, 1, corTeste)

	for _, p := range []image.Point{{24, r.Min.Y}, {24, r.Max.Y - 1}, {r.Min.X, 16}, {r.Max.X - 1, 16}} {
		if got := tela.RGBAAt(p.X, p.Y); got != corTeste {
			t.Errorf("beirada %v deveria ser %v; veio %v", p, corTeste, got)
		}
	}
	for _, p := range []image.Point{{24, 16}, {24, r.Min.Y + 2}, {r.Min.X, r.Min.Y}, {r.Max.X - 1, r.Max.Y - 1}} {
		if got, want := tela.RGBAAt(p.X, p.Y), fundo.RGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v deveria ficar intocado (%v); veio %v", p, want, got)
		}
	}
}

// TestStrokeRoundRectEspesso: espessura maior que o raio preenche o canto até
// o arco externo sem buracos entre as barras retas e os blocos de canto.
func TestStrokeRoundRectEspesso(t *testing.T) {
	tela := image.NewRGBA(image.Rect(0, 0, 48, 32))
	r := image.Rect(4, 4, 44, 28)
	StrokeRoundRect(tela, r, 3, 6, corTeste)
	// Região do canto entre o raio (3) e a espessura (6): precisa estar cheia.
	for _, p := range []image.Point{{r.Min.X + 4, r.Min.Y + 4}, {r.Min.X + 1, r.Min.Y + 4}, {r.Min.X + 4, r.Min.Y + 1}} {
		if got := tela.RGBAAt(p.X, p.Y); got != corTeste {
			t.Errorf("canto espesso %v deveria ser %v; veio %v", p, corTeste, got)
		}
	}
}

// TestRaioMaiorQueORetangulo: raios exagerados são reduzidos à metade do
// menor lado — o resultado é idêntico ao do raio máximo.
func TestRaioMaiorQueORetangulo(t *testing.T) {
	r := image.Rect(0, 0, 20, 8)
	a := novaTela(image.Rect(0, 0, 24, 12))
	b := novaTela(image.Rect(0, 0, 24, 12))
	FillRoundRect(a, r, 100, corTeste)
	FillRoundRect(b, r, 4, corTeste)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Error("raio 100 num retângulo de altura 8 deveria equivaler ao raio 4")
	}
}

// TestRecorteNosLimites: desenhar parcialmente fora do buffer recorta sem
// pânico e produz, na área visível, o mesmo resultado de um buffer maior.
func TestRecorteNosLimites(t *testing.T) {
	r := image.Rect(-8, -6, 20, 14)
	pequena := novaTela(image.Rect(0, 0, 32, 24))
	grande := novaTela(image.Rect(-16, -16, 32, 24))

	FillRoundRect(pequena, r, 5, corTeste)
	StrokeRoundRect(pequena, r.Add(image.Pt(12, 4)), 5, 2, corTeste)
	FillRoundRect(grande, r, 5, corTeste)
	StrokeRoundRect(grande, r.Add(image.Pt(12, 4)), 5, 2, corTeste)

	for y := 0; y < 24; y++ {
		for x := 0; x < 32; x++ {
			if a, b := pequena.RGBAAt(x, y), grande.RGBAAt(x, y); a != b {
				t.Fatalf("recorte divergente em (%d,%d): %v != %v", x, y, a, b)
			}
		}
	}
}

// TestFillRoundRectOverTransluz: o Over mistura com o fundo no miolo (como
// FillRectOver) e poupa os cantos externos.
func TestFillRoundRectOverTransluz(t *testing.T) {
	tela := novaTela(image.Rect(0, 0, 48, 32))
	referencia := novaTela(image.Rect(0, 0, 48, 32))
	r := image.Rect(4, 4, 44, 28)
	FillRoundRectOver(tela, r, 4, corTransl)
	FillRectOver(referencia, r, corTransl)

	if got, want := tela.RGBAAt(24, 16), referencia.RGBAAt(24, 16); got != want {
		t.Errorf("miolo do Over deveria ser %v; veio %v", want, got)
	}
	fundo := novaTela(image.Rect(0, 0, 48, 32))
	if got, want := tela.RGBAAt(r.Min.X, r.Min.Y), fundo.RGBAAt(r.Min.X, r.Min.Y); got != want {
		t.Errorf("canto do Over deveria ficar intocado (%v); veio %v", want, got)
	}
}

// TestRoundRectNaoAloca: as três primitivas mantêm a regra do caminho quente
// de desenho — zero alocações.
func TestRoundRectNaoAloca(t *testing.T) {
	tela := image.NewRGBA(image.Rect(0, 0, 64, 64))
	r := image.Rect(4, 4, 60, 60)
	if n := testing.AllocsPerRun(50, func() {
		FillRoundRect(tela, r, 8, corTeste)
		StrokeRoundRect(tela, r, 8, 2, corTeste)
		FillRoundRectOver(tela, r, 8, corTransl)
	}); n != 0 {
		t.Errorf("primitivas arredondadas alocaram %.0f vez(es) por chamada", n)
	}
}
