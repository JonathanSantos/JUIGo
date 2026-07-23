package state

import (
	"testing"

	"juigo/internal/hooks"
)

func TestStateSetWatchMap(t *testing.T) {
	s := New(1)
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

// TestRepaintHook garante que Set agenda redesenho pelo hook do App e que
// tudo continua seguro sem aplicação em execução (hook nil).
func TestRepaintHook(t *testing.T) {
	repaints := 0
	hooks.Repaint = func() { repaints++ }
	defer func() { hooks.Repaint = nil }()

	s := New(0)
	s.Set(1)
	if repaints != 1 {
		t.Fatalf("repaints = %d, esperado 1", repaints)
	}

	hooks.Repaint = nil
	s.Set(2) // não deve entrar em pânico sem app
}
