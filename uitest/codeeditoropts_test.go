package uitest_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/syntax"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCodeEditorWrap: com WrapLines a linha lógica flui para linhas
// visuais (sem rolagem horizontal), Baixo navega por linha VISUAL dentro
// da mesma linha lógica, e desligar volta à rolagem horizontal.
func TestCodeEditorWrap(t *testing.T) {
	ed := juigo.NewCodeEditor().WrapLines(true)
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 260, 220)
	h.Click(uitest.OfType[*juigo.CodeEditor]())

	longa := strings.Repeat("abcdefghij", 8) // 80 colunas numa janela estreita
	ed.SetText(longa)
	if ed.LineCount() != 1 {
		t.Fatalf("uma linha lógica; veio %d", ed.LineCount())
	}

	ed.SetCursor(0, 0)
	h.Screenshot()
	topo := ed.CaretRect()
	ed.SetCursor(0, 70)
	fundo := ed.CaretRect()
	if fundo.Min.Y <= topo.Min.Y {
		t.Fatal("com wrap, uma coluna alta deveria cair numa linha visual de baixo")
	}
	if fundo.Min.X >= ed.Bounds().Max.X {
		t.Fatal("com wrap, nada deveria estourar a largura")
	}

	// Baixo desce UMA linha visual, continuando na mesma linha lógica.
	ed.SetCursor(0, 3)
	h.Key(juigo.KeyDown)
	if l, col := ed.Cursor(); l != 0 || col <= 3 {
		t.Fatalf("Baixo deveria avançar dentro da linha lógica; veio %d,%d", l, col)
	}

	// Desligar o wrap: volta à rolagem horizontal (caret visível à direita).
	ed.WrapLines(false)
	ed.SetCursor(0, 70)
	h.Screenshot()
	caret := ed.CaretRect()
	if caret.Min.Y != topo.Min.Y {
		t.Fatal("sem wrap, a linha volta a ser uma só")
	}
	if caret.Min.X < ed.Bounds().Min.X || caret.Max.X > ed.Bounds().Max.X {
		t.Fatalf("sem wrap, a rolagem horizontal deveria manter o caret visível; %v", caret)
	}
}

// TestCodeEditorScrollbars: o indicador vertical aparece quando o conteúdo
// excede a altura; o horizontal só sem wrap, quando a linha mais larga
// excede a largura.
func TestCodeEditorScrollbars(t *testing.T) {
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 260, 200)
	th := h.Session().Theme()
	h.Click(uitest.OfType[*juigo.CodeEditor]())

	longa := strings.Repeat("x", 90)
	ed.SetText(longa + "\n" + strings.Repeat("linha\n", 60))
	img := h.Screenshot()

	borda := th.BorderPx()
	margem := 2 * borda
	dirX := ed.Bounds().Max.X - borda - margem - 1
	meioY := ed.Bounds().Min.Y + ed.Bounds().Dy()/2
	// O indicador vertical vive na beirada direita; o cursor está no topo,
	// então o polegar cobre o topo — amostra lá.
	topoY := ed.Bounds().Min.Y + borda + margem + 2
	if got := img.RGBAAt(dirX, topoY); got != th.Placeholder {
		t.Fatalf("indicador vertical deveria estar em cena; veio %v", got)
	}
	_ = meioY

	baixoY := ed.Bounds().Max.Y - borda - margem - 1
	gutterFim := ed.Bounds().Min.X + borda + 40 // além do gutter
	if got := img.RGBAAt(gutterFim+30, baixoY); got != th.Placeholder {
		t.Fatalf("indicador horizontal deveria estar em cena; veio %v", got)
	}

	// Com wrap, o horizontal some.
	ed.WrapLines(true)
	img = h.Screenshot()
	if got := img.RGBAAt(gutterFim+30, baixoY); got == th.Placeholder {
		t.Fatal("com wrap não deveria haver indicador horizontal")
	}
}

// TestCodeEditorFontSizeEFira: FontSize muda as métricas DESTE editor
// (linha mais alta) e a Fira Code embutida troca a fonte mono do tema com
// a conta coluna↔pixel intacta.
func TestCodeEditorFontSizeEFira(t *testing.T) {
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 320, 220)
	th := h.Session().Theme()
	h.Click(uitest.OfType[*juigo.CodeEditor]())
	ed.SetText("abcdef")

	h.Screenshot()
	h1 := ed.CaretRect().Dy()
	ed.FontSize(24)
	h.Screenshot()
	if h2 := ed.CaretRect().Dy(); h2 <= h1 {
		t.Fatalf("FontSize(24) deveria aumentar a linha; %d → %d", h1, h2)
	}
	ed.FontSize(0)
	h.Screenshot()
	if h3 := ed.CaretRect().Dy(); h3 != h1 {
		t.Fatalf("FontSize(0) deveria voltar ao tema; %d ≠ %d", h3, h1)
	}

	// Fira Code: a coluna N continua a N células da origem.
	fira, err := theme.FiraCode()
	if err != nil {
		t.Fatal(err)
	}
	if err := th.UseMonoFont(fira); err != nil {
		t.Fatal(err)
	}
	h.Session().InvalidateAll()
	ed.SetCursor(0, 0)
	h.Screenshot()
	base := ed.CaretRect().Min.X
	ed.SetCursor(0, 3)
	if got := ed.CaretRect().Min.X; got != base+3*th.MonoAdvance() {
		t.Fatalf("com a Fira, coluna 3 deveria estar a 3 células; %d ≠ %d", got, base+3*th.MonoAdvance())
	}
}

// TestIncrementalCodeEditorWrap é o golden das opções: wrap ligado (com
// highlight), digitação re-fluindo linhas, liga/desliga, FontSize e
// indicadores mantêm o incremental byte a byte igual ao completo.
func TestIncrementalCodeEditorWrap(t *testing.T) {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go()).WrapLines(true)
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 300, 220)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			t.Fatalf("%s: render incremental divergiu do completo", passo)
		}
	}

	ed.SetText("func nomeBemComprido(argumento int, outroArgumento string) error {\n\treturn nil\n}\n" + strings.Repeat("// comentário\n", 30))
	verifica("SetText com wrap")

	h.Click(uitest.OfType[*juigo.CodeEditor]())
	verifica("clique")
	h.Type("xyz")
	verifica("digitação re-fluindo a linha")
	h.Key(juigo.KeyDown)
	verifica("navegação por linha visual")
	h.Scroll(uitest.OfType[*juigo.CodeEditor](), -4)
	verifica("rolagem com indicador")

	ed.FontSize(20)
	verifica("FontSize maior")
	ed.FontSize(0)
	verifica("FontSize de volta ao tema")

	ed.WrapLines(false)
	verifica("wrap desligado")
	h.Key(juigo.KeyEnd)
	verifica("rolagem horizontal + indicador")
	ed.WrapLines(true)
	verifica("wrap religado")
}
