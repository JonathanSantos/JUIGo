package widget

import (
	"image"

	"golang.org/x/image/font"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/render"
	"juigo/state"
)

// Input é um campo de texto de linha única. Toda manipulação opera sobre
// []rune — nunca sobre bytes — para que acentuação e qualquer texto UTF-8
// funcionem corretamente. É focável; quando focado, desenha o cursor como
// uma linha vertical e recebe caracteres (event.CharEvent) e teclas de edição:
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

	// Callbacks definidos pelos métodos encadeáveis de mesmo nome.
	onChange func(string)
	onFocus  func()
	onBlur   func()
	onSubmit func()
	// filter restringe os caracteres aceitos (ver Filter).
	filter func(rune) bool

	runes []rune
	// cursor e anchor são índices em runes (0..len(runes)); a seleção é o
	// intervalo entre eles e não existe quando são iguais. O cursor é a
	// ponta ativa (a que se move com Shift).
	cursor    int
	anchor    int
	selecting bool // arraste de seleção com o mouse em andamento
	focused   bool
	bound     *state.State[string] // binding de duas vias (ver BindValue)

	// Caches atualizados a cada edição, para que Draw não aloque.
	// syncFace registra a face usada no último sync: se ela mudar (escala
	// nova OU troca de tema em runtime), as medidas são recalculadas.
	text     string
	cursorX  int
	anchorX  int
	textW    int
	syncFace font.Face

	// scrollX é a rolagem horizontal do texto, em pixels: quando o texto é
	// maior que a área útil, o campo rola para manter o cursor visível.
	scrollX int
	// clip é a visão recortada reutilizada pelo Draw (sem alocação).
	clip image.RGBA

	// caretOn e blinkCancel implementam a piscada do cursor: com foco, um
	// timer do App (Theme.CaretBlink) alterna a visibilidade; qualquer
	// edição ou movimento a reinicia com o cursor visível.
	caretOn     bool
	blinkCancel func()
}

// NewInput cria um campo de texto vazio com o placeholder dado. O tema é
// herdado no mount.
func NewInput(placeholder string) *Input {
	return &Input{Placeholder: placeholder, caretOn: true}
}

// OnChange define o callback chamado após qualquer alteração no texto.
// Encadeável.
func (in *Input) OnChange(fn func(string)) *Input {
	in.onChange = fn
	return in
}

// OnFocus define o callback chamado ao ganhar o foco de teclado. Encadeável.
func (in *Input) OnFocus(fn func()) *Input {
	in.onFocus = fn
	return in
}

// OnBlur define o callback chamado ao perder o foco de teclado — útil para
// validação "touched" (ver juigo/form). Encadeável.
func (in *Input) OnBlur(fn func()) *Input {
	in.onBlur = fn
	return in
}

// OnSubmit define o callback chamado quando Enter é pressionado com o campo
// focado (envio de formulários). Encadeável.
func (in *Input) OnSubmit(fn func()) *Input {
	in.onSubmit = fn
	return in
}

// Filter restringe os caracteres aceitos pelo campo: runes reprovadas por
// allowed são ignoradas na digitação e na colagem (campos numéricos, por
// exemplo). Não afeta SetText nem bindings. Encadeável.
func (in *Input) Filter(allowed func(rune) bool) *Input {
	in.filter = allowed
	return in
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
	in.Invalidate()
}

// BindValue vincula o conteúdo do campo ao State em DUAS vias: edições do
// usuário fazem Set no State, e um Set externo atualiza o campo (movendo o
// cursor para o fim). Encadeável.
func (in *Input) BindValue(s *state.State[string]) *Input {
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
func (in *Input) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.CharEvent:
		in.insert(e.Rune)
		return true
	case event.KeyEvent:
		return in.handleKey(e)
	case event.FocusEvent:
		in.focused = e.Gained
		if e.Gained {
			in.restartBlink()
			if in.onFocus != nil {
				in.onFocus()
			}
		} else {
			in.stopBlink()
			if in.onBlur != nil {
				in.onBlur()
			}
		}
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			// Posiciona o cursor no ponto clicado (via MeasureString, a
			// única fonte de verdade de largura) e inicia a seleção por
			// arraste — os event.MouseMove seguintes chegam pela captura do App.
			in.selecting = true
			in.moveCursor(in.runeIndexAt(e.Pos.X), false)
			return true
		case event.MouseMove:
			if !in.selecting {
				return false
			}
			return in.moveCursor(in.runeIndexAt(e.Pos.X), true)
		case event.MouseUp:
			in.selecting = false
			return false
		}
	}
	return false
}

