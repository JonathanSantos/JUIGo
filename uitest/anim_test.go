package uitest_test

import (
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/anim"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestAnimacaoComRelogioVirtual prova a promessa central do anim: a mesma
// animação que roda no App real avança deterministicamente com h.Advance,
// atualizando a interface pelos bindings — sem um único sleep.
func TestAnimacaoComRelogioVirtual(t *testing.T) {
	progresso := juigo.NewState(0.0)
	barra := juigo.NewProgressBar(0, 1).BindValue(progresso)
	h := uitest.New(t, juigo.NewVBox(barra).Pad(8), 300, 60)
	th := h.Session().Theme()

	a := anim.Tween(progresso, 1, 320*time.Millisecond, anim.Linear)
	concluida := false
	a.OnDone = func() { concluida = true }

	// Metade do tempo (10 quadros de 16ms): linear → exatamente 0,5.
	h.Advance(160 * time.Millisecond)
	if progresso.Get() != 0.5 {
		t.Fatalf("na metade, progresso = %v, esperado 0.5", progresso.Get())
	}
	if concluida || !a.Running() {
		t.Fatal("não deveria ter concluído na metade")
	}
	// E a interface reflete: a barra está ~meio preenchida.
	img := h.Screenshot()
	b := barra.Bounds()
	if img.RGBAAt(b.Min.X+b.Dx()/4, b.Min.Y+b.Dy()/2) != th.Accent {
		t.Fatal("o quarto inicial da barra deveria estar preenchido")
	}
	if img.RGBAAt(b.Min.X+3*b.Dx()/4, b.Min.Y+b.Dy()/2) != th.InputBorder {
		t.Fatal("o quarto final da barra ainda deveria estar vazio")
	}

	// O resto do tempo: completa exatamente no alvo e dispara OnDone.
	h.Advance(160 * time.Millisecond)
	if progresso.Get() != 1 || !concluida || a.Running() {
		t.Fatalf("final: valor=%v concluida=%v", progresso.Get(), concluida)
	}
	img = h.Screenshot()
	if img.RGBAAt(b.Min.X+3*b.Dx()/4, b.Min.Y+b.Dy()/2) != th.Accent {
		t.Fatal("ao concluir, a barra deveria estar cheia")
	}
}
