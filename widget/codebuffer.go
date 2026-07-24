package widget

import "strings"

// textPos é uma posição no codeBuffer: linha e coluna em RUNES (a coluna
// len(linha) é o fim da linha, antes do '\n' implícito).
type textPos struct {
	Line, Col int
}

// before informa se p vem antes de q na ordem do texto.
func (p textPos) before(q textPos) bool {
	return p.Line < q.Line || (p.Line == q.Line && p.Col < q.Col)
}

// editKind classifica uma edição para a COALESCÊNCIA do undo: digitação
// contígua vira um grupo só, assim como Backspaces e Deletes em sequência;
// qualquer outra coisa (colar, Enter, substituir seleção) abre grupo novo.
type editKind int

const (
	editOther editKind = iota
	editType
	editDelBack
	editDelFwd
)

// editOp é uma operação primitiva REVERSÍVEL: inserir text em start
// (terminando em end) ou remover o intervalo [start,end) cujo conteúdo era
// text. Desfazer aplica a inversa.
type editOp struct {
	insert     bool
	start, end textPos
	text       string
}

// editGroup é um passo de undo/redo: as operações de uma edição do usuário
// (possivelmente coalescidas).
type editGroup struct {
	ops []editOp
}

// codeLine é uma linha do buffer com os caches dela: a string (para
// desenhar e medir sem alocar por frame), a largura em pixels e o
// highlight (spans + estados de entrada/saída do lexer), mantidos pelo
// widget (flags *OK falsas = recalcular).
type codeLine struct {
	runes   []rune
	text    string
	textOK  bool
	width   int
	widthOK bool

	spans    []HighlightSpan
	stateIn  HighlightState
	stateOut HighlightState
	hlOK     bool

	// wrapStarts são as colunas onde começam as linhas VISUAIS além da
	// primeira (quando o editor está com WrapLines); wrapOK falso =
	// recalcular.
	wrapStarts []int
	wrapOK     bool
}

// codeBuffer é o modelo de texto do CodeEditor: linhas separadas (edições
// tocam uma linha, Enter/Backspace nas bordas dividem/juntam duas), undo e
// redo por GRUPOS de operações com coalescência de digitação, e caches por
// linha — nada aqui conhece tema ou desenho.
type codeBuffer struct {
	lines []codeLine
	undo  []editGroup
	redo  []editGroup
	// lastKind/lastPos guiam a coalescência: a próxima edição adere ao
	// grupo aberto se for do mesmo tipo e contígua ao fim da anterior.
	lastKind editKind
	lastPos  textPos
	// batching agrupa TODAS as operações até endBatch num único passo de
	// undo (indentação de bloco); batchNew marca a primeira do lote.
	batching bool
	batchNew bool
	// version cresce a cada mutação — invalida caches externos.
	version int
}

// newCodeBuffer cria um buffer com uma linha vazia.
func newCodeBuffer() *codeBuffer {
	return &codeBuffer{lines: make([]codeLine, 1)}
}

// count devolve o número de linhas (sempre ≥ 1).
func (b *codeBuffer) count() int {
	return len(b.lines)
}

// lineLen devolve o comprimento da linha i, em runes.
func (b *codeBuffer) lineLen(i int) int {
	return len(b.lines[i].runes)
}

// lineText devolve a linha i como string, do cache (reconstruída só quando
// a linha mudou).
func (b *codeBuffer) lineText(i int) string {
	l := &b.lines[i]
	if !l.textOK {
		l.text = string(l.runes)
		l.textOK = true
	}
	return l.text
}

// String junta o buffer inteiro com '\n' — para Text()/persistência; O(n),
// não usar em caminho quente.
func (b *codeBuffer) String() string {
	var sb strings.Builder
	for i := range b.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(b.lineText(i))
	}
	return sb.String()
}

// setText substitui todo o conteúdo e zera o histórico de undo/redo.
func (b *codeBuffer) setText(s string) {
	parts := strings.Split(s, "\n")
	b.lines = make([]codeLine, len(parts))
	for i, p := range parts {
		b.lines[i] = codeLine{runes: []rune(p), text: p, textOK: true}
	}
	b.undo = b.undo[:0]
	b.redo = b.redo[:0]
	b.lastKind = editOther
	b.version++
}

// clamp limita a posição ao conteúdo.
func (b *codeBuffer) clamp(p textPos) textPos {
	if p.Line < 0 {
		return textPos{}
	}
	if p.Line >= len(b.lines) {
		last := len(b.lines) - 1
		return textPos{last, len(b.lines[last].runes)}
	}
	if p.Col < 0 {
		p.Col = 0
	}
	if n := len(b.lines[p.Line].runes); p.Col > n {
		p.Col = n
	}
	return p
}

