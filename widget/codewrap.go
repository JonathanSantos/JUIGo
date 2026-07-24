package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
)

// Quebra visual de linhas do CodeEditor: com WrapLines ligado nada rola na
// horizontal — cada linha lógica ocupa quantas linhas VISUAIS precisar,
// quebradas por célula (tab stops reiniciam em cada linha visual). Sem
// wrap, linha visual == linha lógica e todos os helpers abaixo degradam
// para a identidade.

// WrapLines liga/desliga a quebra visual. Padrão: DESLIGADA (o editor rola
// na horizontal, como editores de código costumam fazer). Encadeável.
func (c *CodeEditor) WrapLines(on bool) *CodeEditor {
	if c.wrap == on {
		return c
	}
	c.wrap = on
	c.scrollX = 0
	c.wrapW = -1 // força o recálculo na próxima renderização
	c.goalX = -1
	c.Invalidate()
	return c
}

// ensureWrap garante os caches de quebra (colunas de quebra por linha e o
// prefixo linha→primeira linha visual) para a largura corrente. Linhas não
// editadas reaproveitam o cálculo anterior — o passe é um laço de inteiros.
func (c *CodeEditor) ensureWrap() {
	if !c.wrap || c.theme == nil {
		return
	}
	c.ensureMetrics()
	avail := c.textArea().Dx() - c.advance // folga para o caret no fim
	if avail < c.advance {
		avail = c.advance
	}
	if avail != c.wrapW {
		c.wrapW = avail
		for i := range c.buf.lines {
			c.buf.lines[i].wrapOK = false
		}
	} else if c.wrapGen == c.buf.version && len(c.rowStart) == c.buf.count()+1 {
		return
	}
	c.wrapGen = c.buf.version
	c.rowStart = resizeInts(c.rowStart, c.buf.count()+1)
	row := 0
	for i := range c.buf.lines {
		c.rowStart[i] = row
		if !c.buf.lines[i].wrapOK {
			c.wrapLine(i)
		}
		row += 1 + len(c.buf.lines[i].wrapStarts)
	}
	c.rowStart[c.buf.count()] = row
}

// wrapLine recalcula as colunas de quebra da linha i para a largura
// corrente — célula a célula, com tab stops por linha visual.
func (c *CodeEditor) wrapLine(i int) {
	l := &c.buf.lines[i]
	l.wrapStarts = l.wrapStarts[:0]
	x := 0
	for col, r := range l.runes {
		var next int
		if r == '\t' {
			next = (x/c.tabW() + 1) * c.tabW()
		} else {
			next = x + c.advance
		}
		if next > c.wrapW && x > 0 {
			l.wrapStarts = append(l.wrapStarts, col)
			if r == '\t' {
				next = c.tabW()
			} else {
				next = c.advance
			}
		}
		x = next
	}
	l.wrapOK = true
}

// totalRows devolve o total de linhas VISUAIS do conteúdo.
func (c *CodeEditor) totalRows() int {
	if !c.wrap {
		return c.buf.count()
	}
	c.ensureWrap()
	return c.rowStart[c.buf.count()]
}

// rowsOfLine devolve quantas linhas visuais a linha lógica i ocupa.
func (c *CodeEditor) rowsOfLine(i int) int {
	if !c.wrap {
		return 1
	}
	c.ensureWrap()
	return 1 + len(c.buf.lines[i].wrapStarts)
}

// rowStartCol devolve a coluna inicial da k-ésima linha visual da linha i.
func (c *CodeEditor) rowStartCol(i, k int) int {
	if k == 0 {
		return 0
	}
	return c.buf.lines[i].wrapStarts[k-1]
}

// rowEndCol devolve a coluna final (exclusiva) da k-ésima linha visual.
func (c *CodeEditor) rowEndCol(i, k int) int {
	ws := c.buf.lines[i].wrapStarts
	if k < len(ws) {
		return ws[k]
	}
	return len(c.buf.lines[i].runes)
}

// rowIndexIn devolve em qual linha visual da linha i a coluna col vive
// (uma coluna exatamente na quebra pertence à linha visual de baixo).
func (c *CodeEditor) rowIndexIn(i, col int) int {
	if !c.wrap {
		return 0
	}
	c.ensureWrap()
	ws := c.buf.lines[i].wrapStarts
	k := 0
	for k < len(ws) && col >= ws[k] {
		k++
	}
	return k
}

