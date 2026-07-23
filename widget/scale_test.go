package widget

import "testing"

// TestPreferredSizeAcompanhaEscala garante que os tamanhos preferidos dos
// widgets usam as métricas escaladas do tema (contrato HiDPI).
func TestPreferredSizeAcompanhaEscala(t *testing.T) {
	th := newTestTheme(t)
	if err := th.SetScale(2); err != nil {
		t.Fatalf("SetScale(2): %v", err)
	}

	b := NewButton("OK", nil)
	b.SetTheme(th)
	if got := b.PreferredSize().Y; got != th.LineHeight()+2*16 {
		t.Fatalf("Button.PreferredSize().Y = %d, esperado %d", got, th.LineHeight()+2*16)
	}
	in := NewInput("")
	in.SetTheme(th)
	if got := in.PreferredSize().X; got != 440 {
		t.Fatalf("Input.PreferredSize().X = %d, esperado 440", got)
	}
}
