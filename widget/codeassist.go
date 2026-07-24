package widget

// Assistências de edição do CodeEditor (fase 3): auto-indentação,
// indentação de bloco e par de parênteses.

// autoIndent devolve a indentação que uma linha nova deve herdar: os
// espaços/tabs iniciais da linha do cursor (até o cursor — Enter no meio da
// indentação não a duplica), mais um tab quando o caractere anterior abre
// um bloco ({, ( ou [).
func (c *CodeEditor) autoIndent() string {
	pos := c.cursor
	if c.hasSelection() {
		pos, _ = c.selectionRange()
	}
	line := c.buf.lines[pos.Line].runes
	n := 0
	for n < len(line) && n < pos.Col && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	ind := string(line[:n])
	if pos.Col > 0 && pos.Col <= len(line) {
		switch line[pos.Col-1] {
		case '{', '(', '[':
			ind += "\t"
		}
	}
	return ind
}

// spansLines informa se a seleção atravessa mais de uma linha.
func (c *CodeEditor) spansLines() bool {
	s, e := c.selectionRange()
	return s.Line != e.Line
}

// outdentWidth devolve quantas runes iniciais da linha i um Shift+Tab
// remove: um tab, ou até tabCols espaços.
func (c *CodeEditor) outdentWidth(i int) int {
	runes := c.buf.lines[i].runes
	if len(runes) == 0 {
		return 0
	}
	if runes[0] == '\t' {
		return 1
	}
	n := 0
	for n < len(runes) && n < c.tabCols && runes[n] == ' ' {
		n++
	}
	return n
}

// indentBlock indenta (ou remove a indentação de) todas as linhas da
// seleção — ou a do cursor — como UM único passo de undo (lote). A seleção
// que termina no início de uma linha não a inclui, como nos editores.
func (c *CodeEditor) indentBlock(outdent bool) {
	start, end := c.selectionRange()
	first, last := start.Line, end.Line
	if last > first && end.Col == 0 {
		last--
	}
	deltaCursor, deltaAnchor := 0, 0
	c.buf.beginBatch()
	changed := false
	for i := first; i <= last; i++ {
		var d int
		if outdent {
			n := c.outdentWidth(i)
			if n == 0 {
				continue
			}
			c.buf.deleteRange(textPos{i, 0}, textPos{i, n}, editOther)
			d = -n
		} else {
			c.buf.insert(textPos{i, 0}, "\t", editOther)
			d = 1
		}
		changed = true
		if i == c.cursor.Line {
			deltaCursor = d
		}
		if i == c.anchor.Line {
			deltaAnchor = d
		}
	}
	c.buf.endBatch()
	if !changed {
		return
	}
	c.cursor.Col = max(0, c.cursor.Col+deltaCursor)
	c.anchor.Col = max(0, c.anchor.Col+deltaAnchor)
	c.cursor = c.buf.clamp(c.cursor)
	c.anchor = c.buf.clamp(c.anchor)
	c.goalX = -1
	c.maxDirty = true
	c.relexFrom(first)
	c.restartBlink()
	c.updateBrackets()
	c.ensureVisible()
	c.Invalidate()
	c.emitChange()
}

// bracketScanLimit limita a busca do par a esta distância em linhas — um
// abridor sem fecho num arquivo enorme não custa o arquivo inteiro.
const bracketScanLimit = 4000

// updateBrackets recalcula o par de parênteses sob o cursor e danifica as
// linhas dos realces que mudaram. Chamado a cada movimento/edição/foco.
func (c *CodeEditor) updateBrackets() {
	oldOK, oldA, oldB := c.brOK, c.brA, c.brB
	c.brOK = false
	if c.focused && !c.hasSelection() && c.buf != nil {
		if a, b, ok := c.findBracketPair(); ok {
			c.brA, c.brB, c.brOK = a, b, true
		}
	}
	if oldOK == c.brOK && oldA == c.brA && oldB == c.brB {
		return
	}
	if oldOK {
		c.damageLine(oldA.Line)
		c.damageLine(oldB.Line)
	}
	if c.brOK {
		c.damageLine(c.brA.Line)
		c.damageLine(c.brB.Line)
	}
}

