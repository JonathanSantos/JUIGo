package widget

import (
	"image"
	"image/color"
	"strconv"
	"unicode/utf8"

	"golang.org/x/image/font"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// CodeEditor é o editor de texto para CÓDIGO: fonte monoespaçada do tema,
// gutter com números de linha, tabs por tab stop, seleção multilinha,
// undo/redo coalescido (Ctrl/Cmd+Z; com Shift, refaz) e rolagem 2D própria
// — só as linhas visíveis são desenhadas, então arquivos grandes custam o
// que se vê, não o que se tem. Digitar dentro de uma linha repinta SÓ
// aquela linha (dirty regions).
//
// O Tab é LITERAL (ver ConsumesTab): insere '\t', indenta o bloco quando a
// seleção atravessa linhas, e Shift+Tab remove indentação — para tirar o
// foco do editor, use o mouse. Enter herda a indentação da linha (mais um
// tab após um abridor de bloco); a linha do cursor ganha uma faixa sutil e
// o par de parênteses adjacente ao cursor é realçado (pulando strings e
// comentários quando há highlight — ver Highlight e juigo/syntax). Não há
// BindValue deliberadamente: materializar o texto inteiro a cada tecla
// custaria O(n); use OnChange para saber QUE mudou e Text() quando
// precisar do conteúdo (salvar, por exemplo).
type CodeEditor struct {
	BaseWidget

	buf       *codeBuffer
	cursor    textPos
	anchor    textPos
	selecting bool
	focused   bool
	onChange  func()
	// hl é o highlighter de sintaxe (ver Highlight); nil = texto comum.
	hl Highlighter
	// brA/brB são o par de parênteses realçado sob o cursor (brOK válido);
	// recalculado por updateBrackets a cada movimento/edição.
	brA, brB textPos
	brOK     bool
	// tabCols é a largura do tab em COLUNAS (células mono); ver TabWidth.
	tabCols int

	// goalX é a coluna em px desejada nas navegações verticais; -1 deriva
	// da posição atual do cursor.
	goalX            int
	scrollX, scrollY int
	accumX, accumY   float64

	// wrap liga a quebra visual de linhas (ver WrapLines); wrapW é a
	// largura usada no último cálculo, wrapGen a versão do buffer coberta
	// e rowStart o prefixo linha lógica → primeira linha visual.
	wrap     bool
	wrapW    int
	wrapGen  int
	rowStart []int

	// Caches de métrica da fonte mono corrente (ver ensureMetrics): mono é
	// a variação em uso (a padrão do tema, ou a de fontSize).
	mono     *theme.MonoFont
	fontSize float64
	monoSize float64
	syncFace font.Face
	advance  int
	// maxW é a maior largura de linha em px (com maxWLine como dona);
	// maxDirty pede um rescan — os widths por linha ficam no codeBuffer.
	maxW, maxWLine int
	maxDirty       bool
	// numCache guarda as strings dos números de linha (1-based) para o
	// gutter não alocar por frame.
	numCache []string

	clipText image.RGBA
	clipGut  image.RGBA

	caretOn     bool
	blinkCancel func()

	// Composição de IME, desenhada inline na linha do cursor (medida pela
	// face mono); preAntes/preDepois são os trechos da linha ao redor.
	pre       preeditState
	preAntes  string
	preDepois string
}

// NewCodeEditor cria um editor vazio. O tema é herdado no mount.
func NewCodeEditor() *CodeEditor {
	return &CodeEditor{buf: newCodeBuffer(), tabCols: 4, goalX: -1, caretOn: true}
}

// OnChange define o callback chamado após qualquer edição — sem o texto:
// materializá-lo a cada tecla custaria O(n); chame Text() quando precisar.
// Encadeável.
func (c *CodeEditor) OnChange(fn func()) *CodeEditor {
	c.onChange = fn
	return c
}

// TabWidth define a largura do tab em colunas (mínimo 1; padrão 4).
// Encadeável.
func (c *CodeEditor) TabWidth(cols int) *CodeEditor {
	if cols < 1 {
		cols = 1
	}
	c.tabCols = cols
	c.Invalidate()
	return c
}

// Text devolve o conteúdo completo — O(n); fora do caminho quente.
func (c *CodeEditor) Text() string {
	return c.buf.String()
}

// SetText substitui todo o conteúdo (zerando undo/redo), leva o cursor ao
// início e rola ao topo. Não dispara OnChange.
func (c *CodeEditor) SetText(s string) {
	c.buf.setText(s)
	c.cursor, c.anchor = textPos{}, textPos{}
	c.scrollX, c.scrollY = 0, 0
	c.goalX = -1
	c.maxDirty = true
	c.relexFrom(0)
	c.restartBlink()
	c.updateBrackets()
	c.Invalidate()
}

// Cursor devolve a posição do cursor (linha e coluna em runes, base 0).
func (c *CodeEditor) Cursor() (linha, coluna int) {
	return c.cursor.Line, c.cursor.Col
}

// SetCursor move o cursor (limitado ao conteúdo), limpa a seleção e rola
// até ele ficar visível — a base de um "ir à linha".
func (c *CodeEditor) SetCursor(linha, coluna int) {
	c.moveTo(c.buf.clamp(textPos{linha, coluna}), false)
}

// LineCount devolve o número de linhas do conteúdo.
func (c *CodeEditor) LineCount() int {
	return c.buf.count()
}

// Undo desfaz o último passo de edição (grupo coalescido), se houver.
func (c *CodeEditor) Undo() {
	if caret, minLine, ok := c.buf.undoStep(); ok {
		c.afterHistory(caret, minLine)
	}
}

// Redo refaz o último passo desfeito, se houver.
func (c *CodeEditor) Redo() {
	if caret, minLine, ok := c.buf.redoStep(); ok {
		c.afterHistory(caret, minLine)
	}
}

// afterHistory conclui um undo/redo: cursor restaurado, highlight re-lexado
// da menor linha afetada, tudo repintado.
func (c *CodeEditor) afterHistory(caret textPos, minLine int) {
	c.cursor, c.anchor = caret, caret
	c.goalX = -1
	c.maxDirty = true
	c.relexFrom(minLine)
	c.restartBlink()
	c.updateBrackets()
	c.ensureVisible()
	c.Invalidate()
	c.emitChange()
}

// Focusable devolve true; ConsumesTab reivindica o Tab literal e
// CursorShape pede o I-beam.
func (c *CodeEditor) Focusable() bool          { return true }
func (c *CodeEditor) ConsumesTab() bool        { return true }
func (c *CodeEditor) CursorShape() CursorShape { return CursorText }

// PreferredSize devolve um mínimo razoável — o editor é feito para crescer
// com Grow e rolar por conta própria.
func (c *CodeEditor) PreferredSize() image.Point {
	if c.theme == nil {
		return image.Point{}
	}
	c.ensureMetrics()
	return image.Point{
		X: c.theme.InputMinWidthPx(),
		Y: c.theme.TextAreaMinLines * c.mono.LineHeight(),
	}
}

// FontSize define o tamanho LÓGICO da fonte DESTE editor, em pontos (zero
// volta ao tamanho do tema). A variação é fabricada pelo tema na escala
// corrente (Theme.MonoSized). Encadeável.
func (c *CodeEditor) FontSize(pts float64) *CodeEditor {
	if pts < 0 {
		pts = 0
	}
	if pts == c.fontSize {
		return c
	}
	c.fontSize = pts
	c.Invalidate() // ensureMetrics do próximo frame refaz as métricas
	return c
}

// ensureMetrics revalida os caches dependentes da fonte mono corrente
// (escala, tema ou FontSize trocados): variação em uso, célula, larguras
// de linha e quebra visual.
func (c *CodeEditor) ensureMetrics() {
	if c.theme == nil {
		return
	}
	if c.mono != nil && c.syncFace == c.theme.MonoFace && c.monoSize == c.fontSize {
		return
	}
	c.syncFace = c.theme.MonoFace
	c.monoSize = c.fontSize
	c.mono = c.theme.Mono()
	if c.fontSize > 0 {
		if m, err := c.theme.MonoSized(c.fontSize); err == nil {
			c.mono = m
		}
	}
	c.advance = c.mono.Advance()
	for i := range c.buf.lines {
		c.buf.lines[i].widthOK = false
		c.buf.lines[i].wrapOK = false
	}
	c.maxDirty = true
	c.wrapW = -1
	c.pre.measureWith(c.cellMeasure)
}

// cellMeasure mede em CÉLULAS (runes × avanço) — a mesma grade do desenho
// e do caret; a composição de IME usa isto, não a medida natural da fonte.
func (c *CodeEditor) cellMeasure(s string) int {
	return utf8.RuneCountInString(s) * c.advance
}

// tabW devolve a largura do tab stop em px.
func (c *CodeEditor) tabW() int {
	return c.tabCols * c.advance
}

// xAt devolve o deslocamento em px (espaço do conteúdo) da coluna col da
// linha line — aritmética pura de células e tab stops, sem medir fonte.
func (c *CodeEditor) xAt(line, col int) int {
	x := 0
	runes := c.buf.lines[line].runes
	if col > len(runes) {
		col = len(runes)
	}
	for i := 0; i < col; i++ {
		if runes[i] == '\t' {
			x = (x/c.tabW() + 1) * c.tabW()
		} else {
			x += c.advance
		}
	}
	return x
}

// colAt devolve a coluna mais próxima do deslocamento x px na linha.
func (c *CodeEditor) colAt(line, x int) int {
	if x <= 0 {
		return 0
	}
	runes := c.buf.lines[line].runes
	cur := 0
	for i, r := range runes {
		var next int
		if r == '\t' {
			next = (cur/c.tabW() + 1) * c.tabW()
		} else {
			next = cur + c.advance
		}
		if x < (cur+next)/2 {
			return i
		}
		cur = next
	}
	return len(runes)
}

// lineWidth devolve (e cacheia) a largura da linha i em px.
func (c *CodeEditor) lineWidth(i int) int {
	l := &c.buf.lines[i]
	if !l.widthOK {
		l.width = c.xAt(i, len(l.runes))
		l.widthOK = true
	}
	return l.width
}

// noteWidth atualiza a marca d'água de largura após editar a linha i.
func (c *CodeEditor) noteWidth(i int) {
	c.buf.lines[i].widthOK = false
	w := c.lineWidth(i)
	switch {
	case w >= c.maxW:
		c.maxW, c.maxWLine = w, i
	case i == c.maxWLine:
		c.maxDirty = true // a dona encolheu: rescan preguiçoso
	}
}

// maxWidth devolve a maior largura de linha, rescaneando se preciso.
func (c *CodeEditor) maxWidth() int {
	if c.maxDirty {
		c.maxW, c.maxWLine = 0, 0
		for i := range c.buf.lines {
			if w := c.lineWidth(i); w > c.maxW {
				c.maxW, c.maxWLine = w, i
			}
		}
		c.maxDirty = false
	}
	return c.maxW
}

// digits devolve o número de dígitos do maior número de linha.
func digits(n int) int {
	d := 1
	for n >= 10 {
		n /= 10
		d++
	}
	return d
}

// gutterW devolve a largura do gutter em px.
func (c *CodeEditor) gutterW() int {
	return digits(c.buf.count())*c.advance + 2*c.theme.Px(6)
}

// inner devolve a área interna (dentro da borda).
func (c *CodeEditor) inner() image.Rectangle {
	return c.Bounds().Inset(c.theme.BorderPx())
}

// textArea devolve a área do texto (à direita do gutter, com respiro).
func (c *CodeEditor) textArea() image.Rectangle {
	in := c.inner()
	return image.Rect(in.Min.X+c.gutterW()+c.theme.Px(6), in.Min.Y, in.Max.X, in.Max.Y)
}

// rowH devolve a altura de linha da fonte mono em uso.
func (c *CodeEditor) rowH() int {
	c.ensureMetrics()
	return c.mono.LineHeight()
}

// posAt converte um ponto absoluto em posição de texto (pela linha VISUAL
// sob o ponto).
func (c *CodeEditor) posAt(p image.Point) textPos {
	ta := c.textArea()
	row := (p.Y - ta.Min.Y + c.scrollY) / c.rowH()
	if row < 0 {
		row = 0
	}
	if rows := c.totalRows(); row >= rows {
		row = rows - 1
	}
	line, k := c.lineOfRow(row)
	return textPos{line, c.colAtRow(line, k, p.X-ta.Min.X+c.scrollX)}
}

// selectionRange devolve os limites ordenados da seleção.
func (c *CodeEditor) selectionRange() (start, end textPos) {
	if c.anchor.before(c.cursor) {
		return c.anchor, c.cursor
	}
	return c.cursor, c.anchor
}

// hasSelection informa se há trecho selecionado.
func (c *CodeEditor) hasSelection() bool {
	return c.anchor != c.cursor
}

// moveTo move o cursor (com ou sem extensão de seleção), quebra a
// coalescência do undo e garante o cursor visível.
func (c *CodeEditor) moveTo(pos textPos, extend bool) bool {
	pos = c.buf.clamp(pos)
	hadSel := c.hasSelection()
	changed := pos != c.cursor || (!extend && c.anchor != pos)
	old := c.cursor.Line
	c.cursor = pos
	if !extend {
		c.anchor = pos
	}
	if changed {
		c.buf.breakGroup()
		c.restartBlink()
		c.updateBrackets()
		// Seleção surgindo/sumindo ou linha trocada: repinta tudo (a
		// seleção pode atravessar muitas linhas); movimento simples na
		// mesma linha repinta só ela.
		if c.ensureVisible() || extend || hadSel || old != pos.Line {
			c.Invalidate()
		} else {
			c.damageLine(old)
			c.damageLine(pos.Line)
		}
	}
	return changed
}

// ensureVisible ajusta a rolagem para o cursor ficar visível; devolve true
// se a rolagem mudou (o chamador repinta tudo).
func (c *CodeEditor) ensureVisible() bool {
	if c.theme == nil || c.Bounds().Empty() {
		return false
	}
	ta := c.textArea()
	changed := false
	top := c.rowOfPos(c.cursor) * c.rowH()
	if top < c.scrollY {
		c.scrollY = top
		changed = true
	}
	if bot := top + c.rowH(); bot > c.scrollY+ta.Dy() {
		c.scrollY = bot - ta.Dy()
		changed = true
	}
	if c.wrap {
		return changed // sem rolagem horizontal com quebra visual
	}
	cx := c.xAt(c.cursor.Line, c.cursor.Col) + c.pre.caretX
	margin := 2 * c.advance
	if cx < c.scrollX+margin {
		c.scrollX = cx - margin
		if c.scrollX < 0 {
			c.scrollX = 0
		}
		changed = true
	}
	if cx > c.scrollX+ta.Dx()-margin {
		c.scrollX = cx - ta.Dx() + margin
		changed = true
	}
	return changed
}

// damageLine danifica a faixa da linha LÓGICA i (texto e gutter) — com
// WrapLines, todas as linhas visuais dela.
func (c *CodeEditor) damageLine(i int) {
	if c.theme == nil {
		return
	}
	in := c.inner()
	startRow, rows := i, 1
	if c.wrap {
		c.ensureWrap()
		if i >= c.buf.count() {
			i = c.buf.count() - 1
		}
		startRow = c.rowStart[i]
		rows = c.rowsOfLine(i)
	}
	y := in.Min.Y + startRow*c.rowH() - c.scrollY
	c.damage(image.Rect(in.Min.X, y, in.Max.X, y+rows*c.rowH()))
	hooks.RequestFrame()
}

// damageCaret danifica só o retângulo do cursor (piscada).
func (c *CodeEditor) damageCaret() {
	c.damage(c.CaretRect())
	hooks.RequestFrame()
}

// CaretRect devolve o retângulo do cursor em coordenadas absolutas —
// durante uma composição, dentro dela (contrato widget.TextCaret).
func (c *CodeEditor) CaretRect() image.Rectangle {
	if c.theme == nil {
		return image.Rectangle{}
	}
	ta := c.textArea()
	k := c.rowIndexIn(c.cursor.Line, c.cursor.Col)
	x := ta.Min.X + c.rowXAt(c.cursor.Line, k, c.cursor.Col) + c.pre.caretX - c.scrollX
	y := ta.Min.Y + c.rowOfPos(c.cursor)*c.rowH() - c.scrollY
	return image.Rect(x, y, x+c.theme.BorderPx(), y+c.rowH())
}

// emitChange propaga uma edição ao OnChange.
func (c *CodeEditor) emitChange() {
	if c.onChange != nil {
		c.onChange()
	}
}

// insertText insere s no cursor (substituindo a seleção), com o tipo de
// coalescência dado. Digitação dentro da linha repinta só a linha.
func (c *CodeEditor) insertText(s string, kind editKind) {
	structural := c.hasSelection() || containsNewline(s)
	line := c.cursor.Line
	if c.hasSelection() {
		// Substituir a seleção é UM passo de undo (replace).
		start, end := c.selectionRange()
		line = start.Line
		c.cursor = c.buf.replace(start, end, s)
	} else {
		c.cursor = c.buf.insert(c.cursor, s, kind)
	}
	c.anchor = c.cursor
	c.goalX = -1
	c.restartBlink()
	if structural {
		c.afterStructuralAt(line)
		return
	}
	cascata := c.relexFrom(line) > line
	c.noteWidth(line)
	c.updateBrackets()
	// Com WrapLines, editar pode mudar quantas linhas visuais a linha
	// ocupa — tudo abaixo desloca: repinta inteiro.
	if c.wrap || cascata || c.ensureVisible() {
		c.Invalidate()
	} else {
		c.damageLine(line)
		c.damageHScrollbar()
	}
	c.emitChange()
}

// containsNewline evita importar strings só para isto.
func containsNewline(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return true
		}
	}
	return false
}

