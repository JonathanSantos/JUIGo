package juigo

import "testing"

// TestTemaEscala cobre o contrato HiDPI do tema: campos lógicos intactos,
// conversão Px e métricas de fonte acompanhando a escala.
func TestTemaEscala(t *testing.T) {
	th := newTestTheme(t)
	if th.Scale() != 1 {
		t.Fatalf("Scale() inicial = %v, esperado 1", th.Scale())
	}
	lh1 := th.LineHeight()
	w1 := th.MeasureString("Ação métrica")

	if err := th.SetScale(2); err != nil {
		t.Fatalf("SetScale(2): %v", err)
	}

	if th.Scale() != 2 {
		t.Fatalf("Scale() = %v, esperado 2", th.Scale())
	}
	if th.Padding != 8 {
		t.Fatalf("campo lógico Padding mudou para %d; deveria permanecer 8", th.Padding)
	}
	if got := th.Px(8); got != 16 {
		t.Fatalf("Px(8) na escala 2 = %d, esperado 16", got)
	}
	if got := th.PaddingPx(); got != 16 {
		t.Fatalf("PaddingPx() na escala 2 = %d, esperado 16", got)
	}
	if got := th.InputMinWidthPx(); got != 440 {
		t.Fatalf("InputMinWidthPx() na escala 2 = %d, esperado 440", got)
	}

	// A fonte é re-rasterizada: métricas dobram (tolerância pelo hinting,
	// que arredonda avanço por glyph).
	lh2 := th.LineHeight()
	if lh2 < 2*lh1-2 || lh2 > 2*lh1+2 {
		t.Fatalf("LineHeight na escala 2 = %d, esperado ~%d", lh2, 2*lh1)
	}
	w2 := th.MeasureString("Ação métrica")
	if w2 < 2*w1-8 || w2 > 2*w1+8 {
		t.Fatalf("MeasureString na escala 2 = %d, esperado ~%d", w2, 2*w1)
	}

	// PreferredSize dos widgets acompanha a escala.
	b := NewButton("OK", nil)
	b.SetTheme(th)
	if got := b.PreferredSize().Y; got != lh2+2*16 {
		t.Fatalf("Button.PreferredSize().Y = %d, esperado %d", got, lh2+2*16)
	}
	in := NewInput("")
	in.SetTheme(th)
	if got := in.PreferredSize().X; got != 440 {
		t.Fatalf("Input.PreferredSize().X = %d, esperado 440", got)
	}

	if err := th.SetScale(0); err == nil {
		t.Fatal("SetScale(0) deveria devolver erro")
	}
}
