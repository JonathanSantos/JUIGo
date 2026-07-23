package widget

import (
	"image"
	"unicode/utf8"

	"golang.org/x/image/font"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// TextArea é um editor de texto MULTILINHA: linhas separadas por '\n' e
// QUEBRA AUTOMÁTICA (soft wrap) na largura do campo — preferindo espaços,
// quebrando no meio da palavra quando não há —, com rolagem vertical
// automática para manter o cursor visível e rolagem pela roda do mouse.
// Opera sobre []rune, como o Input, com seleção por âncora+cursor
// atravessando linhas. Cima/Baixo navegam por linhas VISUAIS (preservando a
// coluna desejada); Home/End vão ao início/fim da linha REAL.
//
// Suporta: Enter insere linha; setas em todas as direções (Cima/Baixo
// preservam a coluna desejada); Home/End vão ao início/fim da LINHA;
// Shift+movimento seleciona; Ctrl/Cmd+A/C/X/V com quebras de linha
// preservadas; clique e arraste posicionam e selecionam; cursor piscante.
//
// Nasce pronto para reatividade: BindValue vincula o conteúdo a um
// State[string] em duas vias.
type TextArea struct {
	BaseWidget
	// Placeholder é exibido quando o campo está vazio e sem foco.
	Placeholder string

	// Callbacks definidos pelos métodos encadeáveis de mesmo nome.
	onChange func(string)
	onFocus  func()
	onBlur   func()
	// invalid liga o realce de erro (ver SetInvalid/BindInvalid).
	invalid bool

	runes     []rune
	cursor    int
	anchor    int
	selecting bool
	focused   bool
	bound     *state.State[string]

	// Caches reconstruídos em sync (por evento, não por frame).
	text       string
	lines      []string
	lineStarts []int
	caretLine  int
	caretX     int
	anchorLine int
	anchorX    int
	syncFace   font.Face

	// goalX é a coluna (px) desejada nas navegações verticais; -1 usa a
	// posição atual do cursor.
	goalX   int
	scrollY int
	accum   float64
	clip    image.RGBA

	// Soft wrap: vlines são as linhas VISUAIS (cada linha real quebrada na
	// largura útil), recalculadas quando texto, largura ou fonte mudam.
	vlines    []vline
	wrapWidth int
	wrapFace  font.Face
	wrapGen   int
	textGen   int

	caretOn     bool
	blinkCancel func()
}

// NewTextArea cria um editor vazio com o placeholder dado. O tema é herdado
// no mount.
func NewTextArea(placeholder string) *TextArea {
	t := &TextArea{Placeholder: placeholder, goalX: -1, caretOn: true}
	t.sync()
	return t
}

// OnChange define o callback chamado após qualquer alteração no texto.
// Encadeável.
func (t *TextArea) OnChange(fn func(string)) *TextArea {
	t.onChange = fn
	return t
}

// OnFocus define o callback chamado ao ganhar o foco de teclado. Encadeável.
func (t *TextArea) OnFocus(fn func()) *TextArea {
	t.onFocus = fn
	return t
}

// OnBlur define o callback chamado ao perder o foco de teclado — útil para
// validação "touched" (ver juigo/form). Encadeável.
func (t *TextArea) OnBlur(fn func()) *TextArea {
	t.onBlur = fn
	return t
}

// SetInvalid liga/desliga o realce de erro do campo (borda na cor
// Theme.Danger) e agenda um redesenho.
func (t *TextArea) SetInvalid(v bool) {
	if t.invalid == v {
		return
	}
	t.invalid = v
	t.Invalidate()
}

// BindInvalid vincula o realce de erro ao State (true = borda Danger) — o
// par visual de form.ErrorOf. Encadeável.
func (t *TextArea) BindInvalid(s *state.State[bool]) *TextArea {
	t.SetInvalid(s.Get())
	s.Watch(func(v bool) { t.SetInvalid(v) })
	return t
}

// Text devolve o conteúdo atual.
func (t *TextArea) Text() string {
	return t.text
}

// SetText substitui o conteúdo, move o cursor para o fim e agenda um
// redesenho. Não dispara OnChange.
func (t *TextArea) SetText(s string) {
	t.runes = []rune(s)
	t.cursor = len(t.runes)
	t.anchor = t.cursor
	t.goalX = -1
	t.sync()
	t.Invalidate()
}

// Cursor devolve a posição do cursor, em runes.
func (t *TextArea) Cursor() int {
	return t.cursor
}

// BindValue vincula o conteúdo ao State em DUAS vias, como no Input.
// Encadeável.
func (t *TextArea) BindValue(s *state.State[string]) *TextArea {
	t.bound = s
	t.SetText(s.Get())
	s.Watch(func(v string) {
		if t.text != v {
			t.SetText(v)
		}
	})
	return t
}

// Focusable devolve true.
func (t *TextArea) Focusable() bool {
	return true
}

// CursorShape da TextArea: I-beam.
func (t *TextArea) CursorShape() CursorShape {
	return CursorText
}

// PreferredSize devolve a largura mínima do Input e TextAreaMinLines linhas.
func (t *TextArea) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: t.theme.InputMinWidthPx(),
		Y: t.theme.TextAreaMinLines*t.theme.LineHeight() + 2*t.theme.PaddingPx(),
	}
}

