package uitest_test

import (
	"bytes"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// telaCor é uma tela de cor sólida: transições viram geometria pura,
// verificável pixel a pixel.
type telaCor struct {
	juigo.BaseWidget
	cor color.RGBA
}

func (t *telaCor) Draw(dst *image.RGBA) {
	render.FillRect(dst, t.Bounds(), t.cor)
}

var (
	vermelho = color.RGBA{R: 255, A: 255}
	azul     = color.RGBA{B: 255, A: 255}
)

// pixel compara a cor de um ponto do último frame com tolerância por canal.
func pixel(t *testing.T, h *uitest.Harness, p image.Point, quer color.RGBA, tol int, passo string) {
	t.Helper()
	got := h.Screenshot().RGBAAt(p.X, p.Y)
	diff := func(a, b uint8) int {
		d := int(a) - int(b)
		if d < 0 {
			return -d
		}
		return d
	}
	if diff(got.R, quer.R) > tol || diff(got.G, quer.G) > tol || diff(got.B, quer.B) > tol {
		t.Fatalf("%s: pixel %v = %v, esperava ~%v", passo, p, got, quer)
	}
}

// TestNavigatorSlidePorPixels acompanha um Push (SlideLeft) e o Pop reverso
// (SlideRight) no meio do caminho. Com duração 160ms, Advance(96ms) dá seis
// quadros de 16ms: t=0,6, easing EaseInOut → progresso 0,744 — a fronteira
// entre as telas fica a 74,4% da largura de 300px (deslocamento 223).
func TestNavigatorSlidePorPixels(t *testing.T) {
	nav := juigo.NewNavigator().Duration(160 * time.Millisecond)
	nav.Push(&telaCor{cor: vermelho})
	h := uitest.New(t, nav, 300, 200)

	pixel(t, h, image.Pt(150, 100), vermelho, 0, "tela inicial")

	nav.Push(&telaCor{cor: azul}) // padrão: SlideLeft, a nova entra da direita
	h.Advance(96 * time.Millisecond)
	if !nav.Animating() {
		t.Fatal("a transição deveria estar em andamento")
	}
	// Fronteira em x=300-223=77: a antiga ainda ocupa a beirada esquerda.
	pixel(t, h, image.Pt(10, 100), vermelho, 0, "meio do Push (lado da antiga)")
	pixel(t, h, image.Pt(150, 100), azul, 0, "meio do Push (lado da nova)")

	h.Advance(96 * time.Millisecond)
	if nav.Animating() {
		t.Fatal("a transição deveria ter terminado")
	}
	pixel(t, h, image.Pt(10, 100), azul, 0, "Push concluído")

	nav.Pop() // reverte: SlideRight, a anterior volta pela esquerda
	h.Advance(96 * time.Millisecond)
	// Mesmo progresso, direção oposta: fronteira em x=223 — o centro já é da
	// tela que VOLTA (no Push, no mesmo instante, o centro era da nova).
	pixel(t, h, image.Pt(150, 100), vermelho, 0, "meio do Pop (lado da que volta)")
	pixel(t, h, image.Pt(290, 100), azul, 0, "meio do Pop (lado da que sai)")

	h.Advance(96 * time.Millisecond)
	pixel(t, h, image.Pt(290, 100), vermelho, 0, "Pop concluído")
}

// TestNavigatorFadeMistura verifica o crossfade no meio do caminho: cada
// canal fica na mistura proporcional ao progresso (0,744 após 96ms/160ms).
func TestNavigatorFadeMistura(t *testing.T) {
	nav := juigo.NewNavigator().Duration(160 * time.Millisecond)
	nav.Push(&telaCor{cor: vermelho})
	h := uitest.New(t, nav, 300, 200)

	nav.Push(&telaCor{cor: azul}, juigo.TransitionFade)
	h.Advance(96 * time.Millisecond)
	p := 0.744
	misto := color.RGBA{R: uint8(255 * (1 - p)), B: uint8(255 * p), A: 255}
	pixel(t, h, image.Pt(150, 100), misto, 3, "meio do Fade")

	h.Advance(96 * time.Millisecond)
	pixel(t, h, image.Pt(150, 100), azul, 0, "Fade concluído")
}

// TestNavigatorEngoleInteracao: durante a transição as telas são retratos —
// cliques não alcançam os widgets; ao terminar, a tela nova está viva.
func TestNavigatorEngoleInteracao(t *testing.T) {
	cliques := 0
	nav := juigo.NewNavigator().Duration(160 * time.Millisecond)
	// A tela nova é um botão gigante: qualquer ponto serve para o clique.
	nav.Push(juigo.NewButton("Início", nil))
	h := uitest.New(t, nav, 300, 200)

	nav.Push(juigo.NewButton("Conta", func() { cliques++ }))
	h.Advance(48 * time.Millisecond)
	h.ClickAt(image.Pt(150, 100))
	if cliques != 0 {
		t.Fatalf("clique no meio da transição não deveria alcançar o botão; contou %d", cliques)
	}

	h.Advance(200 * time.Millisecond)
	h.Click(uitest.Text("Conta")) // a tela viva volta a ser encontrável
	if cliques != 1 {
		t.Fatalf("após a transição o botão deveria estar vivo; contou %d", cliques)
	}
}

// TestIncrementalNavegacao é a rede de segurança das dirty regions para o
// Navigator: em pleno slide, fade, pop e corte seco, o frame incremental
// deve ser byte a byte idêntico à repintura completa.
func TestIncrementalNavegacao(t *testing.T) {
	nav := juigo.NewNavigator().Duration(160 * time.Millisecond)
	nav.Push(juigo.NewVBox(
		juigo.NewText("Tela inicial"),
		juigo.NewInput("busca…"),
		juigo.NewButton("Avançar", nil),
	).Pad(12))
	h := uitest.New(t, nav, 320, 240)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			diff := 0
			for i := range incremental.Pix {
				if incremental.Pix[i] != completo.Pix[i] {
					diff++
				}
			}
			t.Fatalf("%s: render incremental divergiu do completo em %d bytes", passo, diff)
		}
	}

	verifica("frame inicial")

	detalhe := juigo.NewVBox(
		juigo.NewText("Detalhe"),
		juigo.NewButton("Voltar", func() { nav.Pop() }),
	).Pad(12)
	nav.Push(detalhe)
	h.Advance(48 * time.Millisecond)
	verifica("meio do slide")
	h.Advance(48 * time.Millisecond)
	verifica("mais adiante no slide")
	h.Advance(200 * time.Millisecond)
	verifica("slide concluído")

	nav.Replace(juigo.NewVBox(juigo.NewText("Substituta")).Pad(12))
	h.Advance(80 * time.Millisecond)
	verifica("meio do fade do Replace")
	h.Advance(200 * time.Millisecond)
	verifica("fade concluído")

	nav.Pop()
	h.Advance(48 * time.Millisecond)
	verifica("meio do pop")
	h.Advance(200 * time.Millisecond)
	verifica("pop concluído")

	nav.Push(juigo.NewText("corte"), juigo.TransitionNone)
	verifica("corte seco")
}
