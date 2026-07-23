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

// TestFrameHook garante que Set agenda um FRAME (sem dano — as regiões vêm
// dos setters dos widgets) e que tudo continua seguro sem aplicação.
func TestFrameHook(t *testing.T) {
	frames := 0
	hooks.Frame = func() { frames++ }
	defer func() { hooks.Frame = nil }()

	s := New(0)
	s.Set(1)
	if frames != 1 {
		t.Fatalf("frames = %d, esperado 1", frames)
	}

	hooks.Frame = nil
	s.Set(2) // não deve entrar em pânico sem app
}

func TestCombine(t *testing.T) {
	a := New(2)
	b := New(3)
	soma := Combine(func() int { return a.Get() + b.Get() }, a, b)
	if soma.Get() != 5 {
		t.Fatalf("Combine inicial = %d, esperado 5", soma.Get())
	}
	a.Set(10)
	if soma.Get() != 13 {
		t.Fatalf("após a.Set: %d, esperado 13", soma.Get())
	}
	b.Set(-3)
	if soma.Get() != 7 {
		t.Fatalf("após b.Set: %d, esperado 7", soma.Get())
	}
}
