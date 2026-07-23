package widget

import (
	"image"
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

// TestSetTextDamage garante que setters reportam DANO PRECISO (os próprios
// bounds), apenas quando algo muda.
func TestSetTextDamage(t *testing.T) {
	var danos []image.Rectangle
	hooks.Damage = func(r image.Rectangle) { danos = append(danos, r) }
	defer func() { hooks.Damage = nil }()

	txt := NewText("a")
	txt.Layout(image.Rect(10, 20, 110, 40))
	danos = nil // ignora o dano do próprio Layout (diff de bounds)

	txt.SetText("b")
	txt.SetText("b") // sem mudança: não deve danificar
	if len(danos) != 1 || danos[0] != image.Rect(10, 20, 110, 40) {
		t.Fatalf("danos = %v, esperado só os bounds do widget", danos)
	}
}
