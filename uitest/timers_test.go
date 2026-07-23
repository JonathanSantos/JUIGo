package uitest_test

import (
	"testing"
	"time"

	"juigo"
	"juigo/uitest"
)

// TestAfterEEvery cobre os timers públicos das aplicações no relógio
// virtual: After dispara uma vez (e cancela), Every repete até parar.
func TestAfterEEvery(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("raiz")), 200, 100)

	disparos := 0
	juigo.After(100*time.Millisecond, func() { disparos++ })
	h.Advance(50 * time.Millisecond)
	if disparos != 0 {
		t.Fatal("After não deveria disparar antes do prazo")
	}
	h.Advance(60 * time.Millisecond)
	if disparos != 1 {
		t.Fatalf("After deveria disparar exatamente 1 vez; got %d", disparos)
	}
	h.Advance(200 * time.Millisecond)
	if disparos != 1 {
		t.Fatal("After não deveria repetir")
	}

	// After cancelado não dispara.
	cancelado := 0
	cancela := juigo.After(100*time.Millisecond, func() { cancelado++ })
	cancela()
	h.Advance(200 * time.Millisecond)
	if cancelado != 0 {
		t.Fatal("After cancelado não deveria disparar")
	}

	// Every repete até parar.
	tiques := 0
	para := juigo.Every(100*time.Millisecond, func() { tiques++ })
	h.Advance(350 * time.Millisecond)
	if tiques != 3 {
		t.Fatalf("Every deveria ter 3 tiques em 350ms; got %d", tiques)
	}
	para()
	h.Advance(300 * time.Millisecond)
	if tiques != 3 {
		t.Fatalf("Every parado não deveria seguir; got %d", tiques)
	}
}
