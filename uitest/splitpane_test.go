package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestSplitPaneArrasteERatio arrasta o divisor e confere a geometria dos
// painéis, o vínculo em duas vias e os mínimos.
func TestSplitPaneArrasteERatio(t *testing.T) {
	ratio := juigo.NewState(0.5)
	esq := juigo.NewText("esquerda")
	dir := juigo.NewText("direita")
	sp := juigo.NewSplitPane(esq, dir).BindRatio(ratio).Min(40)
	h := uitest.New(t, sp, 400, 200)

	th := h.Session().Theme()
	faixa := th.Px(th.SplitterThickness)
	util := 400 - faixa

	// Meio a meio: o painel esquerdo termina perto da metade.
	meio := esq.Bounds().Max.X
	if diff := meio - util/2; diff < -1 || diff > 1 {
		t.Fatalf("com ratio 0,5 o painel esquerdo deveria terminar em ~%d; terminou em %d", util/2, meio)
	}

	// Arrasta o divisor 100px à direita: ratio e geometria acompanham.
	h.Drag(image.Pt(meio+faixa/2, 100), image.Pt(meio+faixa/2+100, 100))
	if got := ratio.Get(); got < 0.70 || got > 0.80 {
		t.Fatalf("após o arraste, ratio deveria ficar em ~0,75; ficou %v", got)
	}
	h.Layout()
	if got := esq.Bounds().Max.X; got <= meio+80 {
		t.Fatalf("o painel esquerdo deveria ter crescido ~100px; foi de %d a %d", meio, got)
	}

	// Set externo move o divisor (duas vias).
	ratio.Set(0.25)
	h.Layout()
	quarto := int(0.25*float64(util) + 0.5)
	if got := esq.Bounds().Max.X; got < quarto-1 || got > quarto+1 {
		t.Fatalf("Set(0,25) deveria levar o divisor a ~%d; foi a %d", quarto, got)
	}

	// Mínimos: nem o teclado leva além (Home tenta 0; o layout segura em 40).
	h.ClickAt(image.Pt(esq.Bounds().Max.X+faixa/2, 100)) // foca o divisor
	if h.Focused() != sp {
		t.Fatalf("clicar na faixa deveria focar o SplitPane; focou %v", h.Focused())
	}
	h.Key(juigo.KeyHome)
	h.Layout()
	minPx := th.Px(40)
	if got := esq.Bounds().Dx(); got != minPx {
		t.Fatalf("com Min(40), Home deveria parar o painel em %dpx; parou em %d", minPx, got)
	}
	h.Key(juigo.KeyEnd)
	h.Layout()
	if got := dir.Bounds().Dx(); got != minPx {
		t.Fatalf("com Min(40), End deveria parar o painel direito em %dpx; parou em %d", minPx, got)
	}
}

// TestSplitPaneVertical confere o eixo empilhado: divisor horizontal, setas
// Cima/Baixo movem.
func TestSplitPaneVertical(t *testing.T) {
	topo := juigo.NewText("topo")
	base := juigo.NewText("base")
	sp := juigo.NewSplitPane(topo, base).Vertical().Ratio(0.5)
	h := uitest.New(t, sp, 200, 400)

	th := h.Session().Theme()
	faixa := th.Px(th.SplitterThickness)
	meio := topo.Bounds().Max.Y

	h.ClickAt(image.Pt(100, meio+faixa/2))
	if h.Focused() != sp {
		t.Fatal("clicar na faixa deveria focar o divisor")
	}
	h.Key(juigo.KeyDown)
	h.Layout()
	if topo.Bounds().Max.Y <= meio {
		t.Fatalf("KeyDown deveria descer o divisor; %d → %d", meio, topo.Bounds().Max.Y)
	}
	h.Key(juigo.KeyUp)
	h.Key(juigo.KeyUp)
	h.Layout()
	if topo.Bounds().Max.Y >= meio {
		t.Fatalf("KeyUp duas vezes deveria subir além do início; %d → %d", meio, topo.Bounds().Max.Y)
	}
}