// sync reconstrói os caches (linhas, posições do cursor/âncora) e reinicia a
// piscada. Aloca — mas por evento de edição, nunca por frame.
func (t *TextArea) sync() {
	t.text = string(t.runes)
	t.textGen++
	t.lineStarts = t.lineStarts[:0]
	t.lineStarts = append(t.lineStarts, 0)
	for i, r := range t.runes {
		if r == '\n' {
			t.lineStarts = append(t.lineStarts, i+1)
		}
	}
	t.lines = t.lines[:0]
	for li, start := range t.lineStarts {
		end := len(t.runes)
		if li+1 < len(t.lineStarts) {
			end = t.lineStarts[li+1] - 1 // sem o '\n'
		}
		t.lines = append(t.lines, string(t.runes[start:end]))
	}
	if t.theme != nil {
		t.caretLine = t.lineOf(t.cursor)
		t.caretX = t.theme.MeasureString(string(t.runes[t.lineStarts[t.caretLine]:t.cursor]))
		t.anchorLine = t.lineOf(t.anchor)
		t.anchorX = t.theme.MeasureString(string(t.runes[t.lineStarts[t.anchorLine]:t.anchor]))
		t.syncFace = t.theme.Face
	} else {
		t.caretLine, t.caretX, t.anchorLine, t.anchorX, t.syncFace = 0, 0, 0, 0, nil
	}
	t.restartBlink()
}

// lineOf devolve a linha que contém o índice dado.
func (t *TextArea) lineOf(idx int) int {
	li := 0
	for li+1 < len(t.lineStarts) && t.lineStarts[li+1] <= idx {
		li++
	}
	return li
}

// lineEnd devolve o índice (em runes) do fim da linha, sem o '\n'.
func (t *TextArea) lineEnd(li int) int {
	if li+1 < len(t.lineStarts) {
		return t.lineStarts[li+1] - 1
	}
	return len(t.runes)
}

// vline é uma linha VISUAL: um trecho [startCol, endCol) de runes da linha
// real hard, resultado do soft wrap.
type vline struct {
	hard     int
	startCol int
	endCol   int
}

// segment devolve o texto da vline, sem alocar (fatia de bytes da linha).
func (t *TextArea) segment(v vline) string {
	line := t.lines[v.hard]
	b0 := len(runePrefix(line, v.startCol))
	b1 := len(runePrefix(line, v.endCol))
	return line[b0:b1]
}

// startIdx devolve o índice absoluto (em runes) do início da vline.
func (t *TextArea) startIdx(v vline) int {
	return t.lineStarts[v.hard] + v.startCol
}

// innerWidth devolve a largura útil de texto do campo.
func (t *TextArea) innerWidth() int {
	return t.Bounds().Dx() - 2*t.theme.PaddingPx()
}

