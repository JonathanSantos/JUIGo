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
//
// Suporta seleção de texto — arrastando com o mouse (via captura do App) ou
// com Shift+setas/Home/End — e os atalhos de edição com o modificador de
// comando (Ctrl ou Cmd): A seleciona tudo, C copia, X recorta e V cola
// usando a área de transferência do sistema. Digitar ou colar substitui a
// seleção; Backspace/Delete a apagam.
type Input struct {
	BaseWidget
	// Placeholder é exibido quando o campo está vazio e sem foco.
	Placeholder string
	// OnChange é chamado após qualquer alteração no texto. Pode ser nil.
	OnChange func(string)

	runes []rune
	// cursor e anchor são índices em runes (0..len(runes)); a seleção é o
	// intervalo entre eles e não existe quando são iguais. O cursor é a
	// ponta ativa (a que se move com Shift).
	cursor    int
	anchor    int
	selecting bool // arraste de seleção com o mouse em andamento
	focused   bool
	bound     *State[string] // binding de duas vias (ver BindValue)

	// Caches atualizados a cada edição, para que Draw não aloque.
	// syncScale registra a escala do tema usada no último sync: se a escala
	// mudar (ex.: janela movida para outro monitor), são recalculados.
	text      string
	cursorX   int
	anchorX   int
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

// SetText substitui o conteúdo do campo, move o cursor para o fim (limpando
// a seleção) e agenda um redesenho. Não dispara OnChange.
func (in *Input) SetText(s string) {
	in.runes = []rune(s)
	in.cursor = len(in.runes)
	in.anchor = in.cursor
	in.sync()
	requestRepaint()
}

// BindValue vincula o conteúdo do campo ao State em DUAS vias: edições do
// usuário fazem Set no State, e um Set externo atualiza o campo (movendo o
// cursor para o fim). Encadeável.
func (in *Input) BindValue(s *State[string]) *Input {
	in.bound = s
	in.SetText(s.Get())
	s.Watch(func(v string) {
		// Guarda contra eco: quando o Set veio de uma edição deste próprio
		// campo, o texto já está atualizado e nada precisa ser feito.
		if in.text != v {
			in.SetText(v)
		}
	})
	return in
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

// HandleEvent trata caracteres digitados, teclas de edição e atalhos, foco,
// clique e arraste de seleção.
func (in *Input) HandleEvent(ev Event) bool {
	switch e := ev.(type) {
	case CharEvent:
		in.insert(e.Rune)
		return true
	case KeyEvent:
		return in.handleKey(e)
	case FocusEvent:
		in.focused = e.Gained
		return true
	case MouseEvent:
		switch e.Kind {
		case MouseDown:
			if e.Button != MouseButtonLeft {
				return false
			}
			// Posiciona o cursor no ponto clicado (via MeasureString, a
			// única fonte de verdade de largura) e inicia a seleção por
			// arraste — os MouseMove seguintes chegam pela captura do App.
			in.selecting = true
			in.moveCursor(in.runeIndexAt(e.Pos.X), false)
			return true
		case MouseMove:
			if !in.selecting {
				return false
			}
			return in.moveCursor(in.runeIndexAt(e.Pos.X), true)
		case MouseUp:
			in.selecting = false
			return false
		}
	}
	return false
}

// insert insere r na posição do cursor (substituindo a seleção, se houver) e
// avança o cursor.
func (in *Input) insert(r rune) {
	in.deleteSelection()
	in.runes = append(in.runes, 0)
	copy(in.runes[in.cursor+1:], in.runes[in.cursor:])
	in.runes[in.cursor] = r
	in.cursor++
	in.anchor = in.cursor
	in.sync()
	in.emitChange()
}

// handleKey aplica uma tecla de edição ou atalho. Devolve true se algo
// mudou (texto, cursor ou seleção).
func (in *Input) handleKey(e KeyEvent) bool {
	switch e.Key {
	case KeyBackspace:
		if in.deleteSelection() {
			in.sync()
			in.emitChange()
			return true
		}
		if in.cursor == 0 {
			return false
		}
		in.runes = append(in.runes[:in.cursor-1], in.runes[in.cursor:]...)
		in.cursor--
		in.anchor = in.cursor
		in.sync()
		in.emitChange()
		return true
	case KeyDelete:
		if in.deleteSelection() {
			in.sync()
			in.emitChange()
			return true
		}
		if in.cursor >= len(in.runes) {
			return false
		}
		in.runes = append(in.runes[:in.cursor], in.runes[in.cursor+1:]...)
		in.anchor = in.cursor
		in.sync()
		in.emitChange()
		return true
	case KeyLeft:
		// Sem Shift, uma seleção existente recolhe para a própria borda.
		if !e.Mods.Shift() && in.hasSelection() {
			start, _ := in.selection()
			return in.moveCursor(start, false)
		}
		return in.moveCursor(in.cursor-1, e.Mods.Shift())
	case KeyRight:
		if !e.Mods.Shift() && in.hasSelection() {
			_, end := in.selection()
			return in.moveCursor(end, false)
		}
		return in.moveCursor(in.cursor+1, e.Mods.Shift())
	case KeyHome:
		return in.moveCursor(0, e.Mods.Shift())
	case KeyEnd:
		return in.moveCursor(len(in.runes), e.Mods.Shift())
	case KeyA:
		if !e.Mods.Command() {
			return false
		}
		in.anchor, in.cursor = 0, len(in.runes)
		in.sync()
		return true
	case KeyC:
		if !e.Mods.Command() {
			return false
		}
		return in.copySelection()
	case KeyX:
		if !e.Mods.Command() || !in.copySelection() {
			return false
		}
		in.deleteSelection()
		in.sync()
		in.emitChange()
		return true
	case KeyV:
		if !e.Mods.Command() {
			return false
		}
		return in.paste()
	}
	return false
}

// moveCursor move o cursor para pos (limitado ao texto). Com extend, a
// âncora fica onde está (estendendo a seleção); sem, recolhe para o cursor.
// Devolve true se cursor ou âncora mudaram.
func (in *Input) moveCursor(pos int, extend bool) bool {
	if pos < 0 {
		pos = 0
	}
	if pos > len(in.runes) {
		pos = len(in.runes)
	}
	changed := pos != in.cursor || (!extend && in.anchor != pos)
	in.cursor = pos
	if !extend {
		in.anchor = pos
	}
	if changed {
		in.sync()
	}
	return changed
}

// selection devolve os limites ordenados da seleção, em runes.
func (in *Input) selection() (start, end int) {
	if in.anchor <= in.cursor {
		return in.anchor, in.cursor
	}
	return in.cursor, in.anchor
}

// hasSelection informa se há um trecho selecionado.
func (in *Input) hasSelection() bool {
	return in.anchor != in.cursor
}

// deleteSelection remove o trecho selecionado, deixando cursor e âncora no
// início dele. Devolve true se havia seleção. Não faz sync nem emite
// OnChange — responsabilidade do chamador.
func (in *Input) deleteSelection() bool {
	if !in.hasSelection() {
		return false
	}
	start, end := in.selection()
	in.runes = append(in.runes[:start], in.runes[end:]...)
	in.cursor, in.anchor = start, start
	return true
}

// copySelection copia o trecho selecionado para a área de transferência.
// Devolve true se havia seleção.
func (in *Input) copySelection() bool {
	if !in.hasSelection() {
		return false
	}
	start, end := in.selection()
	clipboardWriteText(string(in.runes[start:end]))
	return true
}

// paste cola o texto da área de transferência na posição do cursor,
// substituindo a seleção. Caracteres de controle (quebras de linha, tabs)
// são descartados: o campo é de linha única. Devolve true se algo mudou.
func (in *Input) paste() bool {
	var clean []rune
	for _, r := range clipboardReadText() {
		if r >= 0x20 && r != 0x7F {
			clean = append(clean, r)
		}
	}
	if len(clean) == 0 {
		return false
	}
	in.deleteSelection()
	out := make([]rune, 0, len(in.runes)+len(clean))
	out = append(out, in.runes[:in.cursor]...)
	out = append(out, clean...)
	out = append(out, in.runes[in.cursor:]...)
	in.runes = out
	in.cursor += len(clean)
	in.anchor = in.cursor
	in.sync()
	in.emitChange()
	return true
}

// sync atualiza os caches derivados (string do texto e X do cursor) após
// qualquer mudança. Alocar aqui é aceitável: acontece por evento de edição
// (ou uma única vez após mudança de escala), nunca por frame desenhado.
func (in *Input) sync() {
	in.text = string(in.runes)
	if in.theme == nil {
		// Antes do mount não há como medir; Draw refaz o sync ao detectar a
		// mudança de escala (0 → escala do tema).
		in.cursorX, in.anchorX, in.syncScale = 0, 0, 0
		return
	}
	in.cursorX = in.theme.MeasureString(string(in.runes[:in.cursor]))
	in.anchorX = in.theme.MeasureString(string(in.runes[:in.anchor]))
	in.syncScale = in.theme.Scale()
}

// emitChange propaga uma edição para o State vinculado (se houver) e para o
// callback OnChange.
func (in *Input) emitChange() {
	if in.bound != nil && in.bound.Get() != in.text {
		in.bound.Set(in.text)
	}
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

	if in.focused && in.hasSelection() {
		sx, ex := in.anchorX, in.cursorX
		if sx > ex {
			sx, ex = ex, sx
		}
		top := baseline - th.Ascent()
		render.FillRect(dst, image.Rect(textX+sx, top, textX+ex, top+th.LineHeight()), th.Selection)
	}

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
