package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Session é o NÚCLEO DE INTERAÇÃO de uma interface JUIGo, sem janela e sem
// GLFW: roteamento de mouse/teclado/rolagem, foco (clique e Tab), captura de
// mouse, hover (Enter/Leave, formato do cursor), pilha de overlays e
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
	// overlays é a PILHA de camadas de sobreposição (popups/modais),
	// da base ao topo: desenhadas na ordem, com o topo por cima e com
	// prioridade nos eventos. Cada camada guarda o foco a restaurar no seu
	// próprio fechamento — um popup aberto de dentro de um modal fecha
	// devolvendo o foco ao campo do modal, e o modal segue aberto.
	// overlayOpening marca um OpenOverlay em andamento: fechamentos
	// re-entrantes (camada que fecha ao perder o foco para a recém-aberta)
	// removem só a própria camada, sem derrubar a que está nascendo.
	overlays       []overlayEntry
	overlayOpening bool

	cursorShape CursorShape
	lastCursor  image.Point

	tipView   *TooltipView
	tipCancel func()
	tipShown  bool

	// toastView é a caixa do aviso transitório (ver toast.go) — camada
	// passiva como o tooltip, ancorada na base da janela.
	toastView   *TooltipView
	toastCancel func()
	toastShown  bool

	// Estado do arrasto (ver drag.go): fantasma seguindo o cursor, o alvo
	// realçado sob ele e o indicador de inserção corrente (DropIndicator).
	dragging      bool
	dragPayload   any
	dragView      *TooltipView
	dragTarget    Widget
	dragIndicator image.Rectangle

	// inspect liga a camada do inspector de depuração (ver inspector.go).
	inspect bool

	// damage acumula a REGIÃO suja do próximo frame; full força repintura
	// total (primeiro frame, resize, overlay, tema...). clipScratch é a
	// visão reutilizada do redesenho parcial.
	damage      image.Rectangle
	full        bool
	clipScratch image.RGBA
}

// overlayEntry é uma camada da pilha de sobreposição: o widget exibido e o
// foco a restaurar quando ELA fechar.
type overlayEntry struct {
	widget    Widget
	prevFocus Widget
}

// NewSession cria uma sessão com o tema dado.
func NewSession(th *theme.Theme) *Session {
	return &Session{theme: th, full: true}
}

// AddDamage acumula uma região suja para o próximo frame e o agenda. O dono
// da sessão liga este método em hooks.Damage.
func (s *Session) AddDamage(r image.Rectangle) {
	if !s.full {
		s.damage = s.damage.Union(r)
	}
	s.markDirty()
}

// InvalidateAll marca o frame inteiro como sujo e o agenda.
func (s *Session) InvalidateAll() {
	s.full = true
	s.markDirty()
}

// markDirty notifica o dono de que um redesenho é necessário.
func (s *Session) markDirty() {
	if s.OnDirty != nil {
		s.OnDirty()
	}
}

// mount injeta o tema E anexa esta sessão à árvore — a versão de Mount que
// a própria Session usa a cada renderização. O anexo dá identidade de
// janela ao dano: Invalidate/Layout dos widgets montados vão direto à
// sessão dona, sem passar pelo fallback global (que, com múltiplas
// janelas, reparte o dano entre todas).
func (s *Session) mount(w Widget, t *theme.Theme) {
	if h, ok := w.(sessionHost); ok {
		h.attachSession(s)
	}
	if h, ok := w.(themeHost); ok {
		t = h.inheritTheme(t)
	}
	if p, ok := w.(ParentWidget); ok {
		for _, ch := range p.Children() {
			s.mount(ch, t)
		}
	}
}