// deleteBack apaga a seleção ou a rune antes do cursor (juntando linhas na
// borda).
func (c *CodeEditor) deleteBack() bool {
	if c.hasSelection() {
		c.removeSelection()
		return true
	}
	pos := c.cursor
	if pos.Col > 0 {
		start := textPos{pos.Line, pos.Col - 1}
		c.buf.deleteRange(start, pos, editDelBack)
		c.cursor, c.anchor = start, start
		c.afterLineEdit(start.Line)
		return true
	}
	if pos.Line == 0 {
		return false
	}
	start := textPos{pos.Line - 1, c.buf.lineLen(pos.Line - 1)}
	c.buf.deleteRange(start, pos, editOther)
	c.cursor, c.anchor = start, start
	c.afterStructuralAt(start.Line)
	return true
}

// deleteFwd apaga a seleção ou a rune sob o cursor.
func (c *CodeEditor) deleteFwd() bool {
	if c.hasSelection() {
		c.removeSelection()
		return true
	}
	pos := c.cursor
	if pos.Col < c.buf.lineLen(pos.Line) {
		c.buf.deleteRange(pos, textPos{pos.Line, pos.Col + 1}, editDelFwd)
		c.afterLineEdit(pos.Line)
		return true
	}
	if pos.Line == c.buf.count()-1 {
		return false
	}
	c.buf.deleteRange(pos, textPos{pos.Line + 1, 0}, editOther)
	c.afterStructuralAt(pos.Line)
	return true
}

