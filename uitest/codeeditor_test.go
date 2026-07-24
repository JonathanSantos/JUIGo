package uitest_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// editorTeste monta um CodeEditor focado num harness.
func editorTeste(t *testing.T) (*juigo.CodeEditor, *uitest.Harness) {
	t.Helper()
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 480, 300)
	h.Click(uitest.OfType[*juigo.CodeEditor]())
	return ed, h
}

// TestCodeEditorEdicao cobre digitação, Enter, Tab LITERAL (sem navegar o
// foco), tab stops na célula mono e o undo/redo coalescido por teclado.
func TestCodeEditorEdicao(t *testing.T) {
	ed, h := editorTeste(t)
	th := h.Session().Theme()

	h.Type("func main() {")
	h.Key(juigo.KeyEnter) // auto-indent: herda e ganha um tab após '{'
	h.Type(`println("olá")`)
	h.Key(juigo.KeyEnter)               // herda o "\t"
	h.Key(juigo.KeyTab, juigo.ModShift) // Shift+Tab remove a indentação
	h.Type("}")

	want := "func main() {\n\tprintln(\"olá\")\n}"
	if ed.Text() != want || ed.LineCount() != 3 {
		t.Fatalf("conteúdo: %q (%d linhas)", ed.Text(), ed.LineCount())
	}
	if h.Focused() != juigo.Widget(ed) {
		t.Fatal("Tab literal não deveria navegar o foco")
	}

	// Tab stop: a coluna 1 da linha do tab fica na 4ª célula.
	adv := th.MonoAdvance()
	ed.SetCursor(1, 0)
	h.Screenshot()
	base := ed.CaretRect().Min.X
	ed.SetCursor(1, 1)
	if got := ed.CaretRect().Min.X; got != base+4*adv {
		t.Fatalf("tab stop: caret em %d, esperado %d", got, base+4*adv)
	}

	// Undo coalescido: "abc" digitado de uma vez sai de uma vez.
	ed.SetCursor(2, 1)
	h.Type("abc")
	h.Key(juigo.KeyZ, juigo.ModControl)
	if ed.Text() != want {
		t.Fatalf("undo da digitação: %q", ed.Text())
	}
	h.Key(juigo.KeyZ, juigo.ModControl|juigo.ModShift)
	if !strings.HasSuffix(ed.Text(), "}abc") {
		t.Fatalf("redo: %q", ed.Text())
	}
	if l, c := ed.Cursor(); l != 2 || c != 4 {
		t.Fatalf("cursor após redo: %d,%d", l, c)
	}
}

// TestCodeEditorSelecaoEClipboard: seleção multilinha por Shift+setas,
// copiar/recortar/colar com '\n' e substituição da seleção.
func TestCodeEditorSelecaoEClipboard(t *testing.T) {
	var clip string
	prevW, prevR := hooks.ClipboardWrite, hooks.ClipboardRead
	hooks.ClipboardWrite = func(s string) { clip = s }
	hooks.ClipboardRead = func() string { return clip }
	defer func() { hooks.ClipboardWrite, hooks.ClipboardRead = prevW, prevR }()

	ed, h := editorTeste(t)
	ed.SetCursor(0, 0)
	h.Type("um\ndois\ntrês")

	ed.SetCursor(0, 1)
	h.Key(juigo.KeyDown, juigo.ModShift)
	h.Key(juigo.KeyRight, juigo.ModShift)
	h.Key(juigo.KeyC, juigo.ModControl)
	if clip != "m\ndo" {
		t.Fatalf("copiar multilinha: %q", clip)
	}

	h.Key(juigo.KeyX, juigo.ModControl)
	if ed.Text() != "uis\ntrês" {
		t.Fatalf("recortar deveria remover a seleção: %q", ed.Text())
	}
	h.Key(juigo.KeyV, juigo.ModControl)
	if ed.Text() != "um\ndois\ntrês" {
		t.Fatalf("colar deveria restaurar: %q", ed.Text())
	}

	// Selecionar tudo + digitar substitui.
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("x")
	if ed.Text() != "x" || ed.LineCount() != 1 {
		t.Fatalf("substituir seleção total: %q", ed.Text())
	}
	h.Key(juigo.KeyZ, juigo.ModControl)
	if ed.Text() != "um\ndois\ntrês" {
		t.Fatalf("undo da substituição: %q", ed.Text())
	}
}