// SetRoot define a raiz da árvore e injeta o tema (mount).
func (s *Session) SetRoot(w Widget) {
	s.root = w
	if w != nil {
		s.mount(w, s.theme)
	}
	s.InvalidateAll()
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
	s.InvalidateAll()
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

// Overlay devolve a camada de sobreposição do TOPO da pilha (nil se
// nenhuma) — a que recebe os eventos.
func (s *Session) Overlay() Widget {
	return s.topOverlay()
}

// Overlays devolve as camadas de sobreposição abertas, da base ao topo (nil
// se nenhuma). A fatia é uma cópia recém-alocada — pensada para asserções e
// depuração, não para o caminho quente.
func (s *Session) Overlays() []Widget {
	if len(s.overlays) == 0 {
		return nil
	}
	out := make([]Widget, len(s.overlays))
	for i, e := range s.overlays {
		out[i] = e.widget
	}
	return out
}

// topOverlay devolve o widget do topo da pilha de overlays (nil se vazia).
func (s *Session) topOverlay() Widget {
	if n := len(s.overlays); n > 0 {
		return s.overlays[n-1].widget
	}
	return nil
}

// inOverlay informa se w pertence a alguma camada da pilha de overlays.
func (s *Session) inOverlay(w Widget) bool {
	for i := range s.overlays {
		if Contains(s.overlays[i].widget, w) {
			return true
		}
	}
	return false
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
// antigo fecham (em qualquer posição da pilha); overlays de janela inteira
// (Modal) apenas se reacomodam.
func (s *Session) Resize(size image.Point) {
	s.size = size
	for i := len(s.overlays) - 1; i >= 0; i-- {
		// O fechamento de uma camada pode fechar outras re-entrantemente
		// (restauração de foco); o índice pode ter ficado para trás.
		if i >= len(s.overlays) {
			continue
		}
		if !spansWindow(s.overlays[i].widget) {
			s.removeOverlayAt(i)
		}
	}
	s.hideTooltip()
	s.InvalidateAll()
}

// PointerDown processa o pressionar de um botão do mouse em pos: fecha o
// tooltip, aplica as regras de overlay (clique fora da camada do TOPO fecha
// só ela e é ENGOLIDO), move o foco para o focável sob o ponto e inicia a
// captura em quem consumir.
func (s *Session) PointerDown(pos image.Point, btn event.MouseButton) {
	s.lastCursor = pos
	s.hideTooltip()
	ev := event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: btn}
	if top := s.topOverlay(); top != nil {
		if !pos.In(top.Bounds()) {
			s.closeTopOverlay()
			return
		}
		if f := FocusableAt(top, pos); f != nil {
			s.setFocus(f)
		}
		if c := DispatchMouse(top, ev); c != nil {
			s.captured = c
			s.AddDamage(c.Bounds())
		}
		return
	}
	s.setFocus(FocusableAt(s.root, pos))
	s.captured = s.dispatch(ev)
}

// PointerUp processa o soltar de um botão: encerra a captura entregando o
// evento ao capturado, ou roteia por geometria (restrito à camada do topo,
// se houver overlay aberta).
func (s *Session) PointerUp(pos image.Point, btn event.MouseButton) {
	s.lastCursor = pos
	ev := event.MouseEvent{Kind: event.MouseUp, Pos: pos, Button: btn}
	if c := s.captured; c != nil {
		if c.HandleEvent(ev) {
			s.AddDamage(c.Bounds())
		}
		s.captured = nil
		s.finishDrag(pos) // solta no alvo sob o cursor, se houver arrasto
		return
	}
	if s.dragging {
		s.finishDrag(pos)
		return
	}
	if top := s.topOverlay(); top != nil {
		if pos.In(top.Bounds()) {
			if c := DispatchMouse(top, ev); c != nil {
				s.AddDamage(c.Bounds())
			}
		}
		return
	}
	s.dispatch(ev)
}

// PointerMove processa o movimento do ponteiro: durante a captura vai direto
// ao capturado (hover suspenso); com overlay, restrito à camada do topo;
// senão, rastreia hover e roteia por geometria.
func (s *Session) PointerMove(pos image.Point) {
	s.lastCursor = pos
	if s.inspect {
		// O crachá segue o ponteiro: redesenha tudo a cada movimento.
		s.InvalidateAll()
	}
	ev := event.MouseEvent{Kind: event.MouseMove, Pos: pos, Button: event.MouseButtonLeft}
	if c := s.captured; c != nil {
		if c.HandleEvent(ev) {
			s.AddDamage(c.Bounds())
		}
		// O arrasto pode ter começado NESTE movimento (limiar da fonte);
		// o fantasma e o alvo seguem o cursor a partir daqui.
		s.updateDrag(pos)
		return
	}
	if s.dragging {
		s.updateDrag(pos)
	}
	if top := s.topOverlay(); top != nil {
		if pos.In(top.Bounds()) {
			s.updateHoverIn(top, pos)
			if c := DispatchMouse(top, ev); c != nil {
				s.AddDamage(c.Bounds())
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
// camada do topo quando há overlay aberta); as demais vão ao widget focado;
// Escape não consumido com overlay aberta vai à camada do topo (fecha Modal
// mesmo com o foco em um campo interno).
func (s *Session) KeyPress(k event.Key, mods event.Modifiers) {
	if k == event.KeyUnknown {
		return
	}
	if s.dragging && k == event.KeyEscape {
		s.CancelDrag()
		return
	}
	if k == event.KeyTab {
		// Editores de código consomem o Tab LITERAL (ver ConsumesTab); para
		// eles o Tab não navega o foco — saia com o mouse ou Escape.
		if f := s.focused; f != nil && !DisabledOf(f) && consumesTab(f) {
			if f.HandleEvent(event.KeyEvent{Key: k, Mods: mods}) {
				s.AddDamage(f.Bounds())
			}
			return
		}
		if mods.Shift() {
			s.focusPrev()
		} else {
			s.focusNext()
		}
		return
	}
	if k == event.KeyI && mods.Command() {
		s.ToggleInspector()
		return
	}
	ke := event.KeyEvent{Key: k, Mods: mods}
	consumed := false
	// Referência local: o próprio handler pode fechar overlays ou mover o
	// foco (Escape num Modal o tira da pilha), e o dano é do widget original.
	if f := s.focused; f != nil && !DisabledOf(f) {
		consumed = f.HandleEvent(ke)
		if consumed {
			s.AddDamage(f.Bounds())
		}
	}
	if top := s.topOverlay(); !consumed && k == event.KeyEscape && top != nil {
		if top.HandleEvent(ke) {
			s.AddDamage(top.Bounds())
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
	if f := s.focused; f.HandleEvent(event.CharEvent{Rune: r}) {
		s.AddDamage(f.Bounds())
	}
}

// Scroll roteia a rolagem por geometria (restrita à camada do topo quando há
// overlay aberta; fora dela, fecha só o topo e engole).
func (s *Session) Scroll(pos image.Point, dx, dy float64) {
	if s.root == nil {
		return
	}
	s.lastCursor = pos
	s.hideTooltip()
	ev := event.ScrollEvent{Pos: pos, DX: dx, DY: dy}
	if top := s.topOverlay(); top != nil {
		if pos.In(top.Bounds()) {
			if c := DispatchScroll(top, ev); c != nil {
				s.AddDamage(c.Bounds())
			}
		} else {
			s.closeTopOverlay()
		}
		return
	}
	if c := DispatchScroll(s.root, ev); c != nil {
		s.AddDamage(c.Bounds())
	}
}

// OpenOverlay EMPILHA w como nova camada de sobreposição por cima das já
// abertas (um popup aberto de dentro de um modal fica sobre ele), guardando
// o foco atual — restaurado quando ESTA camada fechar — e focando o primeiro
// focável de w. O dono da sessão liga este método em hooks.OpenOverlay.
func (s *Session) OpenOverlay(w Widget) {
	if w == nil {
		return
	}
	s.hideTooltip()
	s.CancelDrag() // abrir uma camada no meio de um arrasto o cancela
	for i := range s.overlays {
		if s.overlays[i].widget == w {
			// Já empilhada: mantém a posição e só garante o redesenho
			// (Popup.ShowAt reposiciona por conta própria).
			s.InvalidateAll()
			return
		}
	}
	s.overlays = append(s.overlays, overlayEntry{widget: w, prevFocus: s.focused})
	// Do empilhamento até o foco inicial, fechamentos re-entrantes ficam em
	// modo camada única (ver CloseOverlayIf).
	prevOpening := s.overlayOpening
	s.overlayOpening = true
	s.mount(w, s.theme)
	// Layout imediato: a camada nasce com geometria válida, pronta para
	// eventos e foco antes mesmo do primeiro frame (como o Render faria).
	if spansWindow(w) {
		w.Layout(image.Rectangle{Max: s.size})
	} else {
		w.Layout(w.Bounds())
	}
	// Foca o primeiro focável DENTRO da camada (um campo de diálogo fica
	// pronto para digitar); sem descendentes focáveis, a própria camada.
	// Escape segue fechando modais pela entrega de fallback à overlay.
	var target Widget
	for _, f := range Focusables(w) {
		if f != w {
			target = f
			break
		}
	}
	if target == nil && w.Focusable() {
		target = w
	}
	if target != nil {
		s.setFocus(target)
	}
	s.overlayOpening = prevOpening
	s.InvalidateAll()
}

// CloseOverlayIf fecha a camada de w se ele estiver na pilha, junto com
// TODAS as camadas acima dela — abertas a partir do conteúdo dela, não fazem
// sentido órfãs (hooks.CloseOverlay).
func (s *Session) CloseOverlayIf(w Widget) {
	if w == nil {
		return
	}
	for i := len(s.overlays) - 1; i >= 0; i-- {
		if s.overlays[i].widget != w {
			continue
		}
		if s.overlayOpening {
			// Fechamento re-entrante no meio do empilhamento de OUTRA
			// camada (ex.: popup do Dropdown fechando ao perder o foco
			// para um modal recém-aberto por cima): remove só a camada
			// de w, preservando a que está nascendo acima.
			s.removeOverlayAt(i)
			return
		}
		// A restauração de foco de cada fechamento pode fechar outras
		// camadas re-entrantemente; o teto é reavaliado a cada volta.
		for len(s.overlays) > i {
			s.closeTopOverlay()
		}
		return
	}
}

// closeTopOverlay remove a camada do topo e restaura o foco dela.
func (s *Session) closeTopOverlay() {
	if n := len(s.overlays); n > 0 {
		s.removeOverlayAt(n - 1)
	}
}

// removeOverlayAt remove a camada i da pilha. No topo, restaura o foco
// guardado por ela; no meio (Resize fechando um popup sob um modal), o foco
// atual — que vive numa camada acima — fica intocado, e o foco guardado é
// repassado à camada de cima, para a cadeia de restauração não apontar para
// uma camada já fechada.
func (s *Session) removeOverlayAt(i int) {
	e := s.overlays[i]
	top := i == len(s.overlays)-1
	if !top {
		above := &s.overlays[i+1]
		if above.prevFocus != nil && Contains(e.widget, above.prevFocus) {
			above.prevFocus = e.prevFocus
		}
	}
	// Remove ANTES de mexer no foco: os FocusEvents disparados podem tentar
	// fechar esta mesma camada de novo (ex.: dropdownList ao perder o foco).
	n := len(s.overlays) - 1
	copy(s.overlays[i:], s.overlays[i+1:])
	s.overlays[n] = overlayEntry{}
	s.overlays = s.overlays[:n]
	if top {
		s.setFocus(e.prevFocus)
	}
	s.InvalidateAll()
}

// dispatch roteia um evento de mouse pela raiz e devolve o consumidor.
func (s *Session) dispatch(ev event.MouseEvent) Widget {
	if s.root == nil {
		return nil
	}
	consumer := DispatchMouse(s.root, ev)
	if consumer != nil {
		s.AddDamage(consumer.Bounds())
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
	if prev := s.hovered; prev != nil && prev.HandleEvent(event.MouseEvent{Kind: event.MouseLeave, Pos: pos}) {
		s.AddDamage(prev.Bounds())
	}
	s.hovered = target
	if target != nil && target.HandleEvent(event.MouseEvent{Kind: event.MouseEnter, Pos: pos}) {
		s.AddDamage(target.Bounds())
	}

	shape := CursorDefault
	if target != nil {
		shape = CursorShapeOf(target)
	}
	s.cursorShape = shape

	// Tooltip: cancela o do alvo anterior e agenda o do novo, se houver.
	s.hideTooltip()
	if target != nil && len(s.overlays) == 0 && s.theme != nil {
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
	s.mount(s.tipView, s.theme)
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
	s.AddDamage(s.tipView.Bounds())
}

// hideTooltip cancela a espera pendente e esconde a dica, se visível.
func (s *Session) hideTooltip() {
	if s.tipCancel != nil {
		s.tipCancel()
		s.tipCancel = nil
	}
	if s.tipShown {
		s.tipShown = false
		if s.tipView != nil {
			s.AddDamage(s.tipView.Bounds())
		} else {
			s.markDirty()
		}
	}
}

// Focus move o foco de teclado para w (nil limpa) — a versão pública do
// setFocus, para foco programático (hooks.Focus, widget.Focus).
func (s *Session) Focus(w Widget) {
	s.setFocus(w)
}

// setFocus move o foco para w (nil limpa), notificando com FocusEvent.
// Focar fora da camada do topo a fecha (Tab, clique em outro campo) —
// camada por camada, até a que contém w ou até a pilha esvaziar.
func (s *Session) setFocus(w Widget) {
	for w != nil {
		top := s.topOverlay()
		if top == nil || Contains(top, w) {
			break
		}
		s.closeTopOverlay()
	}
	if s.focused == w {
		return
	}
	if s.focused != nil {
		s.focused.HandleEvent(event.FocusEvent{Gained: false})
		s.AddDamage(s.focused.Bounds())
	}
	s.focused = w
	if w != nil {
		w.HandleEvent(event.FocusEvent{Gained: true})
		s.AddDamage(w.Bounds())
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

// focusRoot devolve a árvore onde o Tab circula: a camada do topo quando há
// overlay aberta.
func (s *Session) focusRoot() Widget {
	if top := s.topOverlay(); top != nil {
		return top
	}
	return s.root
}

// spansWindow informa se a overlay cobre a janela inteira.
func spansWindow(w Widget) bool {
	o, ok := w.(OverlaySpanning)
	return ok && o.SpansWindow()
}

// Render compõe o frame em dst e devolve a região efetivamente repintada e
// se a repintura foi total (o App usa isso para o upload parcial à GPU).
//
// dst deve ser PERSISTENTE entre frames: com dano parcial acumulado, apenas
// a região suja é repintada — fundo, árvore, overlay e tooltip são
// redesenhados dentro de uma visão recortada (render.Clip), então nada fora
// do dano é tocado e a ordem de camadas se mantém. O layout roda completo
// ANTES do recorte: o diff de bounds do BaseWidget acrescenta ao dano deste
// mesmo frame qualquer widget que o layout moveu. Não aloca com os caches
// aquecidos.
func (s *Session) Render(dst *image.RGBA) (image.Rectangle, bool) {
	if s.theme == nil {
		return image.Rectangle{}, false
	}
	// Layout completo primeiro: gera o dano de geometria do frame atual.
	if s.root != nil {
		// Re-injeta o tema antes do layout para cobrir widgets adicionados
		// dinamicamente à árvore (idempotente, sem alocações).
		s.mount(s.root, s.theme)
		s.root.Layout(dst.Bounds())
	}
	for i := range s.overlays {
		ov := s.overlays[i].widget
		s.mount(ov, s.theme)
		if spansWindow(ov) {
			ov.Layout(dst.Bounds())
		} else {
			ov.Layout(ov.Bounds())
		}
	}

	if s.toastShown && s.toastView != nil {
		s.mount(s.toastView, s.theme)
		s.layoutToast(dst.Bounds())
	}

	// O inspector desenha contornos da árvore inteira: sempre frame total.
	if s.inspect {
		s.full = true
	}

	full := s.full
	region := dst.Bounds()
	if !full {
		region = s.damage.Intersect(dst.Bounds())
	}
	s.damage = image.Rectangle{}
	s.full = false
	if region.Empty() {
		return image.Rectangle{}, false
	}

	target := dst
	if !full {
		target = render.Clip(dst, region, &s.clipScratch)
	}
	render.FillRect(target, region, s.theme.Background)
	if s.root != nil {
		s.root.Draw(target)
	}
	for i := range s.overlays {
		s.overlays[i].widget.Draw(target)
	}
	if s.tipShown && s.tipView != nil {
		s.tipView.Draw(target)
	}
	if s.toastShown && s.toastView != nil {
		s.toastView.Draw(target)
	}
	s.drawDrag(target)
	if s.inspect {
		s.drawInspector(target)
	}
	return region, full
}
