package uitest_test

import (
	"image"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestRolagemComTravaDeEixo: um gesto vertical com ruído horizontal (o
// trackpad real) NÃO mexe na horizontal; uma dominância forte do outro
// eixo retrava na hora; o gesto expira pelo relógio e libera o eixo.
func TestRolagemComTravaDeEixo(t *testing.T) {
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 260, 200)
	th := h.Session().Theme()
	h.Click(uitest.OfType[*juigo.CodeEditor]())

	ed.SetText(strings.Repeat("x", 120) + "\n" + strings.Repeat("linha\n", 80))
	ed.SetCursor(0, 0)
	h.Screenshot()
	base := ed.CaretRect()
	s := h.Session()
	meio := ed.Bounds().Min.Add(ed.Bounds().Size().Div(2))

	// Gesto vertical com ruído horizontal: só o Y rola.
	s.Scroll(meio, -0.4, -3)
	s.Scroll(meio, -0.6, -2)
	h.Screenshot()
	c1 := ed.CaretRect()
	if c1.Min.X != base.Min.X {
		t.Fatalf("o ruído horizontal não deveria rolar X; caret %d → %d", base.Min.X, c1.Min.X)
	}
	if c1.Min.Y >= base.Min.Y {
		t.Fatal("o gesto vertical deveria rolar Y")
	}

	// Dominância forte do X no meio do gesto: retrava na hora.
	s.Scroll(meio, -3, -0.2)
	h.Screenshot()
	c2 := ed.CaretRect()
	if c2.Min.X >= c1.Min.X {
		t.Fatal("a dominância forte do X deveria retravar e rolar X")
	}
	if c2.Min.Y != c1.Min.Y {
		t.Fatal("com o gesto travado no X, o Y não deveria mexer")
	}

	// O gesto expira: um gesto novo trava de novo pelo eixo dominante.
	h.Advance(2 * th.ScrollAxisLock)
	s.Scroll(meio, -0.3, -2)
	h.Screenshot()
	c3 := ed.CaretRect()
	if c3.Min.X != c2.Min.X || c3.Min.Y >= c2.Min.Y {
		t.Fatal("após expirar, o gesto novo deveria travar no eixo vertical")
	}
}

// blocoGrande é um filho 2D para exercitar o Scroll com Horizontal.
type blocoGrande struct{ juigo.BaseWidget }

func (b *blocoGrande) PreferredSize() image.Point { return image.Pt(900, 900) }

// TestTravaDeEixoNoScroll: o mesmo comportamento no container Scroll com o
// eixo horizontal habilitado.
func TestTravaDeEixoNoScroll(t *testing.T) {
	filho := &blocoGrande{}
	sc := juigo.NewScroll(filho).Horizontal()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(sc, 1)).Pad(8), 220, 180)
	s := h.Session()
	meio := sc.Bounds().Min.Add(sc.Bounds().Size().Div(2))

	h.Screenshot()
	base := filho.Bounds().Min
	s.Scroll(meio, -0.5, -3)
	h.Screenshot()
	depois := filho.Bounds().Min
	if depois.X != base.X {
		t.Fatalf("ruído horizontal vazou para o Scroll; X %d → %d", base.X, depois.X)
	}
	if depois.Y >= base.Y {
		t.Fatal("o gesto vertical deveria rolar o Scroll")
	}
}

// TestDuploCliqueSelecionaPalavra: nos três campos de texto, o duplo
// clique seleciona a corrida de caracteres sob o ponteiro.
func TestDuploCliqueSelecionaPalavra(t *testing.T) {
	var clip string
	prevW, prevR := hooks.ClipboardWrite, hooks.ClipboardRead
	hooks.ClipboardWrite = func(s string) { clip = s }
	hooks.ClipboardRead = func() string { return clip }
	defer func() { hooks.ClipboardWrite, hooks.ClipboardRead = prevW, prevR }()

	campo := juigo.NewInput("campo")
	area := juigo.NewTextArea("area")
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(campo, juigo.Grow(area, 1), juigo.Grow(ed, 1)).Pad(8), 420, 320)
	th := h.Session().Theme()

	// Input: palavra com '_' e acento é uma corrida só.
	h.Click(uitest.Placeholder("campo"))
	h.Type("olá mundo_São fim")
	x := campo.Bounds().Min.X + th.PaddingPx() + th.MeasureString("olá mu")
	h.DoubleClickAt(image.Pt(x, campo.Bounds().Min.Y+campo.Bounds().Dy()/2))
	h.Key(juigo.KeyC, juigo.ModControl)
	if clip != "mundo_São" {
		t.Fatalf("duplo clique no Input: %q", clip)
	}

	// TextArea: palavra na segunda linha.
	h.Click(uitest.Placeholder("area"))
	h.Type("um dois\ntrês quatro")
	pad := th.PaddingPx()
	ax := area.Bounds().Min.X + pad + th.MeasureString("trê")
	ay := area.Bounds().Min.Y + pad + th.LineHeight() + th.LineHeight()/2
	h.DoubleClickAt(image.Pt(ax, ay))
	h.Key(juigo.KeyC, juigo.ModControl)
	if clip != "três" {
		t.Fatalf("duplo clique na TextArea: %q", clip)
	}

	// CodeEditor: identificador sob o cursor.
	h.Click(uitest.OfType[*juigo.CodeEditor]())
	h.Type("func minhaFunc(x int) {")
	ed.SetCursor(0, 8) // dentro de minhaFunc
	h.Screenshot()
	p := ed.CaretRect()
	h.DoubleClickAt(image.Pt(p.Min.X+1, p.Min.Y+p.Dy()/2))
	h.Key(juigo.KeyC, juigo.ModControl)
	if clip != "minhaFunc" {
		t.Fatalf("duplo clique no CodeEditor: %q", clip)
	}
}