// TestCodeEditorNavegacao: setas atravessam linhas, Cima/Baixo preservam a
// coluna DESEJADA e o cursor fora da janela rola até ficar visível.
func TestCodeEditorNavegacao(t *testing.T) {
	ed, h := editorTeste(t)
	h.Type("aaaa\nb\ncccc")

	ed.SetCursor(0, 4)
	h.Key(juigo.KeyDown)
	if l, c := ed.Cursor(); l != 1 || c != 1 {
		t.Fatalf("descer para linha curta: %d,%d", l, c)
	}
	h.Key(juigo.KeyDown)
	if l, c := ed.Cursor(); l != 2 || c != 4 {
		t.Fatalf("a coluna desejada deveria sobreviver: %d,%d", l, c)
	}
	h.Key(juigo.KeyRight) // fim da linha 2 → não sai
	h.Key(juigo.KeyLeft)
	h.Key(juigo.KeyHome)
	if l, c := ed.Cursor(); l != 2 || c != 0 {
		t.Fatalf("Home: %d,%d", l, c)
	}

	// Documento grande: ir ao fim rola; a roda rola de volta sem mover o
	// cursor (o CaretRect acompanha o deslocamento).
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("linha\n")
	}
	ed.SetText(sb.String())
	ed.SetCursor(199, 0)
	h.Screenshot()
	caret := ed.CaretRect()
	if caret.Min.Y >= ed.Bounds().Max.Y || caret.Max.Y <= ed.Bounds().Min.Y {
		t.Fatalf("o cursor deveria estar visível após SetCursor; %v", caret)
	}
	h.Scroll(uitest.OfType[*juigo.CodeEditor](), 5)
	if ed.CaretRect().Min.Y <= caret.Min.Y {
		t.Fatal("rolar para cima deveria empurrar o caret para baixo da vista")
	}
	if l, _ := ed.Cursor(); l != 199 {
		t.Fatal("a roda não deveria mover o cursor")
	}
}

// TestCodeEditorPreedit: a composição de IME aparece no cursor (medida pela
// célula mono) sem entrar no texto; o commit chega pelo Type.
func TestCodeEditorPreedit(t *testing.T) {
	ed, h := editorTeste(t)
	h.Type("ab")
	h.Key(juigo.KeyLeft)
	h.Screenshot()
	base := ed.CaretRect()

	h.Preedit("xy", 1)
	if ed.Text() != "ab" {
		t.Fatalf("a composição não deveria entrar no texto: %q", ed.Text())
	}
	th := h.Session().Theme()
	if got := ed.CaretRect().Min.X; got != base.Min.X+th.MonoAdvance() {
		t.Fatalf("caret dentro da composição: %d, esperado %d", got, base.Min.X+th.MonoAdvance())
	}
	h.Preedit("", 0)
	h.Type("XY")
	if ed.Text() != "aXYb" {
		t.Fatalf("commit: %q", ed.Text())
	}
}

// TestIncrementalCodeEditor é o golden do editor: após cada interação, o
// frame incremental (dano por linha, caret, rolagem) deve ser byte a byte
// igual à repintura completa.
func TestIncrementalCodeEditor(t *testing.T) {
	ed := juigo.NewCodeEditor()
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 420, 260)
	th := h.Session().Theme()

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
	h.Type("package main")
	verifica("digitação na linha")
	h.Key(juigo.KeyEnter)
	h.Key(juigo.KeyEnter)
	h.Type("func main() {}")
	verifica("linhas novas")

	h.Key(juigo.KeyLeft)
	verifica("cursor na mesma linha")
	h.Key(juigo.KeyUp)
	verifica("cursor trocando de linha")
	h.Advance(th.CaretBlink)
	verifica("caret apagado")
	h.Advance(th.CaretBlink)
	verifica("caret aceso")

	h.Key(juigo.KeyEnd)
	h.Key(juigo.KeyLeft, juigo.ModShift)
	h.Key(juigo.KeyLeft, juigo.ModShift)
	verifica("seleção")
	h.Key(juigo.KeyBackspace)
	verifica("apagar seleção")
	h.Key(juigo.KeyTab)
	verifica("tab literal")
	h.Key(juigo.KeyZ, juigo.ModControl)
	verifica("undo")
	h.Key(juigo.KeyZ, juigo.ModControl|juigo.ModShift)
	verifica("redo")

	// Documento maior que a janela: rolagem por roda e por navegação.
	var sb strings.Builder
	for i := 0; i < 120; i++ {
		sb.WriteString("linha de código para rolar\n")
	}
	ed.SetText(sb.String())
	verifica("SetText grande")
	h.Click(uitest.OfType[*juigo.CodeEditor]())
	verifica("clique posiciona o cursor")
	h.Scroll(uitest.OfType[*juigo.CodeEditor](), -6)
	verifica("rolagem")
	ed.SetCursor(119, 0)
	verifica("ir ao fim")

	h.Preedit("comp", 2)
	verifica("composição de IME")
	h.Preedit("", 0)
	verifica("composição encerrada")
}
