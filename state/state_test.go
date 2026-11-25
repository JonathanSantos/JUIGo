package state

import (
	"runtime"
	"testing"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
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

func TestWatchCancel(t *testing.T) {
	s := New(0)
	calls := 0
	cancel := s.Watch(func(int) { calls++ })
	s.Set(1)
	cancel()
	cancel() // idempotente
	s.Set(2)
	if calls != 1 {
		t.Fatalf("chamadas = %d, esperado 1", calls)
	}
	if s.Watchers() != 0 {
		t.Fatalf("Watchers() = %d, esperado 0", s.Watchers())
	}
}

// Cancelar (a si mesmo ou a outro) durante a notificação não pula nem
// repete observadores, e o cancelado deixa de ser chamado na hora.
func TestWatchCancelDuranteNotificacao(t *testing.T) {
	s := New(0)
	var seen []string
	var cancelB func()
	s.Watch(func(int) {
		seen = append(seen, "a")
		cancelB()
	})
	cancelB = s.Watch(func(int) { seen = append(seen, "b") })
	s.Watch(func(int) { seen = append(seen, "c") })

	s.Set(1)
	s.Set(2)
	want := []string{"a", "c", "a", "c"}
	if len(seen) != len(want) {
		t.Fatalf("seen = %v, esperado %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("seen = %v, esperado %v", seen, want)
		}
	}
	if s.Watchers() != 2 {
		t.Fatalf("Watchers() = %d, esperado 2", s.Watchers())
	}
}

// Um observador registrado durante a notificação só vale no próximo Set.
func TestWatchRegistradoDuranteNotificacao(t *testing.T) {
	s := New(0)
	late := 0
	added := false
	s.Watch(func(int) {
		if !added {
			added = true
			s.Watch(func(int) { late++ })
		}
	})
	s.Set(1)
	if late != 0 {
		t.Fatalf("o observador novo rodou no mesmo Set (%d)", late)
	}
	s.Set(2)
	if late != 1 {
		t.Fatalf("late = %d, esperado 1", late)
	}
}

// owner é o dono de teste das assinaturas fracas. O enchimento o tira do
// alocador de objetos minúsculos do runtime: objetos de até 16 bytes sem
// ponteiros dividem um bloco, e um vizinho vivo manteria o bloco (e a
// referência fraca) de pé — o que fazia este teste falhar de vez em quando.
// Widgets reais são grandes e não passam por esse caminho.
type owner struct {
	got int
	_   [32]byte
}

func TestWatchWeak(t *testing.T) {
	s := New(0)
	o := &owner{}
	WatchWeak(s, o, func(o *owner, v int) { o.got = v })
	s.Set(7)
	if o.got != 7 {
		t.Fatalf("got = %d, esperado 7", o.got)
	}

	// Sem referências fortes, o dono é coletado e a assinatura se desfaz.
	func() {
		tmp := &owner{}
		WatchWeak(s, tmp, func(o *owner, v int) { o.got = v })
	}()
	if s.Watchers() != 2 {
		t.Fatalf("Watchers() = %d, esperado 2", s.Watchers())
	}
	runtime.GC()
	runtime.GC()
	s.Set(8)
	if s.Watchers() != 1 {
		t.Fatalf("após a coleta: Watchers() = %d, esperado 1", s.Watchers())
	}
	if o.got != 8 {
		t.Fatalf("o dono vivo deveria seguir recebendo; got = %d", o.got)
	}
	runtime.KeepAlive(o)
}
