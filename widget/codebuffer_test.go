package widget

import "testing"

// TestCodeBufferEdicao cobre as primitivas: inserção simples e multilinha,
// remoção na mesma linha e atravessando linhas, e o texto completo.
func TestCodeBufferEdicao(t *testing.T) {
	b := newCodeBuffer()
	fim := b.insert(textPos{0, 0}, "hello mundo", editOther)
	if fim != (textPos{0, 11}) || b.String() != "hello mundo" {
		t.Fatalf("inserção simples: fim=%v texto=%q", fim, b.String())
	}

	fim = b.insert(textPos{0, 5}, ",\nbrave\nnew", editOther)
	if fim != (textPos{2, 3}) {
		t.Fatalf("inserção multilinha: fim=%v", fim)
	}
	if b.String() != "hello,\nbrave\nnew mundo" || b.count() != 3 {
		t.Fatalf("texto após multilinha: %q (%d linhas)", b.String(), b.count())
	}

	b.deleteRange(textPos{0, 5}, textPos{2, 3}, editOther)
	if b.String() != "hello mundo" || b.count() != 1 {
		t.Fatalf("remoção atravessando linhas: %q", b.String())
	}

	b.deleteRange(textPos{0, 0}, textPos{0, 6}, editOther)
	if b.String() != "mundo" {
		t.Fatalf("remoção na linha: %q", b.String())
	}

	if got := b.textRange(textPos{0, 1}, textPos{0, 4}); got != "und" {
		t.Fatalf("textRange: %q", got)
	}
}

// TestCodeBufferUndoCoalescido: digitação corrida desfaz de uma vez;
// mover o cursor (breakGroup) separa grupos; Enter e colar abrem grupo
// próprio; redo refaz e uma edição nova o descarta.
func TestCodeBufferUndoCoalescido(t *testing.T) {
	b := newCodeBuffer()
	pos := textPos{}
	for _, r := range "abc" {
		pos = b.insert(pos, string(r), editType)
	}
	b.breakGroup()
	for _, r := range "de" {
		pos = b.insert(pos, string(r), editType)
	}
	if b.String() != "abcde" || len(b.undo) != 2 {
		t.Fatalf("dois grupos esperados; texto=%q grupos=%d", b.String(), len(b.undo))
	}

	caret, _, ok := b.undoStep()
	if !ok || b.String() != "abc" || caret != (textPos{0, 3}) {
		t.Fatalf("undo do 2º grupo: texto=%q caret=%v", b.String(), caret)
	}
	caret, _, _ = b.undoStep()
	if b.String() != "" || caret != (textPos{0, 0}) {
		t.Fatalf("undo do 1º grupo: texto=%q caret=%v", b.String(), caret)
	}

	caret, _, _ = b.redoStep()
	if b.String() != "abc" || caret != (textPos{0, 3}) {
		t.Fatalf("redo: texto=%q caret=%v", b.String(), caret)
	}

	// Edição nova descarta o redo pendente.
	b.insert(textPos{0, 3}, "!", editType)
	if _, _, ok := b.redoStep(); ok {
		t.Fatal("edição nova deveria descartar o redo")
	}

	// Backspaces em sequência coalescem num grupo só.
	b2 := newCodeBuffer()
	b2.insert(textPos{}, "xyz", editOther)
	b2.deleteRange(textPos{0, 2}, textPos{0, 3}, editDelBack)
	b2.deleteRange(textPos{0, 1}, textPos{0, 2}, editDelBack)
	if len(b2.undo) != 2 { // "xyz" + o grupo dos backspaces
		t.Fatalf("backspaces deveriam coalescer; grupos=%d", len(b2.undo))
	}
	caret, _, _ = b2.undoStep()
	if b2.String() != "xyz" || caret != (textPos{0, 3}) {
		t.Fatalf("undo dos backspaces: texto=%q caret=%v", b2.String(), caret)
	}

	// Enter (multilinha) nunca adere ao grupo de digitação.
	b3 := newCodeBuffer()
	p := b3.insert(textPos{}, "a", editType)
	p = b3.insert(p, "\n", editOther)
	b3.insert(p, "b", editType)
	if len(b3.undo) != 3 {
		t.Fatalf("Enter deveria abrir grupo próprio; grupos=%d", len(b3.undo))
	}
	b3.undoStep()
	if b3.String() != "a\n" {
		t.Fatalf("undo do 'b': %q", b3.String())
	}
	b3.undoStep()
	if b3.String() != "a" || b3.count() != 1 {
		t.Fatalf("undo do Enter: %q", b3.String())
	}
}

// TestCodeBufferSetText zera o histórico e reindexa as linhas.
func TestCodeBufferSetText(t *testing.T) {
	b := newCodeBuffer()
	b.insert(textPos{}, "velho", editType)
	b.setText("um\ndois\ntrês")
	if b.count() != 3 || b.lineText(1) != "dois" || b.String() != "um\ndois\ntrês" {
		t.Fatalf("setText: %q (%d linhas)", b.String(), b.count())
	}
	if _, _, ok := b.undoStep(); ok {
		t.Fatal("setText deveria zerar o undo")
	}
	if b.clamp(textPos{5, 99}) != (textPos{2, 4}) {
		t.Fatalf("clamp: %v", b.clamp(textPos{5, 99}))
	}
}
