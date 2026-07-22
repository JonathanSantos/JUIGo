package juigo

import (
	"image"

	"juigo/render"
)

// Input é um campo de texto de linha única. Toda manipulação opera sobre
// []rune — nunca sobre bytes — para que acentuação e qualquer texto UTF-8
// funcionem corretamente. É focável; quando focado, desenha o cursor como
// uma linha vertical e recebe caracteres (CharEvent) e teclas de edição:
// Backspace, Delete, setas, Home e End.
type Input struct {
	BaseWidget
	// Placeholder é exibido quando o campo está vazio e sem foco.
	Placeholder string
	// OnChange é chamado após qualquer alteração no texto. Pode ser nil.
	OnChange func(string)

	theme   *Theme
	runes   []rune
	cursor  int // índice do cursor em runes (0..len(runes))
	focused bool

	// Caches atualizados a cada edição, para que Draw não aloque.
	text    string
	cursorX int
}

// NewInput cria um campo de texto vazio com o tema e o placeholder dados.
func NewInput(theme *Theme, placeholder string) *Input {
	return &Input{theme: theme, Placeholder: placeholder}
}

// Text devolve o conteúdo atual do campo.
func (in *Input) Text() string {
	return in.text
}

// SetText substitui o conteúdo do campo e move o cursor para o fim.
// Não dispara OnChange.
func (in *Input) SetText(s string) {
	in.runes = []rune(s)
	in.cursor = len(in.runes)
	in.sync()
}

// Cursor devolve a posição atual do cursor, em runes.
func (in *Input) Cursor() int {
	return in.cursor
}

// Focusable devolve true: o campo participa da cadeia de foco.
func (in *Input) Focusable() bool {
	return true
}

// PreferredSize devolve a largura mínima do tema e a altura de uma linha
// mais o padding interno.
func (in *Input) PreferredSize() image.Point {
	return image.Point{
		X: in.theme.InputMinWidth,
		Y: in.theme.LineHeight() + 2*in.theme.Padding,
	}
}

// HandleEvent trata caracteres digitados, teclas de edição, foco e clique.
func (in *Input) HandleEvent(ev Event) bool {
	switch e := ev.(type) {
	case CharEvent:
		in.insert(e.Rune)
		return true
	case KeyEvent:
		return in.handleKey(e.Key)
	case FocusEvent:
		in.focused = e.Gained
		return true
	case MouseEvent:
		if e.Kind == MouseDown && e.Button == MouseButtonLeft {
			// Posiciona o cursor no ponto clicado, medindo prefixos com
			// MeasureString (a única fonte de verdade de largura).
			in.cursor = in.runeIndexAt(e.Pos.X)
			in.sync()
			return true
		}
	}
	return false
}

// insert insere r na posição do cursor e avança o cursor.
func (in *Input) insert(r rune) {
	in.runes = append(in.runes, 0)
	copy(in.runes[in.cursor+1:], in.runes[in.cursor:])
	in.runes[in.cursor] = r
	in.cursor++
	in.sync()
	in.emitChange()
}

// handleKey aplica uma tecla de edição. Devolve true se algo mudou.
func (in *Input) handleKey(k Key) bool {
	switch k {
	case KeyBackspace:
		if in.cursor == 0 {
			return false
		}
		in.runes = append(in.runes[:in.cursor-1], in.runes[in.cursor:]...)
		in.cursor--
		in.sync()
		in.emitChange()
		return true
	case KeyDelete:
		if in.cursor >= len(in.runes) {
			return false
		}
		in.runes = append(in.runes[:in.cursor], in.runes[in.cursor+1:]...)
		in.sync()
		in.emitChange()
		return true
	case KeyLeft:
		if in.cursor == 0 {
			return false
		}
		in.cursor--
		in.sync()
		return true
	case KeyRight:
		if in.cursor >= len(in.runes) {
			return false
		}
		in.cursor++
		in.sync()
		return true
	case KeyHome:
		if in.cursor == 0 {
			return false
		}
		in.cursor = 0
		in.sync()
		return true
	case KeyEnd:
		if in.cursor == len(in.runes) {
			return false
		}
		in.cursor = len(in.runes)
		in.sync()
		return true
	}
	return false
}

// sync atualiza os caches derivados (string do texto e X do cursor) após
// qualquer mudança. Alocar aqui é aceitável: acontece por evento de edição,
// nunca por frame desenhado.
func (in *Input) sync() {
	in.text = string(in.runes)
	in.cursorX = in.theme.MeasureString(string(in.runes[:in.cursor]))
}

// emitChange notifica OnChange com o texto atual.
func (in *Input) emitChange() {
	if in.OnChange != nil {
		in.OnChange(in.text)
	}
}

// runeIndexAt devolve o índice de cursor (em runes) mais próximo da
// coordenada X absoluta dada.
func (in *Input) runeIndexAt(x int) int {
	rel := x - (in.Bounds().Min.X + in.theme.Padding)
	if rel <= 0 {
		return 0
	}
	best, bestDist := 0, rel
	for i := 1; i <= len(in.runes); i++ {
		w := in.theme.MeasureString(string(in.runes[:i]))
		d := w - rel
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			best, bestDist = i, d
		}
		if w >= rel {
			break
		}
	}
	return best
}

// Draw desenha o fundo, a borda (realçada com foco), o texto ou o
// placeholder, e o cursor quando focado.
func (in *Input) Draw(dst *image.RGBA) {
	bounds := in.Bounds()
	th := in.theme

	render.FillRect(dst, bounds, th.InputBackground)
	border := th.InputBorder
	if in.focused {
		border = th.InputBorderFocused
	}
	render.StrokeRect(dst, bounds, th.BorderWidth, border)

	textX := bounds.Min.X + th.Padding
	baseline := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()

	switch {
	case len(in.runes) > 0:
		th.DrawText(dst, in.text, image.Pt(textX, baseline), th.Text)
	case !in.focused && in.Placeholder != "":
		th.DrawText(dst, in.Placeholder, image.Pt(textX, baseline), th.Placeholder)
	}

	if in.focused {
		top := baseline - th.Ascent()
		cx := textX + in.cursorX
		render.FillRect(dst, image.Rect(cx, top, cx+th.BorderWidth, top+th.LineHeight()), th.Cursor)
	}
}
