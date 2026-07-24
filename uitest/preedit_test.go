package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
	"github.com/JonathanSantos/JUIGo/widget"
)

// TestPreeditNoInput cobre a fase D do IME no Input: a composição aparece
// inline no cursor com sublinhado (sem entrar no texto), o cursor anda
// dentro dela (CaretRect), blocos destacam o em conversão, o commit chega
// pelo Type normal e o blur descarta a composição.
func TestPreeditNoInput(t *testing.T) {
	campo := juigo.NewInput("Digite…")
	outro := juigo.NewInput("outro")
	h := uitest.New(t, juigo.NewVBox(campo, outro).Pad(8), 360, 160)
	th := h.Session().Theme()

	// Escreve "ab" e volta o cursor para o meio: a|b.
	h.Click(uitest.Placeholder("Digite…"))
	h.Type("ab")
	h.Key(juigo.KeyLeft)
	h.Screenshot()
	base := campo.CaretRect()

	// Composição "にほ" com o cursor após a 1ª rune.
	h.Preedit("にほ", 1)
	if campo.Text() != "ab" {
		t.Fatalf("a composição não deveria entrar no texto; veio %q", campo.Text())
	}
	img := h.Screenshot()

	// Sublinhado sob a composição (baseline + 2 lógicos), ausente sob "a".
	subY := campo.Bounds().Min.Y + (campo.Bounds().Dy()-th.LineHeight())/2 + th.Ascent() + th.Px(2)
	larguraComp := th.MeasureString("にほ")
	meioComp := base.Min.X + larguraComp/2
	if got := img.RGBAAt(meioComp, subY); got != th.Text {
		t.Fatalf("a composição deveria estar sublinhada; veio %v", got)
	}
	if got := img.RGBAAt(base.Min.X-2, subY); got == th.Text {
		t.Fatal("o texto confirmado não deveria ganhar sublinhado")
	}

	// O cursor fica DENTRO da composição, deslocado pela 1ª rune.
	caret := campo.CaretRect()
	if want := base.Min.X + th.MeasureString("に"); caret.Min.X != want {
		t.Fatalf("cursor dentro da composição: x=%d, esperado %d", caret.Min.X, want)
	}
	if r, ok := h.Session().CaretRect(); !ok || r != caret {
		t.Fatal("Session.CaretRect deveria expor o cursor do focado")
	}

	// Blocos: o focado (índice 1) ganha o sublinhado de destaque.
	h.Session().Preedit(juigo.PreeditEvent{Text: "にほんご", Caret: 4, Blocks: []int{2, 2}, FocusedBlock: 1})
	img = h.Screenshot()
	x2 := base.Min.X + th.MeasureString("にほ") + th.MeasureString("んご")/2
	if got := img.RGBAAt(x2, subY); got != th.Accent {
		t.Fatalf("o bloco em conversão deveria ter o destaque %v; veio %v", th.Accent, got)
	}
	if got := img.RGBAAt(base.Min.X+th.MeasureString("に"), subY); got != th.Text {
		t.Fatalf("o bloco comum deveria ter o sublinhado normal; veio %v", got)
	}

	// Commit: a composição encerra vazia e o texto chega pelo Type.
	h.Preedit("", 0)
	h.Type("日本語")
	if campo.Text() != "a日本語b" {
		t.Fatalf("o commit deveria entrar no cursor; veio %q", campo.Text())
	}
	img = h.Screenshot()
	if got := img.RGBAAt(meioComp, subY); got == th.Accent {
		t.Fatal("após o commit não deveria restar sublinhado")
	}

	// Blur no meio de uma composição a descarta.
	h.Preedit("かん", 2)
	h.Click(uitest.Placeholder("outro"))
	if campo.Text() != "a日本語b" {
		t.Fatalf("o blur não deveria alterar o texto; veio %q", campo.Text())
	}
	h.Click(uitest.Placeholder("Digite…"))
	h.Key(juigo.KeyEnd)
	h.Type("!")
	if campo.Text() != "a日本語b!" {
		t.Fatalf("a composição descartada não deveria voltar; veio %q", campo.Text())
	}
}

