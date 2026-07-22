package juigo

import "testing"

func TestStateSetWatchMap(t *testing.T) {
	s := NewState(1)
	if s.Get() != 1 {
		t.Fatalf("Get() = %d, esperado 1", s.Get())
	}

	var seen []int
	s.Watch(func(v int) { seen = append(seen, v) })
	s.Watch(func(v int) { seen = append(seen, v*10) })

	dobro := Map(s, func(v int) int { return v * 2 })
	if dobro.Get() != 2 {
		t.Fatalf("Map inicial = %d, esperado 2", dobro.Get())
	}

	s.Set(3)
	if s.Get() != 3 || dobro.Get() != 6 {
		t.Fatalf("após Set(3): Get()=%d, dobro=%d; esperado 3 e 6", s.Get(), dobro.Get())
	}
	// Watchers notificados sincronamente, na ordem de registro.
	if len(seen) != 2 || seen[0] != 3 || seen[1] != 30 {
		t.Fatalf("watchers = %v, esperado [3 30]", seen)
	}
}

func TestBindText(t *testing.T) {
	th := newTestTheme(t)
	s := NewState("inicial")
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
	s := NewState("oi")
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
	in.HandleEvent(KeyEvent{Key: KeyBackspace})
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

// TestRepaintHook garante que Set/SetText agendam redesenho pelo hook do App
// e que tudo continua seguro sem aplicação em execução (hook nil).
func TestRepaintHook(t *testing.T) {
	repaints := 0
	repaintHook = func() { repaints++ }
	defer func() { repaintHook = nil }()

	s := NewState(0)
	s.Set(1)
	txt := NewText("a")
	txt.SetText("b")
	txt.SetText("b") // sem mudança: não deve repintar
	if repaints != 2 {
		t.Fatalf("repaints = %d, esperado 2 (Set e um SetText com mudança)", repaints)
	}

	repaintHook = nil
	s.Set(2) // não deve entrar em pânico sem app
}