// ensureWrap recalcula as linhas visuais se texto, largura ou fonte mudaram.
func (t *TextArea) ensureWrap() {
	if t.theme == nil {
		return
	}
	w := t.innerWidth()
	if w <= 0 {
		w = 1 << 20 // sem layout ainda: uma vline por linha real
	}
	if t.wrapWidth == w && t.wrapFace == t.theme.Face && t.wrapGen == t.textGen {
		return
	}
	t.wrapWidth, t.wrapFace, t.wrapGen = w, t.theme.Face, t.textGen

	t.vlines = t.vlines[:0]
	for li, line := range t.lines {
		n := t.lineEnd(li) - t.lineStarts[li]
		if n == 0 || t.theme.MeasureString(line) <= w {
			t.vlines = append(t.vlines, vline{hard: li, startCol: 0, endCol: n})
			continue
		}
		col := 0
		for col < n {
			fit := t.fitCols(li, col, n, w)
			if fit < n {
				// Recua até o último espaço do trecho (o espaço fecha a
				// linha visual); sem espaço, quebra no meio da palavra.
				for j := fit; j > col+1; j-- {
					if t.runes[t.lineStarts[li]+j-1] == ' ' {
						fit = j
						break
					}
				}
			}
			t.vlines = append(t.vlines, vline{hard: li, startCol: col, endCol: fit})
			col = fit
		}
	}
}

