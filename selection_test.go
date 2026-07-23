package juigo

import (
	"image"
	"testing"
)

func TestInputSelecaoTeclado(t *testing.T) {
	in := NewInput("")
	in.SetTheme(newTestTheme(t))
	typeString(in, "ação")

	// Shift+Left duas vezes seleciona "ão" (âncora fica no fim).
	in.HandleEvent(KeyEvent{Key: KeyLeft, Mods: ModShift})
	in.HandleEvent(KeyEvent{Key: KeyLeft, Mods: ModShift})
	if s, e := in.selection(); s != 2 || e != 4 {
		t.Fatalf("seleção = [%d,%d), esperado [2,4)", s, e)
	}

	// Seta sem Shift recolhe a seleção para a borda correspondente.
	in.HandleEvent(KeyEvent{Key: KeyLeft})
	if in.hasSelection() || in.Cursor() != 2 {
		t.Fatalf("Left deveria recolher para o início da seleção; cursor=%d", in.Cursor())
	}

	// Digitar substitui a seleção.
	in.HandleEvent(KeyEvent{Key: KeyEnd, Mods: ModShift}) // seleciona "ão"
	typeString(in, "É")
	if in.Text() != "açÉ" {
		t.Fatalf("digitar sobre a seleção: Text() = %q, esperado %q", in.Text(), "açÉ")
	}
	if in.hasSelection() || in.Cursor() != 3 {
		t.Fatalf("após substituir: cursor=%d, seleção=%v", in.Cursor(), in.hasSelection())
	}

	// Ctrl/Cmd+A seleciona tudo; Backspace apaga a seleção inteira.
	in.HandleEvent(KeyEvent{Key: KeyA, Mods: ModControl})
	if s, e := in.selection(); s != 0 || e != 3 {
		t.Fatalf("selecionar tudo = [%d,%d), esperado [0,3)", s, e)
	}
	in.HandleEvent(KeyEvent{Key: KeyBackspace})
	if in.Text() != "" {
		t.Fatalf("Backspace na seleção total deveria esvaziar; Text() = %q", in.Text())
	}
}

func TestInputClipboard(t *testing.T) {
	fake := ""
	clipboardWrite = func(s string) { fake = s }
	clipboardRead = func() string { return fake }
	defer func() { clipboardRead, clipboardWrite = nil, nil }()

	in := NewInput("")
	in.SetTheme(newTestTheme(t))
	typeString(in, "olá mundo")

	// Copiar "olá": Home, 3× Shift+Right, Cmd+C.
	in.HandleEvent(KeyEvent{Key: KeyHome})
	for range 3 {
		in.HandleEvent(KeyEvent{Key: KeyRight, Mods: ModShift})
	}
	in.HandleEvent(KeyEvent{Key: KeyC, Mods: ModSuper})
	if fake != "olá" {
		t.Fatalf("Cmd+C copiou %q, esperado %q", fake, "olá")
	}

	// Colar no fim.
	in.HandleEvent(KeyEvent{Key: KeyEnd})
	in.HandleEvent(KeyEvent{Key: KeyV, Mods: ModSuper})
	if in.Text() != "olá mundoolá" {
		t.Fatalf("após colar: Text() = %q", in.Text())
	}

	// Recortar tudo.
	var changed string
	in.OnChange = func(s string) { changed = s }
	in.HandleEvent(KeyEvent{Key: KeyA, Mods: ModControl})
	in.HandleEvent(KeyEvent{Key: KeyX, Mods: ModControl})
	if fake != "olá mundoolá" || in.Text() != "" || changed != "" {
		t.Fatalf("Cmd+X: clipboard=%q, texto=%q, OnChange=%q", fake, in.Text(), changed)
	}

	// Colar com controle filtrado (campo de linha única).
	fake = "a\nb\tc"
	in.HandleEvent(KeyEvent{Key: KeyV, Mods: ModControl})
	if in.Text() != "abc" {
		t.Fatalf("colar com \\n/\\t deveria filtrar controle; Text() = %q", in.Text())
	}

	// C/X sem seleção e V com clipboard vazio não fazem nada.
	fake = ""
	if in.HandleEvent(KeyEvent{Key: KeyC, Mods: ModControl}) ||
		in.HandleEvent(KeyEvent{Key: KeyX, Mods: ModControl}) ||
		in.HandleEvent(KeyEvent{Key: KeyV, Mods: ModControl}) {
		t.Fatal("C/X sem seleção e V vazio não deveriam ser consumidos")
	}
}

func TestInputSelecaoMouse(t *testing.T) {
	th := newTestTheme(t)
	in := NewInput("")
	in.SetTheme(th)
	in.Layout(image.Rect(0, 0, 300, 32))
	in.SetText("ação teste")
	pad := th.PaddingPx()

	// Down no início e arraste (via captura) até depois de "ação".
	in.HandleEvent(MouseEvent{Kind: MouseDown, Pos: image.Pt(pad, 16), Button: MouseButtonLeft})
	end := pad + th.MeasureString("ação")
	in.HandleEvent(MouseEvent{Kind: MouseMove, Pos: image.Pt(end, 16), Button: MouseButtonLeft})
	if s, e := in.selection(); s != 0 || e != 4 {
		t.Fatalf("seleção por arraste = [%d,%d), esperado [0,4)", s, e)
	}
	in.HandleEvent(MouseEvent{Kind: MouseUp, Pos: image.Pt(end, 16), Button: MouseButtonLeft})

	// Depois de soltar, mover não estende mais.
	in.HandleEvent(MouseEvent{Kind: MouseMove, Pos: image.Pt(pad, 16), Button: MouseButtonLeft})
	if s, e := in.selection(); s != 0 || e != 4 {
		t.Fatalf("seleção mudou após soltar: [%d,%d)", s, e)
	}

	// Novo clique recolhe a seleção.
	in.HandleEvent(MouseEvent{Kind: MouseDown, Pos: image.Pt(pad, 16), Button: MouseButtonLeft})
	if in.hasSelection() {
		t.Fatal("clique simples deveria recolher a seleção")
	}
}