// removeSelection apaga o trecho selecionado (grupo próprio no undo).
func (c *CodeEditor) removeSelection() {
	start, end := c.selectionRange()
	c.buf.deleteRange(start, end, editOther)
	c.cursor, c.anchor = start, start
	c.afterStructuralAt(start.Line)
}

// afterLineEdit conclui uma edição dentro de uma única linha: re-lexa (uma
// cascata de highlight — abrir /* — repinta tudo) e danifica só a linha.
func (c *CodeEditor) afterLineEdit(line int) {
	c.goalX = -1
	cascata := c.relexFrom(line) > line
	c.noteWidth(line)
	c.restartBlink()
	c.updateBrackets()
	if c.wrap || cascata || c.ensureVisible() {
		c.Invalidate()
	} else {
		c.damageLine(line)
		c.damageHScrollbar()
	}
	c.emitChange()
}

// afterStructuralAt conclui uma edição que muda a contagem de linhas,
// re-lexando a partir da primeira afetada.
func (c *CodeEditor) afterStructuralAt(line int) {
	c.goalX = -1
	c.maxDirty = true
	c.relexFrom(line)
	c.restartBlink()
	c.updateBrackets()
	c.ensureVisible()
	c.Invalidate()
	c.emitChange()
}

// HandleEvent trata caracteres, teclas, composição de IME, foco, rolagem e
// mouse.
func (c *CodeEditor) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.CharEvent:
		c.insertText(string(e.Rune), editType)
		return true
	case event.PreeditEvent:
		c.setPreedit(e)
		return true
	case event.KeyEvent:
		return c.handleKey(e)
	case event.FocusEvent:
		c.focused = e.Gained
		if e.Gained {
			c.restartBlink()
		} else {
			c.stopBlink()
			c.buf.breakGroup()
			if c.pre.clear() {
				c.syncPreedit()
				c.Invalidate()
			}
		}
		c.updateBrackets()
		c.Invalidate()
		return true
	case event.ScrollEvent:
		return c.handleScroll(e)
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			c.selecting = true
			c.goalX = -1
			c.moveTo(c.posAt(e.Pos), false)
			return true
		case event.MouseMove:
			if !c.selecting {
				return false
			}
			return c.moveTo(c.posAt(e.Pos), true)
		case event.MouseUp:
			c.selecting = false
			return false
		}
	}
	return false
}