// fitCols devolve, por busca binária, o maior fim em (col, n] cujo trecho
// [col, fim) cabe em w px (pelo menos uma rune).
func (t *TextArea) fitCols(li, col, n, w int) int {
	line := t.lines[li]
	base := len(runePrefix(line, col))
	lo, hi := col+1, n
	for lo < hi {
		mid := (lo + hi + 1) / 2
		end := len(runePrefix(line, mid))
		if t.theme.MeasureString(line[base:end]) <= w {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// visualOf devolve a vline que contém o índice absoluto idx. No exato ponto
// de quebra, o caret pertence à linha visual SEGUINTE (afinidade para
// baixo), exceto no fim real da linha.
func (t *TextArea) visualOf(idx int) int {
	for i, v := range t.vlines {
		end := t.startIdx(v) + (v.endCol - v.startCol)
		if idx < end {
			return i
		}
		if idx == end {
			lastOfHard := i+1 >= len(t.vlines) || t.vlines[i+1].hard != v.hard
			if lastOfHard {
				return i
			}
		}
	}
	if len(t.vlines) == 0 {
		return 0
	}
	return len(t.vlines) - 1
}

// visualX devolve o deslocamento em px de idx dentro da vline vli.
func (t *TextArea) visualX(idx, vli int) int {
	v := t.vlines[vli]
	line := t.lines[v.hard]
	b0 := len(runePrefix(line, v.startCol))
	b1 := len(runePrefix(line, v.startCol+(idx-t.startIdx(v))))
	return t.theme.MeasureString(line[b0:b1])
}

// indexAtVisual devolve o índice absoluto mais próximo de x px na vline.
func (t *TextArea) indexAtVisual(vli, x int) int {
	if vli < 0 {
		return 0
	}
	if vli >= len(t.vlines) {
		return len(t.runes)
	}
	v := t.vlines[vli]
	seg := t.segment(v)
	cols := v.endCol - v.startCol
	if x <= 0 {
		return t.startIdx(v)
	}
	best, bestDist := cols, abs(t.theme.MeasureString(seg)-x)
	for i := 0; i <= cols; i++ {
		w := t.theme.MeasureString(runePrefix(seg, i))
		if d := abs(w - x); d < bestDist {
			best, bestDist = i, d
		}
		if w >= x {
			break
		}
	}
	return t.startIdx(v) + best
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// indexAt converte uma posição absoluta do mouse em índice de rune, pela
// linha VISUAL sob o ponto.
func (t *TextArea) indexAt(p image.Point) int {
	if t.theme == nil {
		return 0
	}
	t.ensureWrap()
	th := t.theme
	vli := (p.Y - (t.Bounds().Min.Y + th.PaddingPx()) + t.scrollY) / th.LineHeight()
	if vli < 0 {
		vli = 0
	}
	if vli >= len(t.vlines) {
		vli = len(t.vlines) - 1
	}
	return t.indexAtVisual(vli, p.X-(t.Bounds().Min.X+th.PaddingPx()))
}

// HandleEvent trata texto, teclas, rolagem, foco, clique e arraste.
func (t *TextArea) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.CharEvent:
		t.insert(e.Rune)
		return true
	case event.KeyEvent:
		return t.handleKey(e)
	case event.FocusEvent:
		t.focused = e.Gained
		if e.Gained {
			t.restartBlink()
			if t.onFocus != nil {
				t.onFocus()
			}
		} else {
			t.stopBlink()
			if t.onBlur != nil {
				t.onBlur()
			}
		}
		return true
	case event.ScrollEvent:
		return t.handleScroll(e)
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			t.selecting = true
			t.goalX = -1
			t.moveCursor(t.indexAt(e.Pos), false)
			return true
		case event.MouseMove:
			if !t.selecting {
				return false
			}
			return t.moveCursor(t.indexAt(e.Pos), true)
		case event.MouseUp:
			t.selecting = false
			return false
		}
	}
	return false
}

// handleScroll rola o conteúdo (deltas fracionários acumulados); nos limites
// propaga, como o Scroll.
func (t *TextArea) handleScroll(e event.ScrollEvent) bool {
	th := t.theme
	if th == nil {
		return false
	}
	max := t.contentHeight() - t.innerHeight()
	if max <= 0 {
		return false
	}
	delta := e.DY*float64(th.Px(th.ScrollStep)) + t.accum
	step := int(delta)
	t.accum = delta - float64(step)
	novo := t.scrollY - step
	if novo < 0 {
		novo = 0
	}
	if novo > max {
		novo = max
	}
	if novo == t.scrollY {
		if step != 0 {
			t.accum = 0
			return false
		}
		return true
	}
	t.scrollY = novo
	return true
}

func (t *TextArea) contentHeight() int {
	t.ensureWrap()
	return len(t.vlines) * t.theme.LineHeight()
}

func (t *TextArea) innerHeight() int {
	return t.Bounds().Dy() - 2*t.theme.PaddingPx()
}

// handleKey aplica edição, navegação e atalhos.
func (t *TextArea) handleKey(e event.KeyEvent) bool {
	switch e.Key {
	case event.KeyEnter:
		t.insert('\n')
		return true
	case event.KeyBackspace:
		t.goalX = -1
		if t.deleteSelection() {
			t.sync()
			t.emitChange()
			return true
		}
		if t.cursor == 0 {
			return false
		}
		t.runes = append(t.runes[:t.cursor-1], t.runes[t.cursor:]...)
		t.cursor--
		t.anchor = t.cursor
		t.sync()
		t.emitChange()
		return true
	case event.KeyDelete:
		t.goalX = -1
		if t.deleteSelection() {
			t.sync()
			t.emitChange()
			return true
		}
		if t.cursor >= len(t.runes) {
			return false
		}
		t.runes = append(t.runes[:t.cursor], t.runes[t.cursor+1:]...)
		t.anchor = t.cursor
		t.sync()
		t.emitChange()
		return true
	case event.KeyLeft:
		t.goalX = -1
		if !e.Mods.Shift() && t.hasSelection() {
			s, _ := t.selection()
			return t.moveCursor(s, false)
		}
		return t.moveCursor(t.cursor-1, e.Mods.Shift())
	case event.KeyRight:
		t.goalX = -1
		if !e.Mods.Shift() && t.hasSelection() {
			_, end := t.selection()
			return t.moveCursor(end, false)
		}
		return t.moveCursor(t.cursor+1, e.Mods.Shift())
	case event.KeyUp, event.KeyDown:
		t.ensureWrap()
		cur := t.visualOf(t.cursor)
		if t.goalX < 0 {
			t.goalX = t.visualX(t.cursor, cur)
		}
		target := cur - 1
		if e.Key == event.KeyDown {
			target = cur + 1
		}
		if target < 0 {
			return t.moveCursor(0, e.Mods.Shift())
		}
		if target >= len(t.vlines) {
			return t.moveCursor(len(t.runes), e.Mods.Shift())
		}
		return t.moveCursor(t.indexAtVisual(target, t.goalX), e.Mods.Shift())
	case event.KeyHome:
		t.goalX = -1
		return t.moveCursor(t.lineStarts[t.caretLine], e.Mods.Shift())
	case event.KeyEnd:
		t.goalX = -1
		return t.moveCursor(t.lineEnd(t.caretLine), e.Mods.Shift())
	case event.KeyA:
		if !e.Mods.Command() {
			return false
		}
		t.anchor, t.cursor = 0, len(t.runes)
		t.sync()
		return true
	case event.KeyC:
		if !e.Mods.Command() {
			return false
		}
		return t.copySelection()
	case event.KeyX:
		if !e.Mods.Command() || !t.copySelection() {
			return false
		}
		t.deleteSelection()
		t.sync()
		t.emitChange()
		return true
	case event.KeyV:
		if !e.Mods.Command() {
			return false
		}
		return t.paste()
	}
	return false
}

// moveCursor move o cursor (limitado ao texto); com extend mantém a âncora.
func (t *TextArea) moveCursor(pos int, extend bool) bool {
	if pos < 0 {
		pos = 0
	}
	if pos > len(t.runes) {
		pos = len(t.runes)
	}
	changed := pos != t.cursor || (!extend && t.anchor != pos)
	t.cursor = pos
	if !extend {
		t.anchor = pos
	}
	if changed {
		t.sync()
	}
	return changed
}

func (t *TextArea) selection() (start, end int) {
	if t.anchor <= t.cursor {
		return t.anchor, t.cursor
	}
	return t.cursor, t.anchor
}

func (t *TextArea) hasSelection() bool {
	return t.anchor != t.cursor
}

func (t *TextArea) deleteSelection() bool {
	if !t.hasSelection() {
		return false
	}
	start, end := t.selection()
	t.runes = append(t.runes[:start], t.runes[end:]...)
	t.cursor, t.anchor = start, start
	return true
}

func (t *TextArea) copySelection() bool {
	if !t.hasSelection() {
		return false
	}
	start, end := t.selection()
	hooks.WriteClipboard(string(t.runes[start:end]))
	return true
}

// paste cola preservando quebras de linha e tabs (demais controles são
// descartados).
func (t *TextArea) paste() bool {
	var clean []rune
	for _, r := range hooks.ReadClipboard() {
		if r >= 0x20 && r != 0x7F || r == '\n' || r == '\t' {
			clean = append(clean, r)
		}
	}
	if len(clean) == 0 {
		return false
	}
	t.goalX = -1
	t.deleteSelection()
	out := make([]rune, 0, len(t.runes)+len(clean))
	out = append(out, t.runes[:t.cursor]...)
	out = append(out, clean...)
	out = append(out, t.runes[t.cursor:]...)
	t.runes = out
	t.cursor += len(clean)
	t.anchor = t.cursor
	t.sync()
	t.emitChange()
	return true
}

// insert insere r no cursor (substituindo a seleção).
func (t *TextArea) insert(r rune) {
	t.goalX = -1
	t.deleteSelection()
	t.runes = append(t.runes, 0)
	copy(t.runes[t.cursor+1:], t.runes[t.cursor:])
	t.runes[t.cursor] = r
	t.cursor++
	t.anchor = t.cursor
	t.sync()
	t.emitChange()
}

func (t *TextArea) emitChange() {
	if t.bound != nil && t.bound.Get() != t.text {
		t.bound.Set(t.text)
	}
	if t.onChange != nil {
		t.onChange(t.text)
	}
}

// restartBlink/blinkTick/stopBlink: mesma piscada do Input.
func (t *TextArea) restartBlink() {
	t.stopBlink()
	t.caretOn = true
	if !t.focused || t.theme == nil || t.theme.CaretBlink <= 0 {
		return
	}
	t.blinkCancel = hooks.ScheduleAfter(t.theme.CaretBlink, t.blinkTick)
}

func (t *TextArea) blinkTick() {
	if !t.focused {
		return
	}
	t.caretOn = !t.caretOn
	t.Invalidate()
	t.blinkCancel = hooks.ScheduleAfter(t.theme.CaretBlink, t.blinkTick)
}

func (t *TextArea) stopBlink() {
	if t.blinkCancel != nil {
		t.blinkCancel()
		t.blinkCancel = nil
	}
	t.caretOn = true
}

// runePrefix devolve o prefixo de s com n runes, sem alocar.
func runePrefix(s string, n int) string {
	b := 0
	for i := 0; i < n && b < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[b:])
		b += size
	}
	return s[:b]
}