// insert insere r na posição do cursor (substituindo a seleção, se houver) e
// avança o cursor.
func (in *Input) insert(r rune) {
	if in.filter != nil && !in.filter(r) {
		return
	}
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
func (in *Input) handleKey(e event.KeyEvent) bool {
	switch e.Key {
	case event.KeyEnter:
		if in.onSubmit == nil {
			return false
		}
		in.onSubmit()
		return true
	case event.KeyBackspace:
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
	case event.KeyDelete:
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
	case event.KeyLeft:
		// Sem Shift, uma seleção existente recolhe para a própria borda.
		if !e.Mods.Shift() && in.hasSelection() {
			start, _ := in.selection()
			return in.moveCursor(start, false)
		}
		return in.moveCursor(in.cursor-1, e.Mods.Shift())
	case event.KeyRight:
		if !e.Mods.Shift() && in.hasSelection() {
			_, end := in.selection()
			return in.moveCursor(end, false)
		}
		return in.moveCursor(in.cursor+1, e.Mods.Shift())
	case event.KeyHome:
		return in.moveCursor(0, e.Mods.Shift())
	case event.KeyEnd:
		return in.moveCursor(len(in.runes), e.Mods.Shift())
	case event.KeyA:
		if !e.Mods.Command() {
			return false
		}
		in.anchor, in.cursor = 0, len(in.runes)
		in.sync()
		return true
	case event.KeyC:
		if !e.Mods.Command() {
			return false
		}
		return in.copySelection()
	case event.KeyX:
		if !e.Mods.Command() || !in.copySelection() {
			return false
		}
		in.deleteSelection()
		in.sync()
		in.emitChange()
		return true
	case event.KeyV:
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
	hooks.WriteClipboard(string(in.runes[start:end]))
	return true
}

// paste cola o texto da área de transferência na posição do cursor,
// substituindo a seleção. Caracteres de controle (quebras de linha, tabs)
// são descartados: o campo é de linha única. Devolve true se algo mudou.
func (in *Input) paste() bool {
	var clean []rune
	for _, r := range hooks.ReadClipboard() {
		if r < 0x20 || r == 0x7F {
			continue
		}
		if in.filter != nil && !in.filter(r) {
			continue
		}
		clean = append(clean, r)
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

// restartBlink reinicia o ciclo de piscada com o cursor visível — chamado ao
// ganhar foco e a cada edição/movimento, para o cursor não sumir enquanto o
// usuário digita. Sem aplicação (testes) ou com Theme.CaretBlink zero, o
// cursor fica sempre visível.
func (in *Input) restartBlink() {
	in.stopBlink()
	in.caretOn = true
	if !in.focused || in.theme == nil || in.theme.CaretBlink <= 0 {
		return
	}
	in.blinkCancel = hooks.ScheduleAfter(in.theme.CaretBlink, in.blinkTick)
}

// blinkTick alterna a visibilidade do cursor e agenda a próxima piscada.
func (in *Input) blinkTick() {
	if !in.focused {
		return
	}
	in.caretOn = !in.caretOn
	in.Invalidate()
	in.blinkCancel = hooks.ScheduleAfter(in.theme.CaretBlink, in.blinkTick)
}

// stopBlink cancela a piscada pendente e deixa o cursor visível.
func (in *Input) stopBlink() {
	if in.blinkCancel != nil {
		in.blinkCancel()
		in.blinkCancel = nil
	}
	in.caretOn = true
}

// sync atualiza os caches derivados (string do texto e X do cursor) após
// qualquer mudança. Alocar aqui é aceitável: acontece por evento de edição
// (ou uma única vez após mudança de escala), nunca por frame desenhado.
func (in *Input) sync() {
	in.restartBlink()
	in.text = string(in.runes)
	if in.theme == nil {
		// Antes do mount não há como medir; Draw refaz o sync ao detectar a
		// primeira face real (nil → face do tema).
		in.cursorX, in.anchorX, in.syncFace = 0, 0, nil
		return
	}
	in.cursorX = in.theme.MeasureString(string(in.runes[:in.cursor]))
	in.anchorX = in.theme.MeasureString(string(in.runes[:in.anchor]))
	in.textW = in.theme.MeasureString(in.text)
	in.syncFace = in.theme.Face
}

// emitChange propaga uma edição para o State vinculado (se houver) e para o
// callback OnChange.
func (in *Input) emitChange() {
	if in.bound != nil && in.bound.Get() != in.text {
		in.bound.Set(in.text)
	}
	if in.onChange != nil {
		in.onChange(in.text)
	}
}

// runeIndexAt devolve o índice de cursor (em runes) mais próximo da
// coordenada X absoluta dada.
func (in *Input) runeIndexAt(x int) int {
	if in.theme == nil {
		return 0
	}
	rel := x - (in.Bounds().Min.X + in.theme.PaddingPx()) + in.scrollX
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
// placeholder, a seleção e o cursor quando focado. O conteúdo é RECORTADO à
// área útil do campo e rola horizontalmente para manter o cursor sempre
// visível quando o texto é maior que o campo.
func (in *Input) Draw(dst *image.RGBA) {
	if in.theme == nil {
		return
	}
	bounds := in.Bounds()
	th := in.theme

	// A face mudou desde o último sync (escala ou tema)? Recalcula uma vez.
	if in.syncFace != th.Face {
		in.sync()
	}

	render.FillRect(dst, bounds, th.InputBackground)
	border := th.InputBorder
	if in.focused {
		border = th.InputBorderFocused
	}
	render.StrokeRect(dst, bounds, th.BorderPx(), border)

	innerX0 := bounds.Min.X + th.PaddingPx()
	innerX1 := bounds.Max.X - th.PaddingPx()
	innerW := innerX1 - innerX0
	if innerW <= 0 {
		return
	}

	// Rolagem horizontal: limita ao intervalo válido e garante o cursor
	// visível (com espaço para a própria linha do cursor à direita).
	if in.textW <= innerW {
		in.scrollX = 0
	} else if in.scrollX > in.textW-innerW {
		in.scrollX = in.textW - innerW
	}
	if in.scrollX < 0 {
		in.scrollX = 0
	}
	if in.focused {
		if lim := innerW - th.BorderPx(); in.cursorX-in.scrollX > lim && lim > 0 {
			in.scrollX = in.cursorX - lim
		}
		if in.cursorX-in.scrollX < 0 {
			in.scrollX = in.cursorX
		}
	}

	textX := innerX0 - in.scrollX
	baseline := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()
	view := render.Clip(dst, image.Rect(innerX0, bounds.Min.Y, innerX1, bounds.Max.Y), &in.clip)

	if in.focused && in.hasSelection() {
		sx, ex := in.anchorX, in.cursorX
		if sx > ex {
			sx, ex = ex, sx
		}
		top := baseline - th.Ascent()
		render.FillRect(view, image.Rect(textX+sx, top, textX+ex, top+th.LineHeight()), th.Selection)
	}

	switch {
	case len(in.runes) > 0:
		th.DrawText(view, in.text, image.Pt(textX, baseline), th.Text)
	case !in.focused && in.Placeholder != "":
		th.DrawText(view, in.Placeholder, image.Pt(textX, baseline), th.Placeholder)
	}

	if in.focused && in.caretOn {
		top := baseline - th.Ascent()
		cx := textX + in.cursorX
		render.FillRect(view, image.Rect(cx, top, cx+th.BorderPx(), top+th.LineHeight()), th.Cursor)
	}
	in.drawDisabledOverlay(dst)
}