// handleKey aplica as teclas de edição, navegação e atalhos.
func (c *CodeEditor) handleKey(e event.KeyEvent) bool {
	switch e.Key {
	case event.KeyEnter:
		// A linha nova herda a indentação (e ganha um tab após { ( [).
		c.insertText("\n"+c.autoIndent(), editOther)
		return true
	case event.KeyTab:
		if e.Mods.Shift() {
			c.indentBlock(true)
			return true
		}
		if c.spansLines() {
			c.indentBlock(false)
			return true
		}
		c.insertText("\t", editType)
		return true
	case event.KeyBackspace:
		return c.deleteBack()
	case event.KeyDelete:
		return c.deleteFwd()
	case event.KeyLeft:
		if !e.Mods.Shift() && c.hasSelection() {
			start, _ := c.selectionRange()
			return c.moveTo(start, false)
		}
		c.goalX = -1
		return c.moveTo(c.prevPos(c.cursor), e.Mods.Shift())
	case event.KeyRight:
		if !e.Mods.Shift() && c.hasSelection() {
			_, end := c.selectionRange()
			return c.moveTo(end, false)
		}
		c.goalX = -1
		return c.moveTo(c.nextPos(c.cursor), e.Mods.Shift())
	case event.KeyUp:
		return c.moveVertical(-1, e.Mods.Shift())
	case event.KeyDown:
		return c.moveVertical(1, e.Mods.Shift())
	case event.KeyHome:
		c.goalX = -1
		return c.moveTo(textPos{c.cursor.Line, 0}, e.Mods.Shift())
	case event.KeyEnd:
		c.goalX = -1
		return c.moveTo(textPos{c.cursor.Line, c.buf.lineLen(c.cursor.Line)}, e.Mods.Shift())
	case event.KeyA:
		if !e.Mods.Command() {
			return false
		}
		last := c.buf.count() - 1
		c.anchor = textPos{}
		c.cursor = textPos{last, c.buf.lineLen(last)}
		c.Invalidate()
		return true
	case event.KeyC:
		if !e.Mods.Command() || !c.hasSelection() {
			return false
		}
		start, end := c.selectionRange()
		hooks.WriteClipboard(c.buf.textRange(start, end))
		return true
	case event.KeyX:
		if !e.Mods.Command() || !c.hasSelection() {
			return false
		}
		start, end := c.selectionRange()
		hooks.WriteClipboard(c.buf.textRange(start, end))
		c.removeSelection()
		return true
	case event.KeyV:
		if !e.Mods.Command() {
			return false
		}
		if s := hooks.ReadClipboard(); s != "" {
			c.insertText(s, editOther)
			return true
		}
		return false
	case event.KeyZ:
		if !e.Mods.Command() {
			return false
		}
		if e.Mods.Shift() {
			c.Redo()
		} else {
			c.Undo()
		}
		return true
	}
	return false
}