// Draw desenha fundo, borda, seleção multilinha, texto por linha e o cursor,
// tudo recortado à área interna, com rolagem vertical mantendo o cursor
// visível.
func (t *TextArea) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	th := t.theme
	bounds := t.Bounds()
	if t.syncFace != th.Face {
		t.sync()
	}

	render.FillRect(dst, bounds, th.InputBackground)
	border := th.InputBorder
	if t.focused {
		border = th.InputBorderFocused
	}
	if t.invalid {
		border = th.Danger // erro vence o foco: o problema fica visível
	}
	render.StrokeRect(dst, bounds, th.BorderPx(), border)

	pad := th.PaddingPx()
	lineH := th.LineHeight()
	innerTop := bounds.Min.Y + pad
	innerH := t.innerHeight()
	if innerH <= 0 {
		return
	}

	// Quebra visual atualizada para a largura corrente.
	t.ensureWrap()
	caretVLine := t.visualOf(t.cursor)

	// Rolagem vertical: limita e garante a linha VISUAL do cursor visível.
	if max := len(t.vlines)*lineH - innerH; t.scrollY > max {
		t.scrollY = max
	}
	if t.scrollY < 0 {
		t.scrollY = 0
	}
	if t.focused {
		caretTop := caretVLine * lineH
		if caretTop-t.scrollY < 0 {
			t.scrollY = caretTop
		}
		if caretTop+lineH-t.scrollY > innerH {
			t.scrollY = caretTop + lineH - innerH
		}
	}

	textX := bounds.Min.X + pad
	view := render.Clip(dst, image.Rect(bounds.Min.X+th.BorderPx(), innerTop, bounds.Max.X-th.BorderPx(), innerTop+innerH), &t.clip)

	if len(t.runes) == 0 && !t.focused && t.Placeholder != "" {
		th.DrawText(view, t.Placeholder, image.Pt(textX, innerTop+th.Ascent()), th.Placeholder)
		return
	}

	selStart, selEnd := t.selection()
	spaceW := th.MeasureString(" ")
	for vi, v := range t.vlines {
		y := innerTop + vi*lineH - t.scrollY
		if y+lineH < innerTop || y > innerTop+innerH {
			continue
		}
		seg := t.segment(v)
		vs := t.startIdx(v)
		ve := vs + (v.endCol - v.startCol)
		// Seleção nesta linha visual; no fim da linha REAL, o '\n'
		// selecionado vira um talão de espaço.
		if t.focused && selStart != selEnd && selStart < ve+1 && selEnd > vs {
			s, e := selStart, selEnd
			if s < vs {
				s = vs
			}
			lastOfHard := vi+1 >= len(t.vlines) || t.vlines[vi+1].hard != v.hard
			x0 := th.MeasureString(runePrefix(seg, s-vs))
			var x1 int
			if e > ve {
				x1 = th.MeasureString(seg)
				if lastOfHard {
					x1 += spaceW
				}
			} else {
				x1 = th.MeasureString(runePrefix(seg, e-vs))
			}
			if x1 > x0 || (e > ve && lastOfHard) {
				render.FillRect(view, image.Rect(textX+x0, y, textX+x1, y+lineH), th.Selection)
			}
		}
		th.DrawText(view, seg, image.Pt(textX, y+th.Ascent()), th.Text)
	}

	if t.focused && t.caretOn {
		y := innerTop + caretVLine*lineH - t.scrollY
		cx := textX + t.visualX(t.cursor, caretVLine)
		render.FillRect(view, image.Rect(cx, y, cx+th.BorderPx(), y+lineH), th.Cursor)
	}
	t.drawDisabledOverlay(dst)
}
