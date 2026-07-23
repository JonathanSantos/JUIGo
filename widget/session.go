package widget

import (
	"image"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/render"
	"juigo/theme"
)

// Session é o NÚCLEO DE INTERAÇÃO de uma interface JUIGo, sem janela e sem
// GLFW: roteamento de mouse/teclado/rolagem, foco (clique e Tab), captura de
// mouse, hover (Enter/Leave, formato do cursor), camada de overlay e
// tooltip, além da composição do frame (Render).
//
// O App real é uma casca fina que traduz eventos do GLFW para os métodos da
// Session; o juigo/uitest dirige a MESMA Session sinteticamente — o que
// garante que testar pelo harness é testar o comportamento real da
// aplicação. Todas as posições são em pixels do buffer.
//
// Como todo o JUIGo, a Session é single-threaded.
type Session struct {
	// OnDirty, se definido, é chamado sempre que algo visível muda e um
	// redesenho passa a ser necessário (o App o usa para marcar o frame).
	OnDirty func()

	theme   *theme.Theme
	size    image.Point
	root    Widget
	hovered Widget
	focused Widget
	// captured é o widget que consumiu o último PointerDown: até o botão
	// ser solto, recebe movimento e soltura DIRETAMENTE (captura de mouse),
	// mesmo fora dos próprios bounds — essencial para arrastar.
	captured Widget
	// overlay é a camada de sobreposição (popups/modais): desenhada por
	// cima e com prioridade nos eventos; overlayPrevFocus guarda o foco a
	// restaurar no fechamento.
	overlay          Widget
	overlayPrevFocus Widget

	cursorShape CursorShape
	lastCursor  image.Point

	tipView   *TooltipView
	tipCancel func()
	tipShown  bool
}

// NewSession cria uma sessão com o tema dado.
func NewSession(th *theme.Theme) *Session {
	return &Session{theme: th}
}

// markDirty notifica o dono de que um redesenho é necessário.
func (s *Session) markDirty() {
	if s.OnDirty != nil {
		s.OnDirty()
	}
}

// SetRoot define a raiz da árvore e injeta o tema (mount).
func (s *Session) SetRoot(w Widget) {
	s.root = w
	if w != nil {
		Mount(w, s.theme)
	}
	s.markDirty()
}

// Root devolve a raiz atual.
func (s *Session) Root() Widget {
	return s.root
}

// Theme devolve o tema da sessão.
func (s *Session) Theme() *theme.Theme {
	return s.theme
}

// SetTheme troca o tema da sessão em runtime: o próximo Render re-propaga o
// novo tema pela árvore (widgets com SetTheme explícito mantêm o próprio).
func (s *Session) SetTheme(th *theme.Theme) {
	if th == nil || th == s.theme {
		return
	}
	s.theme = th
	s.markDirty()
}

// Focused devolve o widget com o foco de teclado (nil se nenhum).
func (s *Session) Focused() Widget {
	return s.focused
}

// Hovered devolve o widget sob o ponteiro (nil se nenhum).
func (s *Session) Hovered() Widget {
	return s.hovered
}

// Captured devolve o widget com a captura de mouse ativa (nil se nenhuma).
func (s *Session) Captured() Widget {
	return s.captured
}

// Overlay devolve a camada de sobreposição atual (nil se nenhuma).
func (s *Session) Overlay() Widget {
	return s.overlay
}

// CursorShape devolve o formato de cursor desejado pelo widget sob o
// ponteiro (o App o traduz para o cursor do sistema).
func (s *Session) CursorShape() CursorShape {
	return s.cursorShape
}

// TooltipVisible informa se uma dica está visível.
func (s *Session) TooltipVisible() bool {
	return s.tipShown
}

// Resize registra o novo tamanho do buffer. Popups ancorados no layout
// antigo fecham; overlays de janela inteira (Modal) apenas se reacomodam.
func (s *Session) Resize(size image.Point) {
	s.size = size
	if s.overlay != nil && !spansWindow(s.overlay) {
		s.closeOverlay()
	}
	s.hideTooltip()
	s.markDirty()
}

// PointerDown processa o pressionar de um botão do mouse em pos: fecha o
// tooltip, aplica as regras de overlay (clique fora fecha e é ENGOLIDO),
// move o foco para o focável sob o ponto e inicia a captura em quem consumir.
func (s *Session) PointerDown(pos image.Point, btn event.MouseButton) {
	s.lastCursor = pos
	s.hideTooltip()
	ev := event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: btn}
	if s.overlay != nil {
		if !pos.In(s.overlay.Bounds()) {
			s.closeOverlay()
			return
		}
		if f := FocusableAt(s.overlay, pos); f != nil {
			s.setFocus(f)
		}
		if c := DispatchMouse(s.overlay, ev); c != nil {
			s.captured = c
			s.markDirty()
		}
		return
	}
	s.setFocus(FocusableAt(s.root, pos))
	s.captured = s.dispatch(ev)
}