// prevPos devolve a posição imediatamente anterior (atravessando linhas).
func (c *CodeEditor) prevPos(p textPos) textPos {
	if p.Col > 0 {
		return textPos{p.Line, p.Col - 1}
	}
	if p.Line == 0 {
		return p
	}
	return textPos{p.Line - 1, c.buf.lineLen(p.Line - 1)}
}

// nextPos devolve a posição imediatamente seguinte (atravessando linhas).
func (c *CodeEditor) nextPos(p textPos) textPos {
	if p.Col < c.buf.lineLen(p.Line) {
		return textPos{p.Line, p.Col + 1}
	}
	if p.Line == c.buf.count()-1 {
		return p
	}
	return textPos{p.Line + 1, 0}
}

// moveVertical sobe/desce uma linha preservando a coluna DESEJADA em px
// (goalX), como a TextArea.
func (c *CodeEditor) moveVertical(delta int, extend bool) bool {
	row := c.rowOfPos(c.cursor) + delta
	if row < 0 || row >= c.totalRows() {
		return false
	}
	if c.goalX < 0 {
		k := c.rowIndexIn(c.cursor.Line, c.cursor.Col)
		c.goalX = c.rowXAt(c.cursor.Line, k, c.cursor.Col)
	}
	goal := c.goalX
	line, k := c.lineOfRow(row)
	moved := c.moveTo(textPos{line, c.colAtRow(line, k, goal)}, extend)
	c.goalX = goal // moveTo não zera: a intenção vertical sobrevive
	return moved
}

