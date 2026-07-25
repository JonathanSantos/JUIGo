package chart

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
)

// desenha monta w com o tema padrão, posiciona e desenha num buffer.
func desenha(t *testing.T, w widget.Widget, largura, altura int) *image.RGBA {
	t.Helper()
	th, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}
	widget.Mount(w, th)
	w.Layout(image.Rect(0, 0, largura, altura))
	dst := image.NewRGBA(image.Rect(0, 0, largura, altura))
	render.FillRect(dst, dst.Bounds(), th.Background)
	w.Draw(dst)
	return dst
}

// contaAccent conta pixels EXATAMENTE na cor de destaque do tema.
func contaAccent(t *testing.T, img *image.RGBA) int {
	t.Helper()
	th, _ := theme.Default()
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] == th.Accent.R && img.Pix[i+1] == th.Accent.G && img.Pix[i+2] == th.Accent.B {
			n++
		}
	}
	return n
}

// TestSparkELineDesenhamASerie: a polilinha aparece em Accent; sem dados,
// nada desenha (e nada estoura).
func TestSparkELineDesenhamASerie(t *testing.T) {
	dados := []float64{3, 7, 2, 9, 4, 6}
	if n := contaAccent(t, desenha(t, NewSpark(dados), 120, 32)); n == 0 {
		t.Fatal("a sparkline deveria pintar pixels em Accent")
	}
	if n := contaAccent(t, desenha(t, NewLine(dados).Min(0), 300, 180)); n == 0 {
		t.Fatal("o gráfico de linha deveria pintar a série em Accent")
	}
	if n := contaAccent(t, desenha(t, NewSpark(nil), 120, 32)); n != 0 {
		t.Fatalf("sem dados não deveria haver série; %d pixels", n)
	}
}

// TestBarsPositivosENegativos: barras sobem e descem da linha do zero.
func TestBarsPositivosENegativos(t *testing.T) {
	img := desenha(t, NewBars([]float64{4, -3, 2}), 240, 160)
	th, _ := theme.Default()
	acima, abaixo := 0, 0
	// A linha do zero fica entre as metades; conta Accent por metade.
	meio := 80
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			c := img.RGBAAt(x, y)
			if c.R == th.Accent.R && c.G == th.Accent.G && c.B == th.Accent.B {
				if y < meio {
					acima++
				} else {
					abaixo++
				}
			}
		}
	}
	if acima == 0 || abaixo == 0 {
		t.Fatalf("positivos deveriam subir e negativos descer; acima=%d abaixo=%d", acima, abaixo)
	}
}

// TestChartsSemAlocacaoNoDraw: desenhar de novo (buffers aquecidos) não
// aloca.
func TestChartsSemAlocacaoNoDraw(t *testing.T) {
	dados := []float64{3, 7, 2, 9, 4, 6, 1, 8}
	th, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}
	l := NewLine(dados)
	widget.Mount(l, th)
	l.Layout(image.Rect(0, 0, 300, 180))
	dst := image.NewRGBA(image.Rect(0, 0, 300, 180))
	l.Draw(dst) // aquece buffers e rótulos
	allocs := testing.AllocsPerRun(20, func() {
		l.Draw(dst)
	})
	if allocs != 0 {
		t.Fatalf("Line.Draw alocou %v vezes por chamada; esperava 0", allocs)
	}
}