// PointerUp processa o soltar de um botão: encerra a captura entregando o
// evento ao capturado, ou roteia por geometria (restrito à overlay, se
// aberta).
func (s *Session) PointerUp(pos image.Point, btn event.MouseButton) {
	s.lastCursor = pos
	ev := event.MouseEvent{Kind: event.MouseUp, Pos: pos, Button: btn}
	if s.captured != nil {
		if s.captured.HandleEvent(ev) {
			s.markDirty()
		}
		s.captured = nil
		return
	}
	if s.overlay != nil {
		if pos.In(s.overlay.Bounds()) {
			if DispatchMouse(s.overlay, ev) != nil {
				s.markDirty()
			}
		}
		return
	}
	s.dispatch(ev)
}

// PointerMove processa o movimento do ponteiro: durante a captura vai direto
// ao capturado (hover suspenso); com overlay, restrito a ela; senão, rastreia
// hover e roteia por geometria.
func (s *Session) PointerMove(pos image.Point) {
	s.lastCursor = pos
	ev := event.MouseEvent{Kind: event.MouseMove, Pos: pos, Button: event.MouseButtonLeft}
	if s.captured != nil {
		if s.captured.HandleEvent(ev) {
			s.markDirty()
		}
		return
	}
	if s.overlay != nil {
		if pos.In(s.overlay.Bounds()) {
			s.updateHoverIn(s.overlay, pos)
			if DispatchMouse(s.overlay, ev) != nil {
				s.markDirty()
			}
		} else {
			s.updateHoverIn(nil, pos)
		}
		return
	}
	s.updateHoverIn(s.root, pos)
	s.dispatch(ev)
}

// KeyPress processa uma tecla: Tab/Shift+Tab navegam o foco (restrito à
// overlay quando aberta); as demais vão ao widget focado; Escape não
// consumido com overlay aberta vai à própria overlay (fecha Modal mesmo com
// o foco em um campo interno).
func (s *Session) KeyPress(k event.Key, mods event.Modifiers) {
	if k == event.KeyUnknown {
		return
	}
	if k == event.KeyTab {
		if mods.Shift() {
			s.focusPrev()
		} else {
			s.focusNext()
		}
		return
	}
	ke := event.KeyEvent{Key: k, Mods: mods}
	consumed := false
	if s.focused != nil && !DisabledOf(s.focused) {
		consumed = s.focused.HandleEvent(ke)
		if consumed {
			s.markDirty()
		}
	}
	if !consumed && k == event.KeyEscape && s.overlay != nil {
		if s.overlay.HandleEvent(ke) {
			s.markDirty()
		}
	}
}

// Char entrega um caractere digitado ao widget focado.
func (s *Session) Char(r rune) {
	// Um widget pode ser desabilitado ENQUANTO focado (ex.: formulário que
	// invalida); a entrega de teclado respeita isso.
	if s.focused == nil || DisabledOf(s.focused) {
		return
	}
	if s.focused.HandleEvent(event.CharEvent{Rune: r}) {
		s.markDirty()
	}
}

// Scroll roteia a rolagem por geometria (restrita à overlay quando aberta;
// fora dela, fecha e engole).
func (s *Session) Scroll(pos image.Point, dx, dy float64) {
	if s.root == nil {
		return
	}
	s.lastCursor = pos
	s.hideTooltip()
	ev := event.ScrollEvent{Pos: pos, DX: dx, DY: dy}
	if s.overlay != nil {
		if pos.In(s.overlay.Bounds()) {
			if DispatchScroll(s.overlay, ev) != nil {
				s.markDirty()
			}
		} else {
			s.closeOverlay()
		}
		return
	}
	if DispatchScroll(s.root, ev) != nil {
		s.markDirty()
	}
}

// OpenOverlay exibe w como camada de sobreposição, guardando o foco atual e
// focando w se for focável. O dono da sessão liga este método em
// hooks.OpenOverlay.
func (s *Session) OpenOverlay(w Widget) {
	if w == nil {
		return
	}
	s.hideTooltip()
	if s.overlay == nil {
		s.overlayPrevFocus = s.focused
	}
	s.overlay = w
	Mount(w, s.theme)
	if w.Focusable() {
		s.setFocus(w)
	}
	s.markDirty()
}

// CloseOverlayIf fecha a camada se w for a camada atual (hooks.CloseOverlay).
func (s *Session) CloseOverlayIf(w Widget) {
	if w != nil && w == s.overlay {
		s.closeOverlay()
	}
}

// closeOverlay remove a camada e restaura o foco anterior.
func (s *Session) closeOverlay() {
	if s.overlay == nil {
		return
	}
	s.overlay = nil
	s.setFocus(s.overlayPrevFocus)
	s.overlayPrevFocus = nil
	s.markDirty()
}

// dispatch roteia um evento de mouse pela raiz e devolve o consumidor.
func (s *Session) dispatch(ev event.MouseEvent) Widget {
	if s.root == nil {
		return nil
	}
	consumer := DispatchMouse(s.root, ev)
	if consumer != nil {
		s.markDirty()
	}
	return consumer
}