// handleScroll rola o conteúdo nos dois eixos (deltas fracionários
// acumulados); nos limites propaga, como o Scroll.
func (c *CodeEditor) handleScroll(e event.ScrollEvent) bool {
	if c.theme == nil {
		return false
	}
	step := float64(c.theme.Px(c.theme.ScrollStep))
	moved := false

	c.accumY += e.DY * step
	dy := int(c.accumY)
	c.accumY -= float64(dy)
	if dy != 0 {
		max := c.totalRows()*c.rowH() - c.textArea().Dy()
		if max < 0 {
			max = 0
		}
		novo := c.scrollY - dy
		if novo < 0 {
			novo = 0
		}
		if novo > max {
			novo = max
		}
		if novo != c.scrollY {
			c.scrollY = novo
			moved = true
		}
	}

	c.accumX += e.DX * step
	dx := int(c.accumX)
	c.accumX -= float64(dx)
	if dx != 0 && !c.wrap {
		max := c.maxWidth() + 2*c.advance - c.textArea().Dx()
		if max < 0 {
			max = 0
		}
		novo := c.scrollX - dx
		if novo < 0 {
			novo = 0
		}
		if novo > max {
			novo = max
		}
		if novo != c.scrollX {
			c.scrollX = novo
			moved = true
		}
	}

	if moved {
		c.Invalidate()
	}
	return moved
}

// restartBlink/blinkTick/stopBlink: piscada do cursor com dano mínimo (só o
// retângulo do caret).
func (c *CodeEditor) restartBlink() {
	c.stopBlink()
	c.caretOn = true
	if !c.focused || c.theme == nil || c.theme.CaretBlink <= 0 {
		return
	}
	c.blinkCancel = hooks.ScheduleAfter(c.theme.CaretBlink, c.blinkTick)
}

func (c *CodeEditor) blinkTick() {
	if !c.focused {
		return
	}
	c.caretOn = !c.caretOn
	c.damageCaret()
	c.blinkCancel = hooks.ScheduleAfter(c.theme.CaretBlink, c.blinkTick)
}

func (c *CodeEditor) stopBlink() {
	if c.blinkCancel != nil {
		c.blinkCancel()
		c.blinkCancel = nil
	}
	c.caretOn = true
}

// setPreedit registra a composição de IME (medida pela face mono); compor
// sobre seleção a substitui.
func (c *CodeEditor) setPreedit(e event.PreeditEvent) {
	if e.Text != "" && c.hasSelection() {
		c.removeSelection()
	}
	c.pre.set(e)
	c.syncPreedit()
	c.restartBlink()
	c.ensureVisible()
	c.Invalidate()
}