// rowOfPos devolve a linha visual absoluta da posição.
func (c *CodeEditor) rowOfPos(p textPos) int {
	if !c.wrap {
		return p.Line
	}
	c.ensureWrap()
	return c.rowStart[p.Line] + c.rowIndexIn(p.Line, p.Col)
}

// lineOfRow devolve a linha lógica e o índice de linha visual dentro dela.
func (c *CodeEditor) lineOfRow(row int) (line, k int) {
	if !c.wrap {
		return row, 0
	}
	c.ensureWrap()
	lo, hi := 0, c.buf.count()-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if c.rowStart[mid] <= row {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo, row - c.rowStart[lo]
}

// rowXAt devolve o deslocamento em px da coluna col DENTRO da k-ésima
// linha visual da linha i (tab stops reiniciados na linha visual). Sem
// wrap (k=0), é o xAt de sempre.
func (c *CodeEditor) rowXAt(i, k, col int) int {
	runes := c.buf.lines[i].runes
	start := c.rowStartCol(i, k)
	if col > len(runes) {
		col = len(runes)
	}
	x := 0
	for j := start; j < col; j++ {
		if runes[j] == '\t' {
			x = (x/c.tabW() + 1) * c.tabW()
		} else {
			x += c.advance
		}
	}
	return x
}

// colAtRow devolve a coluna mais próxima do deslocamento x na k-ésima
// linha visual da linha i.
func (c *CodeEditor) colAtRow(i, k, x int) int {
	runes := c.buf.lines[i].runes
	start := c.rowStartCol(i, k)
	end := len(runes)
	if c.wrap {
		end = c.rowEndCol(i, k)
	}
	if x <= 0 {
		return start
	}
	cur := 0
	for j := start; j < end; j++ {
		var next int
		if runes[j] == '\t' {
			next = (cur/c.tabW() + 1) * c.tabW()
		} else {
			next = cur + c.advance
		}
		if x < (cur+next)/2 {
			return j
		}
		cur = next
	}
	return end
}

// drawScrollbars desenha os indicadores de rolagem em pílula (como o
// Scroll): o vertical quando o conteúdo excede a altura; o horizontal só
// sem WrapLines, quando a linha mais larga excede a largura.
func (c *CodeEditor) drawScrollbars(view *image.RGBA, ta image.Rectangle) {
	th := c.theme
	w := th.Px(th.ScrollbarWidth)
	margin := 2 * th.BorderPx()
	minThumb := th.Px(4 * th.ScrollbarWidth)

	if contentH := c.totalRows() * c.rowH(); contentH > ta.Dy() {
		thumbH := ta.Dy() * ta.Dy() / contentH
		if thumbH < minThumb {
			thumbH = minThumb
		}
		maxOff := contentH - ta.Dy()
		thumbY := ta.Min.Y + (ta.Dy()-thumbH)*c.scrollY/maxOff
		render.FillRoundRect(view, image.Rect(ta.Max.X-w-margin, thumbY, ta.Max.X-margin, thumbY+thumbH), th.RadiusPx(), th.Placeholder)
	}
	if !c.wrap {
		if contentW := c.maxWidth() + 2*c.advance; contentW > ta.Dx() {
			thumbW := ta.Dx() * ta.Dx() / contentW
			if thumbW < minThumb {
				thumbW = minThumb
			}
			maxOff := contentW - ta.Dx()
			thumbX := ta.Min.X + (ta.Dx()-thumbW)*c.scrollX/maxOff
			render.FillRoundRect(view, image.Rect(thumbX, ta.Max.Y-w-margin, thumbX+thumbW, ta.Max.Y-margin), th.RadiusPx(), th.Placeholder)
		}
	}
}

// damageHScrollbar danifica a faixa do indicador horizontal — digitar pode
// mudar a linha mais larga e, com ela, o tamanho/posição do indicador.
func (c *CodeEditor) damageHScrollbar() {
	if c.wrap || c.theme == nil || c.Bounds().Empty() {
		return
	}
	ta := c.textArea()
	alto := c.theme.Px(c.theme.ScrollbarWidth) + 2*c.theme.BorderPx()
	c.damage(image.Rect(ta.Min.X, ta.Max.Y-alto, ta.Max.X, ta.Max.Y))
	hooks.RequestFrame()
}
