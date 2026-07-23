package state

import "testing"

func TestHistoryUndoRedo(t *testing.T) {
	h := NewHistory(1)
	if h.CanUndo().Get() || h.CanRedo().Get() {
		t.Fatal("histórico novo não deveria ter undo nem redo")
	}

	h.Commit(2)
	h.Commit(3)
	if !h.CanUndo().Get() {
		t.Fatal("após Commit deveria haver undo")
	}
	if v, ok := h.Undo(); !ok || v != 2 {
		t.Fatalf("Undo deveria voltar a 2; got %v %v", v, ok)
	}
	if !h.CanRedo().Get() {
		t.Fatal("após Undo deveria haver redo")
	}
	if v, ok := h.Redo(); !ok || v != 3 {
		t.Fatalf("Redo deveria voltar a 3; got %v %v", v, ok)
	}

	// Commit após Undo descarta o futuro.
	h.Undo()
	h.Commit(9)
	if h.CanRedo().Get() {
		t.Fatal("Commit deveria descartar o refazer")
	}
	if v, ok := h.Undo(); !ok || v != 2 {
		t.Fatalf("Undo deveria voltar ao ponto anterior ao Commit; got %v %v", v, ok)
	}

	// CommitFrom registra o estado anterior de uma edição no lugar.
	h2 := NewHistory(10)
	h2.Replace(42) // ajuste ao vivo, sem histórico
	h2.CommitFrom(10)
	if v, ok := h2.Undo(); !ok || v != 10 {
		t.Fatalf("CommitFrom deveria permitir desfazer para 10; got %v %v", v, ok)
	}
	if v, ok := h2.Redo(); !ok || v != 42 {
		t.Fatalf("Redo deveria restaurar 42; got %v %v", v, ok)
	}

	if Not(h2.CanUndo()).Get() != !h2.CanUndo().Get() {
		t.Fatal("Not deveria inverter o estado")
	}
}