// syncPreedit atualiza medidas (face mono) e os trechos da linha do cursor
// ao redor da composição.
func (c *CodeEditor) syncPreedit() {
	c.ensureMetrics()
	if c.mono == nil {
		c.pre.measureWith(nil)
	} else {
		c.pre.measureWith(c.cellMeasure)
	}
	if !c.pre.active() {
		c.preAntes, c.preDepois = "", ""
		return
	}
	line := c.buf.lineText(c.cursor.Line)
	off := byteOffset(line, c.cursor.Col)
	c.preAntes, c.preDepois = line[:off], line[off:]
}

// byteOffset devolve o deslocamento em bytes da coluna col (em runes).
func byteOffset(s string, col int) int {
	n := 0
	for i := range s {
		if n == col {
			return i
		}
		n++
	}
	return len(s)
}

// ensureNums garante as strings dos números de linha até count.
func (c *CodeEditor) ensureNums(count int) {
	for len(c.numCache) < count {
		c.numCache = append(c.numCache, strconv.Itoa(len(c.numCache)+1))
	}
}

// drawTabbed desenha s a partir do deslocamento xContent (espaço do
// conteúdo), quebrando nos tabs para respeitar os tab stops. screenX0 é o
// X absoluto do deslocamento zero. Devolve o deslocamento final. O avanço é
// a MESMA aritmética de células do xAt (runes × célula) — desenho, caret e
// seleção nunca divergem. Não aloca: desenha fatias de s.
func (c *CodeEditor) drawTabbed(view *image.RGBA, s string, xContent, screenX0, baseline int, cor color.RGBA) int {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\t' {
			continue
		}
		if i > start {
			c.mono.Draw(view, s[start:i], image.Pt(screenX0+xContent, baseline), cor)
			xContent += utf8.RuneCountInString(s[start:i]) * c.advance
		}
		xContent = (xContent/c.tabW() + 1) * c.tabW()
		start = i + 1
	}
	if start < len(s) {
		c.mono.Draw(view, s[start:], image.Pt(screenX0+xContent, baseline), cor)
		xContent += utf8.RuneCountInString(s[start:]) * c.advance
	}
	return xContent
}

