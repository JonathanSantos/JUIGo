package widget

import (
	"image"
	"testing"
)

func TestGrowDistribuiSobraProporcional(t *testing.T) {
	th := newTestTheme(t)

	a := NewButton("A", nil)
	b := Grow(NewButton("B", nil), 1)
	c := Grow(NewButton("C", nil), 2)
	h := NewHBox(a, b, c).Gap(0).Pad(0)
	Mount(h, th)
	h.Layout(image.Rect(0, 0, 300, 40))

	prefA := a.PreferredSize().X
	leftover := 300 - prefA
	wb := b.Bounds().Dx()
	wc := c.Bounds().Dx()

	if a.Bounds().Dx() != prefA {
		t.Fatalf("filho sem peso deveria ficar na largura preferida (%d); ficou %d", prefA, a.Bounds().Dx())
	}
	// Peso 2 recebe o dobro (±1 de arredondamento) e a soma fecha exata.
	if wb+wc != leftover {
		t.Fatalf("sobras não somam exato: %d+%d != %d", wb, wc, leftover)
	}
	if diff := wc - 2*wb; diff < -2 || diff > 2 {
		t.Fatalf("peso 2 deveria receber ~2× o peso 1: wb=%d, wc=%d", wb, wc)
	}
	// Layout contíguo, terminando na borda.
	if b.Bounds().Min.X != prefA || c.Bounds().Min.X != prefA+wb || c.Bounds().Max.X != 300 {
		t.Fatalf("posições erradas: a=%v b=%v c=%v", a.Bounds(), b.Bounds(), c.Bounds())
	}
	// Sem tipo perdido: Grow devolve o tipo concreto.
	var _ *Button = Grow(NewButton("t", nil), 1)
}

func TestSpacerEmpurraIrmaos(t *testing.T) {
	th := newTestTheme(t)

	txt := NewText("título")
	btn := NewButton("OK", nil)
	h := NewHBox(txt, NewSpacer(), btn).Gap(0).Pad(4)
	Mount(h, th)
	h.Layout(image.Rect(0, 0, 400, 40))

	if btn.Bounds().Max.X != 396 {
		t.Fatalf("Spacer deveria empurrar o botão até a borda (396); Max.X = %d", btn.Bounds().Max.X)
	}
	if txt.Bounds().Min.X != 4 {
		t.Fatalf("texto deveria ficar no início (4); Min.X = %d", txt.Bounds().Min.X)
	}
}

func TestGrowVertical(t *testing.T) {
	th := newTestTheme(t)

	topo := NewButton("topo", nil)
	meio := Grow(NewContainer(), 1) // "conteúdo" que ocupa o resto
	base := NewButton("base", nil)
	v := NewVBox(topo, meio, base).Gap(0).Pad(0)
	Mount(v, th)
	v.Layout(image.Rect(0, 0, 200, 500))

	want := 500 - topo.PreferredSize().Y - base.PreferredSize().Y
	if meio.Bounds().Dy() != want {
		t.Fatalf("filho com Grow deveria ocupar %d de altura; ocupou %d", want, meio.Bounds().Dy())
	}
	if base.Bounds().Max.Y != 500 {
		t.Fatalf("último filho deveria terminar na borda (500); Max.Y = %d", base.Bounds().Max.Y)
	}
}

func TestAlinhamentoTransversal(t *testing.T) {
	th := newTestTheme(t)

	esticado := NewButton("esticado", nil)
	centro := Centered(NewButton("centro", nil))
	inicio := AtStart(NewButton("início", nil))
	fim := AtEnd(NewButton("fim", nil))
	v := NewVBox(esticado, centro, inicio, fim).Gap(0).Pad(0)
	Mount(v, th)
	v.Layout(image.Rect(0, 0, 400, 300))

	if esticado.Bounds().Dx() != 400 {
		t.Fatalf("padrão deveria esticar (400); Dx = %d", esticado.Bounds().Dx())
	}
	cw := centro.PreferredSize().X
	if centro.Bounds().Min.X != (400-cw)/2 || centro.Bounds().Dx() != cw {
		t.Fatalf("Centered: bounds = %v, esperado largura %d centralizada", centro.Bounds(), cw)
	}
	if inicio.Bounds().Min.X != 0 {
		t.Fatalf("AtStart: Min.X = %d, esperado 0", inicio.Bounds().Min.X)
	}
	if fim.Bounds().Max.X != 400 {
		t.Fatalf("AtEnd: Max.X = %d, esperado 400", fim.Bounds().Max.X)
	}
}
