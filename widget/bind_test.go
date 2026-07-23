package widget

import (
	"testing"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/state"
)

func TestBindText(t *testing.T) {
	th := newTestTheme(t)
	s := state.New("inicial")
	txt := NewText("").BindText(s)
	txt.SetTheme(th)

	if txt.Text() != "inicial" {
		t.Fatalf("BindText deveria adotar o valor atual; Text() = %q", txt.Text())
	}
	s.Set("mudou")
	if txt.Text() != "mudou" {
		t.Fatalf("após Set, Text() = %q, esperado %q", txt.Text(), "mudou")
	}
}

func TestBindValueDuasVias(t *testing.T) {
	th := newTestTheme(t)
	s := state.New("oi")
	in := NewInput("")
	in.SetTheme(th)
	in.BindValue(s)

	if in.Text() != "oi" {
		t.Fatalf("BindValue deveria adotar o valor atual; Text() = %q", in.Text())
	}

	// Edição do usuário → State.
	typeString(in, "á")
	if s.Get() != "oiá" {
		t.Fatalf("edição não propagou ao State: Get() = %q, esperado %q", s.Get(), "oiá")
	}
	in.HandleEvent(event.KeyEvent{Key: event.KeyBackspace})
	if s.Get() != "oi" {
		t.Fatalf("backspace não propagou ao State: Get() = %q, esperado %q", s.Get(), "oi")
	}

	// Set externo → campo (cursor vai para o fim).
	s.Set("reescrito")
	if in.Text() != "reescrito" {
		t.Fatalf("Set externo não atualizou o campo: Text() = %q", in.Text())
	}
	if in.Cursor() != len([]rune("reescrito")) {
		t.Fatalf("cursor após Set externo = %d, esperado fim (%d)", in.Cursor(), len([]rune("reescrito")))
	}

	// O binding não engole o OnChange do usuário.
	var last string
	in.OnChange = func(v string) { last = v }
	typeString(in, "!")
	if last != "reescrito!" || s.Get() != "reescrito!" {
		t.Fatalf("OnChange=%q, State=%q; esperado ambos %q", last, s.Get(), "reescrito!")
	}
}

// TestSetTextRepaint garante que setters de widgets agendam redesenho pelo
// hook, apenas quando algo muda.
func TestSetTextRepaint(t *testing.T) {
	repaints := 0
	hooks.Repaint = func() { repaints++ }
	defer func() { hooks.Repaint = nil }()

	txt := NewText("a")
	txt.SetText("b")
	txt.SetText("b") // sem mudança: não deve repintar
	if repaints != 1 {
		t.Fatalf("repaints = %d, esperado 1 (um SetText com mudança)", repaints)
	}
}