// bracketPairOf devolve o par e a direção de um caractere de abertura ou
// fechamento; ok=false para os demais.
func bracketPairOf(r rune) (open, close rune, forward, ok bool) {
	switch r {
	case '(':
		return '(', ')', true, true
	case '[':
		return '[', ']', true, true
	case '{':
		return '{', '}', true, true
	case ')':
		return '(', ')', false, true
	case ']':
		return '[', ']', false, true
	case '}':
		return '{', '}', false, true
	}
	return 0, 0, false, false
}

// findBracketPair procura um bracket ADJACENTE ao cursor (o anterior tem
// preferência, como nos editores) e o seu par correspondente.
func (c *CodeEditor) findBracketPair() (textPos, textPos, bool) {
	line := c.buf.lines[c.cursor.Line].runes
	try := []textPos{}
	if c.cursor.Col > 0 {
		try = append(try, textPos{c.cursor.Line, c.cursor.Col - 1})
	}
	if c.cursor.Col < len(line) {
		try = append(try, textPos{c.cursor.Line, c.cursor.Col})
	}
	for _, p := range try {
		r := c.buf.lines[p.Line].runes[p.Col]
		open, close, forward, ok := bracketPairOf(r)
		if !ok || c.isCodeExempt(p.Line, p.Col) {
			continue
		}
		if m, found := c.matchBracket(p, open, close, forward); found {
			return p, m, true
		}
	}
	return textPos{}, textPos{}, false
}

// isCodeExempt informa se a posição está em string ou comentário (pelo
// highlight, quando ativo) — brackets ali não contam para o pareamento.
func (c *CodeEditor) isCodeExempt(line, col int) bool {
	st := c.styleAt(line, col)
	return st == SyntaxString || st == SyntaxComment
}

// styleAt devolve a classe léxica da rune na posição (linha, coluna), pelo
// cache de spans; SyntaxText sem highlighter.
func (c *CodeEditor) styleAt(line, col int) SyntaxStyle {
	l := &c.buf.lines[line]
	if c.hl == nil || !l.hlOK {
		return SyntaxText
	}
	off := byteOffset(c.buf.lineText(line), col)
	acc := 0
	for _, sp := range l.spans {
		acc += sp.Len
		if off < acc {
			return sp.Style
		}
	}
	return SyntaxText
}

// matchBracket procura o par de p contando profundidade na direção dada,
// pulando brackets em strings/comentários, até bracketScanLimit linhas.
func (c *CodeEditor) matchBracket(p textPos, open, close rune, forward bool) (textPos, bool) {
	depth := 0
	if forward {
		limite := p.Line + bracketScanLimit
		for li := p.Line; li < c.buf.count() && li <= limite; li++ {
			runes := c.buf.lines[li].runes
			start := 0
			if li == p.Line {
				start = p.Col
			}
			for ci := start; ci < len(runes); ci++ {
				r := runes[ci]
				if r != open && r != close {
					continue
				}
				if !(li == p.Line && ci == p.Col) && c.isCodeExempt(li, ci) {
					continue
				}
				if r == open {
					depth++
				} else if depth--; depth == 0 {
					return textPos{li, ci}, true
				}
			}
		}
		return textPos{}, false
	}
	limite := p.Line - bracketScanLimit
	for li := p.Line; li >= 0 && li >= limite; li-- {
		runes := c.buf.lines[li].runes
		start := len(runes) - 1
		if li == p.Line {
			start = p.Col
		}
		for ci := start; ci >= 0; ci-- {
			r := runes[ci]
			if r != open && r != close {
				continue
			}
			if !(li == p.Line && ci == p.Col) && c.isCodeExempt(li, ci) {
				continue
			}
			if r == close {
				depth++
			} else if depth--; depth == 0 {
				return textPos{li, ci}, true
			}
		}
	}
	return textPos{}, false
}
