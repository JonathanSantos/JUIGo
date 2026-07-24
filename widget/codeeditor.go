package widget

import (
	"image"
	"strconv"
	"unicode/utf8"

	"golang.org/x/image/font"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
)

// CodeEditor é o editor de texto para CÓDIGO: fonte monoespaçada do tema,
// gutter com números de linha, tabs por tab stop, seleção multilinha,
// undo/redo coalescido (Ctrl/Cmd+Z; com Shift, refaz) e rolagem 2D própria
// — só as linhas visíveis são desenhadas, então arquivos grandes custam o
// que se vê, não o que se tem. Digitar dentro de uma linha repinta SÓ
// aquela linha (dirty regions).
//
// O Tab é LITERAL (insere '\t'; ver ConsumesTab) — para tirar o foco do
// editor, use o mouse. Não há BindValue deliberadamente: materializar o
// texto inteiro a cada tecla custaria O(n); use OnChange para saber QUE
// mudou e Text() quando precisar do conteúdo (salvar, por exemplo).
// Highlight de sintaxe é a fase 2 do plano.
type CodeEditor struct {
	BaseWidget

	buf       *codeBuffer
	cursor    textPos
	anchor    textPos
	selecting bool
	focused   bool
	onChange  func()
	// tabCols é a largura do tab em COLUNAS (células mono); ver TabWidth.
	tabCols int

	// goalX é a coluna em px desejada nas navegações verticais; -1 deriva
	// da posição atual do cursor.
	goalX            int
	scrollX, scrollY int
	accumX, accumY   float64

	// Caches de métrica da face mono corrente (ver ensureMetrics).
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
	c.restartBlink()
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
	if caret, ok := c.buf.undoStep(); ok {
		c.afterHistory(caret)
	}
}

// Redo refaz o último passo desfeito, se houver.
func (c *CodeEditor) Redo() {
	if caret, ok := c.buf.redoStep(); ok {
		c.afterHistory(caret)
	}
}

// afterHistory conclui um undo/redo: cursor restaurado, tudo repintado.
func (c *CodeEditor) afterHistory(caret textPos) {
	c.cursor, c.anchor = caret, caret
	c.goalX = -1
	c.maxDirty = true
	c.restartBlink()
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
	return image.Point{
		X: c.theme.InputMinWidthPx(),
		Y: c.theme.TextAreaMinLines * c.theme.MonoLineHeight(),
	}
}