// dirty invalida os caches da linha i (texto, largura, highlight e wrap).
func (b *codeBuffer) dirty(i int) {
	b.lines[i].textOK = false
	b.lines[i].widthOK = false
	b.lines[i].hlOK = false
	b.lines[i].wrapOK = false
}

// insertRaw insere text (pode conter '\n') em pos, sem registrar undo.
// Devolve a posição do fim do texto inserido.
func (b *codeBuffer) insertRaw(pos textPos, text string) textPos {
	b.version++
	segs := strings.Split(text, "\n")
	line := &b.lines[pos.Line]
	if len(segs) == 1 {
		seg := []rune(segs[0])
		line.runes = append(line.runes, seg...) // garante capacidade
		copy(line.runes[pos.Col+len(seg):], line.runes[pos.Col:])
		copy(line.runes[pos.Col:], seg)
		b.dirty(pos.Line)
		return textPos{pos.Line, pos.Col + len(seg)}
	}
	// Multilinha: o resto da linha corrente desce para depois do último
	// segmento; os segmentos do meio viram linhas novas.
	tail := append([]rune(nil), line.runes[pos.Col:]...)
	line.runes = append(line.runes[:pos.Col], []rune(segs[0])...)
	b.dirty(pos.Line)
	novas := make([]codeLine, len(segs)-1)
	for i, seg := range segs[1:] {
		novas[i] = codeLine{runes: []rune(seg)}
	}
	fim := textPos{pos.Line + len(novas), len(novas[len(novas)-1].runes)}
	novas[len(novas)-1].runes = append(novas[len(novas)-1].runes, tail...)
	b.lines = append(b.lines[:pos.Line+1], append(novas, b.lines[pos.Line+1:]...)...)
	return fim
}

// deleteRaw remove o intervalo [start,end) e devolve o texto removido (com
// '\n'), sem registrar undo.
func (b *codeBuffer) deleteRaw(start, end textPos) string {
	b.version++
	if start.Line == end.Line {
		line := &b.lines[start.Line]
		removed := string(line.runes[start.Col:end.Col])
		line.runes = append(line.runes[:start.Col], line.runes[end.Col:]...)
		b.dirty(start.Line)
		return removed
	}
	var sb strings.Builder
	sb.WriteString(string(b.lines[start.Line].runes[start.Col:]))
	for i := start.Line + 1; i < end.Line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(b.lineText(i))
	}
	sb.WriteByte('\n')
	sb.WriteString(string(b.lines[end.Line].runes[:end.Col]))

	first := &b.lines[start.Line]
	first.runes = append(first.runes[:start.Col], b.lines[end.Line].runes[end.Col:]...)
	b.dirty(start.Line)
	b.lines = append(b.lines[:start.Line+1], b.lines[end.Line+1:]...)
	return sb.String()
}

// insert insere text em pos registrando undo (coalescido conforme kind) e
// devolve o fim da inserção.
func (b *codeBuffer) insert(pos textPos, text string, kind editKind) textPos {
	if text == "" {
		return pos
	}
	end := b.insertRaw(pos, text)
	b.record(editOp{insert: true, start: pos, end: end, text: text}, kind, end)
	return end
}

// deleteRange remove [start,end) registrando undo (coalescido conforme
// kind).
func (b *codeBuffer) deleteRange(start, end textPos, kind editKind) {
	if start == end {
		return
	}
	if end.before(start) {
		start, end = end, start
	}
	text := b.deleteRaw(start, end)
	b.record(editOp{insert: false, start: start, end: end, text: text}, kind, start)
}

// replace remove [start,end) e insere text no lugar como UM ÚNICO passo de
// undo (digitar sobre uma seleção, colar sobre uma seleção). Devolve o fim
// do texto inserido.
func (b *codeBuffer) replace(start, end textPos, text string) textPos {
	if end.before(start) {
		start, end = end, start
	}
	b.redo = b.redo[:0]
	var g editGroup
	if start != end {
		removed := b.deleteRaw(start, end)
		g.ops = append(g.ops, editOp{insert: false, start: start, end: end, text: removed})
	}
	fim := start
	if text != "" {
		fim = b.insertRaw(start, text)
		g.ops = append(g.ops, editOp{insert: true, start: start, end: fim, text: text})
	}
	if len(g.ops) > 0 {
		b.undo = append(b.undo, g)
	}
	b.lastKind, b.lastPos = editOther, fim
	return fim
}