// Draw desenha fundo, borda, gutter numerado, as LINHAS VISUAIS visíveis
// (faixa da linha atual, par de parênteses, seleção, texto com tabs,
// composição de IME), o cursor e os indicadores de rolagem.
func (c *CodeEditor) Draw(dst *image.RGBA) {
	if c.theme == nil {
		return
	}
	c.ensureMetrics()
	th := c.theme
	bounds := c.Bounds()
	radius := th.RadiusPx()

	render.FillRoundRect(dst, bounds, radius, th.InputBackground)
	border := th.InputBorder
	if c.focused {
		border = th.InputBorderFocused
	}
	render.StrokeRoundRect(dst, bounds, radius, th.BorderPx(), border)

	in := c.inner()
	ta := c.textArea()
	rowH := c.rowH()
	rows := c.totalRows()

	// Recorte de rolagem (nunca além do conteúdo).
	if max := rows*rowH - ta.Dy(); c.scrollY > max {
		c.scrollY = max
	}
	if c.scrollY < 0 {
		c.scrollY = 0
	}
	if c.wrap {
		c.scrollX = 0
	} else {
		if max := c.maxWidth() + 2*c.advance - ta.Dx(); c.scrollX > max {
			c.scrollX = max
		}
		if c.scrollX < 0 {
			c.scrollX = 0
		}
	}

	// Gutter: fundo, separador e números das linhas visíveis.
	gut := image.Rect(in.Min.X, in.Min.Y, in.Min.X+c.gutterW(), in.Max.Y)
	gutView := render.Clip(dst, gut, &c.clipGut)
	render.FillRect(gutView, gut, th.HoverBackground)
	render.FillRect(dst, image.Rect(gut.Max.X, in.Min.Y, gut.Max.X+1, in.Max.Y), th.InputBorder)

	first := c.scrollY / rowH
	last := (c.scrollY + ta.Dy() - 1) / rowH
	if last >= rows {
		last = rows - 1
	}

	textView := render.Clip(dst, ta, &c.clipText)
	selStart, selEnd := c.selectionRange()
	gutPad := th.Px(6)
	curRowIdx := -1
	if c.pre.active() {
		curRowIdx = c.rowIndexIn(c.cursor.Line, c.cursor.Col)
	}

	for row := first; row <= last; row++ {
		i, k := c.lineOfRow(row)
		y := in.Min.Y + row*rowH - c.scrollY
		baseline := y + c.mono.Ascent()

		// Número só na primeira linha visual; a do cursor em destaque.
		if k == 0 {
			c.ensureNums(i + 1)
			num := c.numCache[i]
			numColor := th.Placeholder
			if i == c.cursor.Line {
				numColor = th.Text
			}
			c.mono.Draw(gutView, num, image.Pt(gut.Max.X-gutPad-len(num)*c.advance, baseline), numColor)
		}

		// Faixa da linha atual (todas as linhas visuais dela) e o par de
		// parênteses.
		if c.focused && !c.hasSelection() && i == c.cursor.Line {
			render.FillRect(textView, image.Rect(ta.Min.X, y, ta.Max.X, y+rowH), th.CurrentLine)
		}
		if c.brOK && c.focused {
			for _, p := range [2]textPos{c.brA, c.brB} {
				if p.Line == i && c.rowIndexIn(p.Line, p.Col) == k {
					bx := ta.Min.X + c.rowXAt(i, k, p.Col) - c.scrollX
					render.FillRect(textView, image.Rect(bx, y, bx+c.advance, y+rowH), th.Selection)
				}
			}
		}

		a := c.rowStartCol(i, k)
		b := c.rowEndCol(i, k)

		// Seleção nesta linha visual (o '\n' selecionado vira um talão de
		// célula na última linha visual da linha lógica).
		if c.focused && c.hasSelection() && i >= selStart.Line && i <= selEnd.Line {
			sCol := 0
			if i == selStart.Line {
				sCol = selStart.Col
			}
			eCol := len(c.buf.lines[i].runes) + 1 // +1 = o '\n'
			if i == selEnd.Line {
				eCol = selEnd.Col
			}
			s2, e2 := max(sCol, a), min(eCol, b)
			talao := 0
			if eCol > b && b == len(c.buf.lines[i].runes) {
				e2 = b
				talao = c.advance
			}
			if e2 > s2 || talao > 0 {
				x0 := c.rowXAt(i, k, s2)
				x1 := c.rowXAt(i, k, e2) + talao
				render.FillRect(textView, image.Rect(ta.Min.X+x0-c.scrollX, y, ta.Min.X+x1-c.scrollX, y+rowH), th.Selection)
			}
		}

		// Texto (com a composição inline na linha visual do cursor).
		screenX0 := ta.Min.X - c.scrollX
		if c.pre.active() && i == c.cursor.Line && k == curRowIdx {
			// Durante a composição o trecho sai em texto comum: os spans
			// não valem com o preedit intercalado (transitório).
			text := c.buf.lineText(i)
			aB, cB, bB := byteOffset(text, a), byteOffset(text, c.cursor.Col), byteOffset(text, b)
			x := c.drawTabbedPart(textView, text[aB:cB], 0, screenX0, baseline)
			c.mono.Draw(textView, c.pre.text, image.Pt(screenX0+x, baseline), th.Text)
			c.pre.drawUnderline(textView, th, screenX0+x, baseline+th.Px(2))
			c.drawTabbedPart(textView, text[cB:bB], x+c.pre.w, screenX0, baseline)
		} else {
			c.drawRow(textView, i, k, screenX0, baseline)
		}
	}

	if c.focused && c.caretOn {
		render.FillRect(textView, c.CaretRect(), th.Cursor)
	}
	c.drawScrollbars(textView, ta)
	c.drawDisabledOverlay(dst)
}

// drawRow desenha a k-ésima linha visual da linha i: os spans do highlight
// recortados ao intervalo da linha visual (ou texto comum), atravessando os
// tab stops dela.
func (c *CodeEditor) drawRow(view *image.RGBA, i, k, screenX0, baseline int) {
	text := c.buf.lineText(i)
	aB := byteOffset(text, c.rowStartCol(i, k))
	bB := byteOffset(text, c.rowEndCol(i, k))
	l := &c.buf.lines[i]
	if c.hl == nil || !l.hlOK {
		if bB > aB {
			c.drawTabbed(view, text[aB:bB], 0, screenX0, baseline, c.theme.Text)
		}
		return
	}
	x, off, covered := 0, 0, aB
	for _, sp := range l.spans {
		spEnd := off + sp.Len
		s, e := max(off, aB), min(spEnd, bB)
		if e > s {
			x = c.drawTabbed(view, text[s:e], x, screenX0, baseline, c.styleColor(sp.Style))
			covered = e
		}
		off = spEnd
		if off >= bB {
			break
		}
	}
	if covered < bB {
		// Lexer não cobriu a linha inteira: o resto sai como texto comum.
		c.drawTabbed(view, text[covered:bB], x, screenX0, baseline, c.theme.Text)
	}
}

// drawTabbedPart é o drawTabbed para os trechos ao redor da composição.
func (c *CodeEditor) drawTabbedPart(view *image.RGBA, s string, xContent, screenX0, baseline int) int {
	if s == "" {
		return xContent
	}
	return c.drawTabbed(view, s, xContent, screenX0, baseline, c.theme.Text)
}