// ensureMetrics revalida os caches dependentes da face mono (escala ou
// tema trocado): célula, larguras de linha e números do gutter.
func (c *CodeEditor) ensureMetrics() {
	if c.theme == nil || c.syncFace == c.theme.MonoFace {
		return
	}
	c.syncFace = c.theme.MonoFace
	c.advance = c.theme.MonoAdvance()
	for i := range c.buf.lines {
		c.buf.lines[i].widthOK = false
	}
	c.maxDirty = true
	c.pre.measureWith(c.theme.MeasureMono)
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

// rowH devolve a altura de linha da face mono.
func (c *CodeEditor) rowH() int {
	return c.theme.MonoLineHeight()
}

// posAt converte um ponto absoluto em posição de texto.
func (c *CodeEditor) posAt(p image.Point) textPos {
	ta := c.textArea()
	line := (p.Y - ta.Min.Y + c.scrollY) / c.rowH()
	if line < 0 {
		line = 0
	}
	if line >= c.buf.count() {
		line = c.buf.count() - 1
	}
	return textPos{line, c.colAt(line, p.X-ta.Min.X+c.scrollX)}
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
	top := c.cursor.Line * c.rowH()
	if top < c.scrollY {
		c.scrollY = top
		changed = true
	}
	if bot := top + c.rowH(); bot > c.scrollY+ta.Dy() {
		c.scrollY = bot - ta.Dy()
		changed = true
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

// damageLine danifica a faixa da linha i (texto e gutter).
func (c *CodeEditor) damageLine(i int) {
	if c.theme == nil {
		return
	}
	in := c.inner()
	y := in.Min.Y + i*c.rowH() - c.scrollY
	c.damage(image.Rect(in.Min.X, y, in.Max.X, y+c.rowH()))
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
	x := ta.Min.X + c.xAt(c.cursor.Line, c.cursor.Col) + c.pre.caretX - c.scrollX
	y := ta.Min.Y + c.cursor.Line*c.rowH() - c.scrollY
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
		c.cursor = c.buf.replace(start, end, s)
	} else {
		c.cursor = c.buf.insert(c.cursor, s, kind)
	}
	c.anchor = c.cursor
	c.goalX = -1
	c.restartBlink()
	if structural {
		c.maxDirty = true
		c.ensureVisible()
		c.Invalidate()
	} else {
		c.noteWidth(line)
		if c.ensureVisible() {
			c.Invalidate()
		} else {
			c.damageLine(line)
		}
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
	c.afterStructural()
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
	c.afterStructural()
	return true
}

// removeSelection apaga o trecho selecionado (grupo próprio no undo).
func (c *CodeEditor) removeSelection() {
	start, end := c.selectionRange()
	c.buf.deleteRange(start, end, editOther)
	c.cursor, c.anchor = start, start
	c.afterStructural()
}

// afterLineEdit conclui uma edição dentro de uma única linha.
func (c *CodeEditor) afterLineEdit(line int) {
	c.goalX = -1
	c.noteWidth(line)
	c.restartBlink()
	if c.ensureVisible() {
		c.Invalidate()
	} else {
		c.damageLine(line)
	}
	c.emitChange()
}

// afterStructural conclui uma edição que muda a contagem de linhas.
func (c *CodeEditor) afterStructural() {
	c.goalX = -1
	c.maxDirty = true
	c.restartBlink()
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
		c.insertText("\n", editOther)
		return true
	case event.KeyTab:
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
	line := c.cursor.Line + delta
	if line < 0 || line >= c.buf.count() {
		return false
	}
	if c.goalX < 0 {
		c.goalX = c.xAt(c.cursor.Line, c.cursor.Col)
	}
	goal := c.goalX
	moved := c.moveTo(textPos{line, c.colAt(line, goal)}, extend)
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
		max := c.buf.count()*c.rowH() - c.textArea().Dy()
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
	if dx != 0 {
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
	if c.theme == nil {
		c.pre.measureWith(nil)
	} else {
		c.pre.measureWith(c.theme.MeasureMono)
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
func (c *CodeEditor) drawTabbed(view *image.RGBA, s string, xContent, screenX0, baseline int) int {
	th := c.theme
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\t' {
			continue
		}
		if i > start {
			th.DrawMono(view, s[start:i], image.Pt(screenX0+xContent, baseline), th.Text)
			xContent += utf8.RuneCountInString(s[start:i]) * c.advance
		}
		xContent = (xContent/c.tabW() + 1) * c.tabW()
		start = i + 1
	}
	if start < len(s) {
		th.DrawMono(view, s[start:], image.Pt(screenX0+xContent, baseline), th.Text)
		xContent += utf8.RuneCountInString(s[start:]) * c.advance
	}
	return xContent
}

// Draw desenha fundo, borda, gutter numerado, as LINHAS VISÍVEIS (seleção,
// texto com tabs, composição de IME) e o cursor.
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
	count := c.buf.count()

	// Recorte de rolagem (nunca além do conteúdo).
	if max := count*rowH - ta.Dy(); c.scrollY > max {
		c.scrollY = max
	}
	if c.scrollY < 0 {
		c.scrollY = 0
	}
	if max := c.maxWidth() + 2*c.advance - ta.Dx(); c.scrollX > max {
		c.scrollX = max
	}
	if c.scrollX < 0 {
		c.scrollX = 0
	}

	// Gutter: fundo, separador e números das linhas visíveis.
	gut := image.Rect(in.Min.X, in.Min.Y, in.Min.X+c.gutterW(), in.Max.Y)
	gutView := render.Clip(dst, gut, &c.clipGut)
	render.FillRect(gutView, gut, th.HoverBackground)
	render.FillRect(dst, image.Rect(gut.Max.X, in.Min.Y, gut.Max.X+1, in.Max.Y), th.InputBorder)

	first := c.scrollY / rowH
	last := (c.scrollY + ta.Dy() - 1) / rowH
	if last >= count {
		last = count - 1
	}
	c.ensureNums(last + 1)

	textView := render.Clip(dst, ta, &c.clipText)
	selStart, selEnd := c.selectionRange()
	gutPad := th.Px(6)

	for i := first; i <= last; i++ {
		y := in.Min.Y + i*rowH - c.scrollY
		baseline := y + th.MonoAscent()

		// Número da linha, alinhado à direita; a do cursor em destaque.
		num := c.numCache[i]
		numColor := th.Placeholder
		if i == c.cursor.Line {
			numColor = th.Text
		}
		th.DrawMono(gutView, num, image.Pt(gut.Max.X-gutPad-len(num)*c.advance, baseline), numColor)

		// Seleção nesta linha (o '\n' selecionado vira um talão de célula).
		if c.focused && c.hasSelection() && i >= selStart.Line && i <= selEnd.Line {
			x0 := 0
			if i == selStart.Line {
				x0 = c.xAt(i, selStart.Col)
			}
			var x1 int
			if i == selEnd.Line {
				x1 = c.xAt(i, selEnd.Col)
			} else {
				x1 = c.lineWidth(i) + c.advance
			}
			if x1 > x0 {
				render.FillRect(textView, image.Rect(ta.Min.X+x0-c.scrollX, y, ta.Min.X+x1-c.scrollX, y+rowH), th.Selection)
			}
		}

		// Texto (com a composição inline na linha do cursor).
		screenX0 := ta.Min.X - c.scrollX
		if c.pre.active() && i == c.cursor.Line {
			x := c.drawTabbedPart(textView, c.preAntes, 0, screenX0, baseline)
			th.DrawMono(textView, c.pre.text, image.Pt(screenX0+x, baseline), th.Text)
			c.pre.drawUnderline(textView, th, screenX0+x, baseline+th.Px(2))
			c.drawTabbedPart(textView, c.preDepois, x+c.pre.w, screenX0, baseline)
		} else {
			c.drawTabbed(textView, c.buf.lineText(i), 0, screenX0, baseline)
		}
	}

	if c.focused && c.caretOn {
		r := c.CaretRect()
		render.FillRect(textView, r, th.Cursor)
	}
	c.drawDisabledOverlay(dst)
}

// drawTabbedPart é o drawTabbed para os trechos ao redor da composição.
func (c *CodeEditor) drawTabbedPart(view *image.RGBA, s string, xContent, screenX0, baseline int) int {
	if s == "" {
		return xContent
	}
	return c.drawTabbed(view, s, xContent, screenX0, baseline)
}