// TestPreeditNaTextArea: a composição aparece inline na linha VISUAL do
// cursor, o CaretRect anda dentro dela e o commit entra pelo Type.
func TestPreeditNaTextArea(t *testing.T) {
	area := juigo.NewTextArea("Notas…")
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(area, 1)), 320, 200)
	th := h.Session().Theme()
	pad := th.PaddingPx()

	h.Click(uitest.OfType[*juigo.TextArea]())
	h.Type("linha um")
	h.Key(juigo.KeyEnter)
	h.Type("dois")
	h.Preedit("かな", 1)
	if area.Text() != "linha um\ndois" {
		t.Fatalf("a composição não deveria entrar no texto; veio %q", area.Text())
	}
	img := h.Screenshot()

	compX := area.Bounds().Min.X + pad + th.MeasureString("dois")
	subY := area.Bounds().Min.Y + pad + th.LineHeight() + th.Ascent() + th.Px(2)
	if got := img.RGBAAt(compX+th.MeasureString("かな")/2, subY); got != th.Text {
		t.Fatalf("a composição na 2ª linha deveria estar sublinhada; veio %v", got)
	}
	caret := area.CaretRect()
	if want := compX + th.MeasureString("か"); caret.Min.X != want {
		t.Fatalf("cursor dentro da composição: x=%d, esperado %d", caret.Min.X, want)
	}

	h.Preedit("", 0)
	h.Type("仮名")
	if area.Text() != "linha um\ndois仮名" {
		t.Fatalf("o commit deveria entrar no cursor; veio %q", area.Text())
	}
}

// TestPreeditEmTextAreaVazia: compor num campo vazio desenha a composição
// na primeira linha (o placeholder não volta durante a composição).
func TestPreeditEmTextAreaVazia(t *testing.T) {
	area := juigo.NewTextArea("Notas…")
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(area, 1)), 320, 160)
	th := h.Session().Theme()
	pad := th.PaddingPx()

	h.Click(uitest.OfType[*juigo.TextArea]())
	h.Preedit("あ", 1)
	img := h.Screenshot()
	subY := area.Bounds().Min.Y + pad + th.Ascent() + th.Px(2)
	x := area.Bounds().Min.X + pad + th.MeasureString("あ")/2
	if got := img.RGBAAt(x, subY); got != th.Text {
		t.Fatalf("a composição em campo vazio deveria aparecer sublinhada; veio %v", got)
	}
	if area.Text() != "" {
		t.Fatalf("o texto deveria seguir vazio; veio %q", area.Text())
	}
}

// TestPreeditSubstituiSelecao: começar a compor sobre uma seleção a
// substitui, como digitar.
func TestPreeditSubstituiSelecao(t *testing.T) {
	campo := juigo.NewInput("Digite…")
	h := uitest.New(t, juigo.NewVBox(campo).Pad(8), 320, 120)

	h.Click(uitest.Placeholder("Digite…"))
	h.Type("velho")
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Preedit("しん", 2)
	if campo.Text() != "" {
		t.Fatalf("compor sobre a seleção deveria apagá-la; veio %q", campo.Text())
	}
	h.Preedit("", 0)
	h.Type("novo")
	if campo.Text() != "novo" {
		t.Fatalf("o commit deveria substituir; veio %q", campo.Text())
	}
}

// TestPreeditRolaParaOCursor: uma composição no fim de um texto maior que o
// campo rola o conteúdo para manter o cursor da composição visível.
func TestPreeditRolaParaOCursor(t *testing.T) {
	campo := juigo.NewInput("Digite…")
	h := uitest.New(t, juigo.NewVBox(campo).Pad(8), 200, 120)
	th := h.Session().Theme()

	h.Click(uitest.Placeholder("Digite…"))
	h.Type("texto bem comprido para estourar o campo")
	h.Preedit("にほんごのぶんしょう", 10)
	h.Screenshot()

	caret := campo.CaretRect()
	interno := image.Rect(
		campo.Bounds().Min.X+th.PaddingPx(), campo.Bounds().Min.Y,
		campo.Bounds().Max.X-th.PaddingPx(), campo.Bounds().Max.Y,
	)
	if caret.Min.X < interno.Min.X || caret.Max.X > interno.Max.X {
		t.Fatalf("o cursor da composição deveria estar visível; caret=%v área=%v", caret, interno)
	}
	var _ widget.TextCaret = campo // Input expõe o contrato do IME
}