// beginBatch abre um LOTE: todas as operações até endBatch entram num
// único grupo de undo (indentação de bloco em várias linhas).
func (b *codeBuffer) beginBatch() {
	b.batching, b.batchNew = true, true
}

// endBatch encerra o lote e a coalescência.
func (b *codeBuffer) endBatch() {
	b.batching = false
	b.lastKind = editOther
}

// record registra a operação no undo, aderindo ao grupo aberto quando a
// edição é do mesmo tipo e contígua (digitação corrida, Backspaces em
// sequência); edições novas descartam o redo.
func (b *codeBuffer) record(op editOp, kind editKind, caretAfter textPos) {
	b.redo = b.redo[:0]
	if b.batching {
		if b.batchNew || len(b.undo) == 0 {
			b.undo = append(b.undo, editGroup{ops: []editOp{op}})
			b.batchNew = false
		} else {
			g := &b.undo[len(b.undo)-1]
			g.ops = append(g.ops, op)
		}
		b.lastKind, b.lastPos = editOther, caretAfter
		return
	}
	adere := kind != editOther && kind == b.lastKind && len(b.undo) > 0
	if adere {
		switch kind {
		case editType:
			adere = op.insert && op.start == b.lastPos && !strings.Contains(op.text, "\n")
		case editDelBack:
			adere = !op.insert && op.end == b.lastPos
		case editDelFwd:
			adere = !op.insert && op.start == b.lastPos
		}
	}
	if adere {
		g := &b.undo[len(b.undo)-1]
		g.ops = append(g.ops, op)
	} else {
		b.undo = append(b.undo, editGroup{ops: []editOp{op}})
	}
	b.lastKind, b.lastPos = kind, caretAfter
}

// breakGroup encerra a coalescência: a próxima edição abre grupo novo
// (chamado quando o cursor se move por conta própria, no blur etc.).
func (b *codeBuffer) breakGroup() {
	b.lastKind = editOther
}

// minLine devolve a menor linha tocada pelas operações do grupo — de onde
// o highlight precisa re-lexar.
func (g editGroup) minLine() int {
	min := 0
	for i, op := range g.ops {
		if i == 0 || op.start.Line < min {
			min = op.start.Line
		}
	}
	return min
}

// undoStep desfaz o último grupo; devolve o cursor resultante e a menor
// linha afetada.
func (b *codeBuffer) undoStep() (textPos, int, bool) {
	if len(b.undo) == 0 {
		return textPos{}, 0, false
	}
	g := b.undo[len(b.undo)-1]
	b.undo = b.undo[:len(b.undo)-1]
	var caret textPos
	for i := len(g.ops) - 1; i >= 0; i-- {
		op := g.ops[i]
		if op.insert {
			b.deleteRaw(op.start, op.end)
			caret = op.start
		} else {
			b.insertRaw(op.start, op.text)
			caret = op.end
		}
	}
	b.redo = append(b.redo, g)
	b.lastKind = editOther
	return caret, g.minLine(), true
}

// redoStep refaz o último grupo desfeito; devolve o cursor resultante e a
// menor linha afetada.
func (b *codeBuffer) redoStep() (textPos, int, bool) {
	if len(b.redo) == 0 {
		return textPos{}, 0, false
	}
	g := b.redo[len(b.redo)-1]
	b.redo = b.redo[:len(b.redo)-1]
	var caret textPos
	for _, op := range g.ops {
		if op.insert {
			b.insertRaw(op.start, op.text)
			caret = op.end
		} else {
			b.deleteRaw(op.start, op.end)
			caret = op.start
		}
	}
	b.undo = append(b.undo, g)
	b.lastKind = editOther
	return caret, g.minLine(), true
}

// textRange devolve o conteúdo do intervalo [start,end) com '\n' — a base
// do copiar/recortar.
func (b *codeBuffer) textRange(start, end textPos) string {
	if end.before(start) {
		start, end = end, start
	}
	if start.Line == end.Line {
		return string(b.lines[start.Line].runes[start.Col:end.Col])
	}
	var sb strings.Builder
	sb.WriteString(string(b.lines[start.Line].runes[start.Col:]))
	for i := start.Line + 1; i < end.Line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(b.lineText(i))
	}
	sb.WriteByte('\n')
	sb.WriteString(string(b.lines[end.Line].runes[:end.Col]))
	return sb.String()
}
