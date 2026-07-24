package uitest_test

import (
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCodeEditorReadOnly: no modo visor a edição é bloqueada (digitação,
// Enter, Backspace, colar, undo, IME) mas navegação, seleção e CÓPIA
// seguem funcionando; SetText ainda carrega conteúdo.
func TestCodeEditorReadOnly(t *testing.T) {
	var clip string
	prevW, prevR := hooks.ClipboardWrite, hooks.ClipboardRead
	hooks.ClipboardWrite = func(s string) { clip = s }
	hooks.ClipboardRead = func() string { return clip }
	defer func() { hooks.ClipboardWrite, hooks.ClipboardRead = prevW, prevR }()

	ed := juigo.NewCodeEditor().ReadOnly(true)
	h := uitest.New(t, juigo.NewVBox(juigo.Grow(ed, 1)).Pad(8), 420, 260)
	ed.SetText("linha um\nlinha dois") // SetText é permitido no visor
	h.Click(uitest.OfType[*juigo.CodeEditor]())

	original := ed.Text()

	// Toda mutação é no-op.
	h.Type("xxx")
	h.Key(juigo.KeyEnter)
	h.Key(juigo.KeyBackspace)
	h.Key(juigo.KeyDelete)
	h.Key(juigo.KeyTab)
	clip = "colado"
	h.Key(juigo.KeyV, juigo.ModControl)
	h.Preedit("かな", 2) // composição de IME ignorada
	if ed.Text() != original {
		t.Fatalf("o visor não deveria mudar; veio %q", ed.Text())
	}

	// Navegação funciona.
	ed.SetCursor(0, 0)
	h.Key(juigo.KeyDown)
	if l, _ := ed.Cursor(); l != 1 {
		t.Fatal("navegação deveria funcionar no visor")
	}

	// Seleção + cópia funcionam (Ctrl+A, Ctrl+C).
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyC, juigo.ModControl)
	if clip != original {
		t.Fatalf("copiar no visor: %q", clip)
	}

	// Ctrl+X no visor apenas copia (não remove).
	clip = ""
	ed.SetCursor(0, 0)
	h.Key(juigo.KeyEnd, juigo.ModShift) // seleciona "linha um"
	h.Key(juigo.KeyX, juigo.ModControl)
	if clip != "linha um" || ed.Text() != original {
		t.Fatalf("Ctrl+X no visor deveria só copiar; clip=%q texto=%q", clip, ed.Text())
	}

	// Desligar volta a permitir edição.
	ed.ReadOnly(false)
	ed.SetCursor(0, 0)
	h.Type("Z")
	if ed.Text() == original {
		t.Fatal("desligar o ReadOnly deveria permitir editar")
	}
}
