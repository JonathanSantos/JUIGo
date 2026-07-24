package widget

import (
	"image"
	"image/color"
)

// SyntaxStyle é a classe léxica de um trecho de código; a cor vem da
// paleta do tema (Theme.Syntax) — texto comum usa Theme.Text.
type SyntaxStyle uint8

const (
	// SyntaxText é texto comum (identificadores, operadores, espaços).
	SyntaxText SyntaxStyle = iota
	// SyntaxKeyword são as palavras-chave da linguagem.
	SyntaxKeyword
	// SyntaxString são literais de texto (aspas, runas, raw strings).
	SyntaxString
	// SyntaxNumber são literais numéricos.
	SyntaxNumber
	// SyntaxComment são comentários.
	SyntaxComment
	// SyntaxBuiltin são tipos e identificadores embutidos (int, true, len…).
	SyntaxBuiltin
)

// HighlightState é o estado do lexer CARREGADO ENTRE LINHAS (dentro de
// comentário de bloco, de raw string…). Zero é o estado inicial, e a
// comparação por igualdade é o que permite ao re-lex incremental parar:
// quando a linha seguinte já foi lexada com o mesmo estado de entrada,
// nada abaixo muda.
type HighlightState uint32

// HighlightSpan é um trecho contíguo de uma linha com a mesma classe. Len
// é em BYTES da linha; os spans de uma linha a cobrem em sequência.
type HighlightSpan struct {
	Len   int
	Style SyntaxStyle
}

// Highlighter colore UMA LINHA por vez: recebe a linha (sem '\n') e o
// estado herdado da linha anterior, e devolve os spans (cobrindo a linha
// inteira) e o estado a carregar para a próxima. Deve ser uma função pura
// da entrada — o editor a chama no re-lex incremental a cada edição e
// cacheia o resultado por linha. Lexers prontos: juigo/syntax.
type Highlighter interface {
	HighlightLine(line string, state HighlightState) (spans []HighlightSpan, next HighlightState)
}

// Highlight define o highlighter de sintaxe do editor (nil desliga) e
// re-lexa o conteúdo. Encadeável.
func (c *CodeEditor) Highlight(h Highlighter) *CodeEditor {
	c.hl = h
	for i := range c.buf.lines {
		c.buf.lines[i].hlOK = false
	}
	if h != nil {
		c.relexFrom(0)
	}
	c.updateBrackets()
	c.Invalidate()
	return c
}

// relexFrom re-lexa da linha k para baixo, parando quando o estado
// CONVERGE: se a linha seguinte já foi lexada com o mesmo estado de
// entrada, os spans dela (e de tudo abaixo) continuam válidos. Devolve a
// última linha re-lexada (-1 se nada mudou) — abrir um /* no topo cascateia
// até o fim; digitar dentro de uma linha custa uma linha só.
func (c *CodeEditor) relexFrom(k int) int {
	if c.hl == nil {
		return -1
	}
	n := c.buf.count()
	if k < 0 {
		k = 0
	}
	if k >= n {
		k = n - 1
	}
	state := HighlightState(0)
	if k > 0 {
		state = c.buf.lines[k-1].stateOut
	}
	last := -1
	for i := k; i < n; i++ {
		l := &c.buf.lines[i]
		if l.hlOK && l.stateIn == state {
			break
		}
		spans, out := c.hl.HighlightLine(c.buf.lineText(i), state)
		l.spans, l.stateIn, l.stateOut, l.hlOK = spans, state, out, true
		last = i
		state = out
	}
	return last
}

// styleColor resolve a classe léxica na cor da paleta do tema.
func (c *CodeEditor) styleColor(s SyntaxStyle) color.RGBA {
	th := c.theme
	switch s {
	case SyntaxKeyword:
		return th.Syntax.Keyword
	case SyntaxString:
		return th.Syntax.String
	case SyntaxNumber:
		return th.Syntax.Number
	case SyntaxComment:
		return th.Syntax.Comment
	case SyntaxBuiltin:
		return th.Syntax.Builtin
	}
	return th.Text
}

// drawLine desenha a linha i com os spans do highlight (ou texto comum sem
// highlighter), respeitando os tab stops através dos spans.
func (c *CodeEditor) drawLine(view *image.RGBA, i, screenX0, baseline int) {
	line := c.buf.lineText(i)
	l := &c.buf.lines[i]
	if c.hl == nil || !l.hlOK {
		c.drawTabbed(view, line, 0, screenX0, baseline, c.theme.Text)
		return
	}
	x, off := 0, 0
	for _, sp := range l.spans {
		end := off + sp.Len
		if end > len(line) {
			end = len(line)
		}
		if end > off {
			x = c.drawTabbed(view, line[off:end], x, screenX0, baseline, c.styleColor(sp.Style))
		}
		off = end
	}
	if off < len(line) {
		// Lexer não cobriu a linha inteira: o resto sai como texto comum.
		c.drawTabbed(view, line[off:], x, screenX0, baseline, c.theme.Text)
	}
}