// updateHoverIn rastreia o hover contra a árvore dada (nil limpa), entrega
// MouseLeave/MouseEnter, atualiza o formato do cursor e agenda o tooltip.
func (s *Session) updateHoverIn(root Widget, pos image.Point) {
	var target Widget
	if root != nil {
		target = DeepestAt(root, pos)
	}
	if target == s.hovered {
		return
	}
	if s.hovered != nil && s.hovered.HandleEvent(event.MouseEvent{Kind: event.MouseLeave, Pos: pos}) {
		s.markDirty()
	}
	s.hovered = target
	if target != nil && target.HandleEvent(event.MouseEvent{Kind: event.MouseEnter, Pos: pos}) {
		s.markDirty()
	}

	shape := CursorDefault
	if target != nil {
		shape = CursorShapeOf(target)
	}
	s.cursorShape = shape

	// Tooltip: cancela o do alvo anterior e agenda o do novo, se houver.
	s.hideTooltip()
	if target != nil && s.overlay == nil && s.theme != nil {
		if text := TooltipTextOf(target); text != "" {
			s.tipCancel = hooks.ScheduleAfter(s.theme.TooltipDelay, func() {
				s.tipCancel = nil
				s.showTooltip(text)
			})
		}
	}
}

// showTooltip exibe a dica próxima ao último ponto do cursor, limitada ao
// tamanho da sessão.
func (s *Session) showTooltip(text string) {
	if s.tipView == nil {
		s.tipView = NewTooltipView()
	}
	Mount(s.tipView, s.theme)
	s.tipView.SetText(text)
	pref := s.tipView.PreferredSize()
	off := s.theme.PaddingPx()
	pos := s.lastCursor.Add(image.Pt(off, off*2))
	if pos.X+pref.X > s.size.X {
		pos.X = s.size.X - pref.X
	}
	if pos.Y+pref.Y > s.size.Y {
		pos.Y = s.lastCursor.Y - off - pref.Y
	}
	if pos.X < 0 {
		pos.X = 0
	}
	if pos.Y < 0 {
		pos.Y = 0
	}
	s.tipView.Layout(image.Rectangle{Min: pos, Max: pos.Add(pref)})
	s.tipShown = true
	s.markDirty()
}

// hideTooltip cancela a espera pendente e esconde a dica, se visível.
func (s *Session) hideTooltip() {
	if s.tipCancel != nil {
		s.tipCancel()
		s.tipCancel = nil
	}
	if s.tipShown {
		s.tipShown = false
		s.markDirty()
	}
}

// setFocus move o foco para w (nil limpa), notificando com FocusEvent.
// Focar fora da overlay aberta a fecha (Tab, clique em outro campo).
func (s *Session) setFocus(w Widget) {
	if s.overlay != nil && w != nil && !Contains(s.overlay, w) {
		s.closeOverlay()
	}
	if s.focused == w {
		return
	}
	if s.focused != nil {
		s.focused.HandleEvent(event.FocusEvent{Gained: false})
	}
	s.focused = w
	if w != nil {
		w.HandleEvent(event.FocusEvent{Gained: true})
	}
	s.markDirty()
}

// focusNext avança o foco na ordem da árvore, com wraparound.
func (s *Session) focusNext() {
	order := Focusables(s.focusRoot())
	if len(order) == 0 {
		return
	}
	next := 0
	for i, w := range order {
		if w == s.focused {
			next = (i + 1) % len(order)
			break
		}
	}
	s.setFocus(order[next])
}

// focusPrev recua o foco na ordem da árvore (Shift+Tab), com wraparound.
func (s *Session) focusPrev() {
	order := Focusables(s.focusRoot())
	if len(order) == 0 {
		return
	}
	prev := len(order) - 1
	for i, w := range order {
		if w == s.focused {
			prev = (i - 1 + len(order)) % len(order)
			break
		}
	}
	s.setFocus(order[prev])
}

// focusRoot devolve a árvore onde o Tab circula: a overlay quando aberta.
func (s *Session) focusRoot() Widget {
	if s.overlay != nil {
		return s.overlay
	}
	return s.root
}

// spansWindow informa se a overlay cobre a janela inteira.
func spansWindow(w Widget) bool {
	o, ok := w.(OverlaySpanning)
	return ok && o.SpansWindow()
}

// Render compõe o frame em dst: fundo do tema, árvore, overlay e tooltip.
// Não aloca com os caches aquecidos.
func (s *Session) Render(dst *image.RGBA) {
	if s.theme == nil {
		return
	}
	render.FillRect(dst, dst.Bounds(), s.theme.Background)
	if s.root != nil {
		// Re-injeta o tema antes do layout para cobrir widgets adicionados
		// dinamicamente à árvore (idempotente, sem alocações).
		Mount(s.root, s.theme)
		s.root.Layout(dst.Bounds())
		s.root.Draw(dst)
	}
	if s.overlay != nil {
		Mount(s.overlay, s.theme)
		if spansWindow(s.overlay) {
			s.overlay.Layout(dst.Bounds())
		} else {
			s.overlay.Layout(s.overlay.Bounds())
		}
		s.overlay.Draw(dst)
	}
	if s.tipShown && s.tipView != nil {
		s.tipView.Draw(dst)
	}
}
