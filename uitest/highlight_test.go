package uitest_test

import (
	"bytes"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/syntax"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCodeEditorHighlight cobre o highlight vivo: keywords coloridas em
// cena, abrir /* no topo comenta as linhas de baixo (cascata do re-lex
// incremental), fechar reverte, e undo também re-lexa.
func TestCodeEditorHighlight(t *testing.T) {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go())
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 480, 300)
	th := h.Session().Theme()
	kw := th.Syntax.Keyword
	cm := th.Syntax.Comment

	ed.SetText("package main\n\nfunc main() {\n\tx := 42\n}")
	img := h.Screenshot()
	if !temCorProxima(img, kw.R, kw.G, kw.B) {
		t.Fatal("as keywords deveriam estar na cor da paleta")
	}
	if !temCorProxima(img, th.Syntax.Number.R, th.Syntax.Number.G, th.Syntax.Number.B) {
		t.Fatal("o número deveria estar na cor da paleta")
	}

	// Abre /* na primeira linha: TUDO abaixo vira comentário (cascata).
	h.Click(uitest.OfType[*juigo.CodeEditor]())
	ed.SetCursor(0, 0)
	h.Type("/*")
	img = h.Screenshot()
	if temCorProxima(img, kw.R, kw.G, kw.B) {
		t.Fatal("com o bloco aberto, nenhuma keyword deveria sobrar")
	}
	if !temCorProxima(img, cm.R, cm.G, cm.B) {
		t.Fatal("o conteúdo deveria estar na cor de comentário")
	}

	// Fecha o bloco no fim da linha (End quebra o grupo de digitação —
	// senão o undo removeria o /**/ inteiro de uma vez): o resto volta ao
	// normal.
	h.Key(juigo.KeyEnd)
	h.Type("*/")
	img = h.Screenshot()
	if !temCorProxima(img, kw.R, kw.G, kw.B) {
		t.Fatal("fechar o bloco deveria devolver as keywords")
	}

	// Undo (remove só o */): a cascata re-lexa de novo.
	h.Key(juigo.KeyZ, juigo.ModControl)
	img = h.Screenshot()
	if temCorProxima(img, kw.R, kw.G, kw.B) {
		t.Fatal("o undo deveria reabrir o comentário")
	}
}

// TestIncrementalCodeEditorHighlight é o golden da fase 2: com highlighter
// ativo, cada interação (inclusive as cascatas de comentário) mantém o
// frame incremental byte a byte igual ao completo.
func TestIncrementalCodeEditorHighlight(t *testing.T) {
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

	ed.SetText("package main\n\nfunc soma(a, b int) int {\n\treturn a + b\n}\n\nvar total = 10")
	verifica("SetText com highlight")

	h.Click(uitest.OfType[*juigo.CodeEditor]())
	ed.SetCursor(3, 8)
	h.Type("x")
	verifica("digitação numa linha destacada")

	ed.SetCursor(0, 0)
	h.Type("/*")
	verifica("cascata: bloco aberto no topo")
	h.Type("*/")
	verifica("cascata: bloco fechado")

	h.Key(juigo.KeyZ, juigo.ModControl)
	verifica("undo da cascata")
	h.Key(juigo.KeyZ, juigo.ModControl|juigo.ModShift)
	verifica("redo da cascata")

	ed.SetCursor(2, 0)
	h.Type("// ")
	verifica("comentário de linha")
	h.Key(juigo.KeyZ, juigo.ModControl)
	verifica("undo do comentário de linha")
}
