package uitest_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/syntax"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCodeEditorAutoIndent: Enter herda a indentação da linha e ganha um
// tab extra após um abridor de bloco; Enter no meio da indentação não a
// duplica.
func TestCodeEditorAutoIndent(t *testing.T) {
	ed, h := editorTeste(t)

	h.Type("\tif x {")
	h.Key(juigo.KeyEnter)
	h.Type("y")
	if ed.Text() != "\tif x {\n\t\ty" {
		t.Fatalf("Enter após '{' deveria herdar e aprofundar: %q", ed.Text())
	}

	// Enter com o cursor DENTRO da indentação herda só o que vem antes.
	ed.SetText("\t\tz")
	ed.SetCursor(0, 1)
	h.Key(juigo.KeyEnter)
	if ed.Text() != "\t\n\t\tz" {
		t.Fatalf("Enter no meio da indentação: %q", ed.Text())
	}
}

// TestCodeEditorIndentarBloco: Tab com seleção multilinha indenta cada
// linha, Shift+Tab remove (tab OU até tabCols espaços) e o lote inteiro é
// UM passo de undo.
func TestCodeEditorIndentarBloco(t *testing.T) {
	ed, h := editorTeste(t)
	ed.SetText("um\n  dois\ntrês")

	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyTab)
	if ed.Text() != "\tum\n\t  dois\n\ttrês" {
		t.Fatalf("indentar bloco: %q", ed.Text())
	}
	h.Key(juigo.KeyZ, juigo.ModControl)
	if ed.Text() != "um\n  dois\ntrês" {
		t.Fatalf("o lote deveria desfazer de uma vez: %q", ed.Text())
	}

	// Shift+Tab remove um tab OU os espaços iniciais (até tabCols).
	ed.SetText("\tum\n  dois\ntrês")
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyTab, juigo.ModShift)
	if ed.Text() != "um\ndois\ntrês" {
		t.Fatalf("remover indentação do bloco: %q", ed.Text())
	}

	// Sem seleção, Shift+Tab age na linha do cursor e ajusta a coluna.
	ed.SetText("\tabc")
	ed.SetCursor(0, 3)
	h.Key(juigo.KeyTab, juigo.ModShift)
	if ed.Text() != "abc" {
		t.Fatalf("Shift+Tab na linha: %q", ed.Text())
	}
	if _, col := ed.Cursor(); col != 2 {
		t.Fatalf("a coluna deveria acompanhar a remoção; veio %d", col)
	}
}

// TestCodeEditorLinhaAtualEParenteses: a faixa da linha do cursor aparece
// (e acompanha o cursor), e o par de parênteses é realçado — pulando os que
// vivem dentro de strings quando o highlight está ativo.
func TestCodeEditorLinhaAtualEParenteses(t *testing.T) {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go())
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 480, 300)
	th := h.Session().Theme()
	h.Click(uitest.OfType[*juigo.CodeEditor]())

	ed.SetText("g := (\"a)\" + x)\noutra")
	ed.SetCursor(0, 6) // logo após o '(' — o ')' de dentro da string não conta
	img := h.Screenshot()

	// Faixa da linha atual: um ponto vazio à direita do texto, na linha 0.
	caret := ed.CaretRect()
	direita := image.Pt(ed.Bounds().Max.X-10, caret.Min.Y+2)
	if got := img.RGBAAt(direita.X, direita.Y); got != th.CurrentLine {
		t.Fatalf("a linha do cursor deveria ter a faixa %v; veio %v", th.CurrentLine, got)
	}

	// O par realçado é o ÚLTIMO ')', não o de dentro da string.
	adv := th.MonoAdvance()
	baseX := caret.Min.X - 6*adv // início do conteúdo (coluna 0)
	parX := baseX + 14*adv       // coluna do ')' real
	if got := img.RGBAAt(parX+1, caret.Min.Y+1); got != th.Selection {
		t.Fatalf("o ')' verdadeiro deveria estar realçado; veio %v", got)
	}
	strX := baseX + 8*adv // o ')' dentro da string
	if got := img.RGBAAt(strX+1, caret.Min.Y+1); got == th.Selection {
		t.Fatal("o ')' dentro da string não deveria ser o par")
	}

	// Mover para a outra linha: a faixa acompanha e o realce some.
	ed.SetCursor(1, 0)
	img = h.Screenshot()
	if got := img.RGBAAt(direita.X, direita.Y); got == th.CurrentLine {
		t.Fatal("a faixa deveria sair da linha 0")
	}
	if got := img.RGBAAt(parX+1, caret.Min.Y+1); got == th.Selection {
		t.Fatal("sem bracket adjacente, nada deveria ficar realçado")
	}
}

// TestIncrementalCodeEditorAssist é o golden da fase 3: faixa da linha,
// par de parênteses, auto-indent e indentação de bloco mantêm o frame
// incremental byte a byte igual ao completo.
func TestIncrementalCodeEditorAssist(t *testing.T) {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go())
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 420, 260)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			t.Fatalf("%s: render incremental divergiu do completo", passo)
		}
	}

	h.Click(uitest.OfType[*juigo.CodeEditor]())
	h.Type("func f(a int) {")
	verifica("digitação com par realçado")
	h.Key(juigo.KeyEnter)
	verifica("auto-indent")
	h.Type("g(x)")
	verifica("par novo na linha de baixo")
	h.Key(juigo.KeyLeft)
	verifica("cursor encosta no par")
	h.Key(juigo.KeyUp)
	verifica("faixa e realce trocam de linha")

	h.Key(juigo.KeyA, juigo.ModControl)
	verifica("seleção total (faixa some)")
	h.Key(juigo.KeyTab)
	verifica("indentar bloco")
	h.Key(juigo.KeyTab, juigo.ModShift)
	verifica("remover indentação do bloco")
	h.Key(juigo.KeyZ, juigo.ModControl)
	verifica("undo do lote")
}
