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

	runes   []rune
	cursor  int // índice do cursor em runes (0..len(runes))
	focused bool

	// Caches atualizados a cada edição, para que Draw não aloque.
	// syncScale registra a escala do tema usada no último sync: se a escala
	// mudar (ex.: janela movida para outro monitor), cursorX é recalculado.
	text      string
	cursorX   int
	syncScale float64
}

// NewInput cria um campo de texto vazio com o placeholder dado. O tema é
// herdado no mount.
func NewInput(placeholder string) *Input {
	return &Input{Placeholder: placeholder}
}

// Text devolve o conteúdo atual do campo.
func (in *Input) Text() string {
	return in.text
}

// SetText substitui o conteúdo do campo, move o cursor para o fim e agenda
// um redesenho. Não dispara OnChange.
func (in *Input) SetText(s string) {
	in.runes = []rune(s)
	in.cursor = len(in.runes)
	in.sync()
	requestRepaint()
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
// mais o padding interno. Antes do mount (sem tema), devolve zero.
func (in *Input) PreferredSize() image.Point {
	if in.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: in.theme.InputMinWidthPx(),
		Y: in.theme.LineHeight() + 2*in.theme.PaddingPx(),
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
// qualquer mudança. Alocar aqui é aceitável: acontece por evento de edição
// (ou uma única vez após mudança de escala), nunca por frame desenhado.
func (in *Input) sync() {
	in.text = string(in.runes)
	if in.theme == nil {
		// Antes do mount não há como medir; Draw refaz o sync ao detectar a
		// mudança de escala (0 → escala do tema).
		in.cursorX, in.syncScale = 0, 0
		return
	}
	in.cursorX = in.theme.MeasureString(string(in.runes[:in.cursor]))
	in.syncScale = in.theme.Scale()
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
	if in.theme == nil {
		return 0
	}
	rel := x - (in.Bounds().Min.X + in.theme.PaddingPx())
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
	if in.theme == nil {
		return
	}
	bounds := in.Bounds()
	th := in.theme

	// A escala mudou desde o último sync? Recalcula cursorX uma única vez.
	if in.syncScale != th.Scale() {
		in.sync()
	}

	render.FillRect(dst, bounds, th.InputBackground)
	border := th.InputBorder
	if in.focused {
		border = th.InputBorderFocused
	}
	render.StrokeRect(dst, bounds, th.BorderPx(), border)

	textX := bounds.Min.X + th.PaddingPx()
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
		render.FillRect(dst, image.Rect(cx, top, cx+th.BorderPx(), top+th.LineHeight()), th.Cursor)
	}
}
