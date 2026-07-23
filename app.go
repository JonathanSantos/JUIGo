package juigo

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/render"
	"juigo/theme"
	"juigo/widget"
)

func init() {
	// GLFW e OpenGL exigem que tudo aconteça na main thread do SO.
	runtime.LockOSThread()
}

// App é a aplicação JUIGo: dona da janela, do buffer RGBA, da textura GL
// (via render.Blitter) e do loop de eventos. Crie com New e inicie com Run.
//
// O buffer é alocado no tamanho do FRAMEBUFFER da janela (pixels físicos):
// em telas HiDPI ele é maior que o tamanho lógico, o tema é escalado pela
// escala de conteúdo do monitor e as coordenadas de mouse são convertidas de
// lógicas para pixels antes do roteamento — widgets só veem pixels.
type App struct {
	window  *glfw.Window
	blitter *render.Blitter
	buf     *image.RGBA
	theme   *theme.Theme
	bus     *event.Bus
	root    widget.Widget
	hovered widget.Widget
	focused widget.Widget
	// captured é o widget que consumiu o último MouseDown: até o botão ser
	// solto, ele recebe MouseMove/MouseUp DIRETAMENTE (captura de mouse),
	// mesmo com o cursor fora dos seus bounds — essencial para arrastar
	// (Slider, seleção de texto).
	captured widget.Widget
	width    int // dimensões do buffer, em pixels físicos
	height   int
	// pixelRatio converte coordenadas lógicas da janela (mouse) em pixels
	// do framebuffer. 2 em telas retina; 1 em telas comuns.
	pixelRatio float64
	dirty      bool
	// fatalErr registra uma falha ocorrida dentro de um callback (que não
	// pode devolver erro); Run a detecta e encerra.
	fatalErr error
	// cursorShape é o formato de cursor aplicado; stdCursors cacheia os
	// cursores padrão do GLFW já criados.
	cursorShape widget.CursorShape
	stdCursors  map[widget.CursorShape]*glfw.Cursor
	// timers são os agendamentos pendentes (hooks.Schedule): o loop usa
	// WaitEventsTimeout para acordar no vencimento mais próximo.
	timers   []appTimer
	timerSeq int
	// overlay é a camada de sobreposição (popups): desenhada por cima e com
	// prioridade nos eventos; clique fora a fecha. overlayPrevFocus guarda o
	// foco a restaurar no fechamento.
	overlay          widget.Widget
	overlayPrevFocus widget.Widget
	// Estado do tooltip: a caixa (lazy), o cancelamento do timer de espera,
	// se está visível e a última posição do cursor em pixels do buffer.
	tipView    *widget.TooltipView
	tipCancel  func()
	tipShown   bool
	lastCursor image.Point
}

// appTimer é um agendamento pendente de hooks.Schedule.
type appTimer struct {
	id int
	at time.Time
	fn func()
}

// New cria a janela com o título e tamanho dados e inicializa o contexto
// OpenGL. Deve ser chamada na main thread (o pacote já trava a goroutine
// corrente na thread do SO via runtime.LockOSThread).
func New(title string, width, height int) (*App, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("juigo: falha ao inicializar GLFW: %w", err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True) // exigido no macOS
	glfw.WindowHint(glfw.Resizable, glfw.True)

	window, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("juigo: falha ao criar janela: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	// O buffer vive em pixels físicos do framebuffer (HiDPI incluso).
	fbw, fbh := window.GetFramebufferSize()
	if fbw <= 0 || fbh <= 0 {
		fbw, fbh = width, height
	}

	blitter, err := render.NewBlitter(fbw, fbh)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, err
	}

	th, err := theme.Default()
	if err != nil {
		blitter.Destroy()
		window.Destroy()
		glfw.Terminate()
		return nil, err
	}

	// Escala o tema (fonte, métricas) para o monitor onde a janela abriu.
	scaleX, _ := window.GetContentScale()
	if scaleX > 0 && scaleX != 1 {
		if err := th.SetScale(float64(scaleX)); err != nil {
			blitter.Destroy()
			window.Destroy()
			glfw.Terminate()
			return nil, err
		}
	}

	a := &App{
		window:     window,
		blitter:    blitter,
		theme:      th,
		bus:        event.NewBus(),
		buf:        image.NewRGBA(image.Rect(0, 0, fbw, fbh)),
		width:      fbw,
		height:     fbh,
		pixelRatio: pixelRatio(window, fbw),
		dirty:      true,
	}
	a.installCallbacks()
	// Setters de widgets e State.Set redesenham através deste hook; o Input
	// copia/cola através da área de transferência do sistema.
	hooks.Repaint = a.Invalidate
	hooks.ClipboardRead = window.GetClipboardString
	hooks.ClipboardWrite = window.SetClipboardString
	hooks.Schedule = a.schedule
	hooks.OpenOverlay = a.openOverlay
	hooks.CloseOverlay = a.closeOverlayIf
	return a, nil
}

// schedule agenda fn para executar na main thread após d e devolve o
// cancelamento. Implementação de hooks.Schedule.
func (a *App) schedule(d time.Duration, fn func()) func() {
	a.timerSeq++
	id := a.timerSeq
	a.timers = append(a.timers, appTimer{id: id, at: time.Now().Add(d), fn: fn})
	glfw.PostEmptyEvent() // acorda o loop para recalcular o timeout
	return func() {
		for i := range a.timers {
			if a.timers[i].id == id {
				a.timers = append(a.timers[:i], a.timers[i+1:]...)
				return
			}
		}
	}
}

// nextTimerWait devolve quanto falta para o agendamento mais próximo.
func (a *App) nextTimerWait() (time.Duration, bool) {
	if len(a.timers) == 0 {
		return 0, false
	}
	next := a.timers[0].at
	for _, t := range a.timers[1:] {
		if t.at.Before(next) {
			next = t.at
		}
	}
	return time.Until(next), true
}

// runDueTimers executa os agendamentos vencidos. Os callbacks rodam após a
// remoção da lista e podem agendar novos timers com segurança.
func (a *App) runDueTimers() {
	now := time.Now()
	var due []func()
	kept := a.timers[:0]
	for _, t := range a.timers {
		if t.at.After(now) {
			kept = append(kept, t)
		} else {
			due = append(due, t.fn)
		}
	}
	a.timers = kept
	for _, fn := range due {
		fn()
	}
}

// openOverlay exibe w como camada de sobreposição (hooks.OpenOverlay).
func (a *App) openOverlay(v any) {
	w, ok := v.(widget.Widget)
	if !ok || w == nil {
		return
	}
	a.hideTooltip()
	if a.overlay == nil {
		a.overlayPrevFocus = a.focused
	}
	a.overlay = w
	widget.Mount(w, a.theme)
	if w.Focusable() {
		a.setFocus(w)
	}
	a.dirty = true
}

// closeOverlayIf fecha a camada se v for a camada atual (hooks.CloseOverlay).
func (a *App) closeOverlayIf(v any) {
	if w, ok := v.(widget.Widget); ok && w == a.overlay {
		a.closeOverlay()
	}
}

// closeOverlay remove a camada de sobreposição e restaura o foco anterior.
func (a *App) closeOverlay() {
	if a.overlay == nil {
		return
	}
	a.overlay = nil
	a.setFocus(a.overlayPrevFocus)
	a.overlayPrevFocus = nil
	a.dirty = true
}

// showTooltip exibe a caixa de dica próxima ao cursor, limitada à janela.
func (a *App) showTooltip(text string) {
	if a.tipView == nil {
		a.tipView = widget.NewTooltipView()
	}
	widget.Mount(a.tipView, a.theme)
	a.tipView.SetText(text)
	pref := a.tipView.PreferredSize()
	off := a.theme.PaddingPx()
	pos := a.lastCursor.Add(image.Pt(off, off*2))
	if pos.X+pref.X > a.width {
		pos.X = a.width - pref.X
	}
	if pos.Y+pref.Y > a.height {
		pos.Y = a.lastCursor.Y - off - pref.Y
	}
	if pos.X < 0 {
		pos.X = 0
	}
	if pos.Y < 0 {
		pos.Y = 0
	}
	a.tipView.Layout(image.Rectangle{Min: pos, Max: pos.Add(pref)})
	a.tipShown = true
	a.dirty = true
}

// hideTooltip cancela a espera pendente e esconde a dica, se visível.
func (a *App) hideTooltip() {
	if a.tipCancel != nil {
		a.tipCancel()
		a.tipCancel = nil
	}
	if a.tipShown {
		a.tipShown = false
		a.dirty = true
	}
}

// Run cria a aplicação com New, define root como raiz da árvore e executa o
// loop de eventos até a janela ser fechada. É o caminho curto para a maioria
// das aplicações; use New quando precisar do *App (tema, Invalidate, Bus).
func Run(title string, width, height int, root widget.Widget) error {
	app, err := New(title, width, height)
	if err != nil {
		return err
	}
	app.SetRoot(root)
	return app.Run()
}

// pixelRatio calcula a razão framebuffer/janela (pixels físicos por unidade
// lógica de janela), usada para converter coordenadas de mouse.
func pixelRatio(window *glfw.Window, fbWidth int) float64 {
	winW, _ := window.GetSize()
	if winW <= 0 || fbWidth <= 0 {
		return 1
	}
	return float64(fbWidth) / float64(winW)
}

// installCallbacks registra os callbacks de janela do GLFW. Os callbacks
// mutam o estado do App diretamente — sem mutex e sem goroutines, pois tudo
// executa na main thread dentro de glfw.WaitEvents.
func (a *App) installCallbacks() {
	a.window.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		a.resize(w, h)
	})
	a.window.SetContentScaleCallback(func(_ *glfw.Window, scaleX, _ float32) {
		// Janela mudou de monitor: refaz fonte, métricas e cache de glyphs.
		if scaleX <= 0 {
			return
		}
		if err := a.theme.SetScale(float64(scaleX)); err != nil {
			a.fatalErr = err
			return
		}
		a.dirty = true
	})
	a.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		x, y := a.window.GetCursorPos()
		pos := a.toBufferCoords(x, y)
		ev := event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: mapMouseButton(button)}
		if action == glfw.Release {
			ev.Kind = event.MouseUp
			// Fim da captura: o widget capturado recebe o MouseUp mesmo
			// que o cursor esteja fora dele.
			if a.captured != nil {
				if a.captured.HandleEvent(ev) {
					a.dirty = true
				}
				a.captured = nil
				return
			}
			if a.overlay != nil {
				if pos.In(a.overlay.Bounds()) {
					if widget.DispatchMouse(a.overlay, ev) != nil {
						a.dirty = true
					}
				}
				return
			}
			a.dispatch(ev)
			return
		}
		a.hideTooltip()
		// Com overlay aberta, ela tem prioridade: clique dentro roteia nela;
		// clique fora a fecha e é ENGOLIDO (não ativa o que está por baixo).
		if a.overlay != nil {
			if !pos.In(a.overlay.Bounds()) {
				a.closeOverlay()
				return
			}
			if f := widget.FocusableAt(a.overlay, pos); f != nil {
				a.setFocus(f)
			}
			if c := widget.DispatchMouse(a.overlay, ev); c != nil {
				a.captured = c
				a.dirty = true
			}
			return
		}
		// Clique em widget focável muda o foco; em área sem widget focável,
		// limpa o foco. Quem consumir o MouseDown captura o mouse.
		a.setFocus(widget.FocusableAt(a.root, pos))
		a.captured = a.dispatch(ev)
	})
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, rawMods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		mods := mapMods(rawMods)
		if key == glfw.KeyTab {
			if mods.Shift() {
				a.focusPrev()
			} else {
				a.focusNext()
			}
			return
		}
		k := mapKey(key)
		if k == event.KeyUnknown || a.focused == nil {
			return
		}
		// Teclado roteia por FOCO: direto ao widget focado, sem hit-test.
		if a.focused.HandleEvent(event.KeyEvent{Key: k, Mods: mods}) {
			a.dirty = true
		}
	})
	a.window.SetCharCallback(func(_ *glfw.Window, r rune) {
		if a.focused == nil {
			return
		}
		if a.focused.HandleEvent(event.CharEvent{Rune: r}) {
			a.dirty = true
		}
	})
	a.window.SetScrollCallback(func(_ *glfw.Window, dx, dy float64) {
		if a.root == nil {
			return
		}
		x, y := a.window.GetCursorPos()
		pos := a.toBufferCoords(x, y)
		a.hideTooltip()
		ev := event.ScrollEvent{Pos: pos, DX: dx, DY: dy}
		if a.overlay != nil {
			// Dentro da overlay, roteia nela; fora, fecha e engole.
			if pos.In(a.overlay.Bounds()) {
				if widget.DispatchScroll(a.overlay, ev) != nil {
					a.dirty = true
				}
			} else {
				a.closeOverlay()
			}
			return
		}
		// Rolagem roteia por GEOMETRIA, como o mouse: vai ao widget mais
		// profundo sob o cursor e propaga para cima se não consumida.
		if widget.DispatchScroll(a.root, ev) != nil {
			a.dirty = true
		}
	})
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		pos := a.toBufferCoords(x, y)
		a.lastCursor = pos
		ev := event.MouseEvent{Kind: event.MouseMove, Pos: pos, Button: event.MouseButtonLeft}
		if a.captured != nil {
			// Durante a captura, os movimentos vão direto ao capturado e o
			// hover fica suspenso (nada de realçar outros widgets no meio
			// de um arraste).
			if a.captured.HandleEvent(ev) {
				a.dirty = true
			}
			return
		}
		if a.overlay != nil {
			// Hover e movimento restritos à overlay; fora dela, nada realça.
			if pos.In(a.overlay.Bounds()) {
				a.updateHoverIn(a.overlay, pos)
				if widget.DispatchMouse(a.overlay, ev) != nil {
					a.dirty = true
				}
			} else {
				a.updateHoverIn(nil, pos)
			}
			return
		}
		a.updateHover(pos)
		a.dispatch(ev)
	})
	// Refresh dispara quando o SO precisa que a janela seja repintada
	// (ex.: durante o redimensionamento, que roda um loop modal no macOS).
	a.window.SetRefreshCallback(func(_ *glfw.Window) {
		a.dirty = true
		a.render()
	})
}

// toBufferCoords converte coordenadas lógicas de janela (como o GLFW entrega
// a posição do cursor) em pixels do buffer/framebuffer.
func (a *App) toBufferCoords(x, y float64) image.Point {
	return image.Pt(
		int(math.Round(x*a.pixelRatio)),
		int(math.Round(y*a.pixelRatio)),
	)
}

// dispatch roteia um evento de mouse pela árvore (por geometria), marca a
// interface como suja se alguém o consumir e devolve o widget consumidor.
func (a *App) dispatch(ev event.MouseEvent) widget.Widget {
	if a.root == nil {
		return nil
	}
	consumer := widget.DispatchMouse(a.root, ev)
	if consumer != nil {
		a.dirty = true
	}
	return consumer
}

// updateHover mantém o widget sob o cursor, entregando event.MouseLeave ao widget
// anterior e event.MouseEnter ao novo quando o alvo muda.
func (a *App) updateHover(pos image.Point) {
	a.updateHoverIn(a.root, pos)
}

// updateHoverIn rastreia o hover contra a árvore dada (a raiz normal ou a
// overlay; nil limpa o hover), entregando MouseLeave/MouseEnter, aplicando o
// cursor e agendando o tooltip do novo alvo.
func (a *App) updateHoverIn(root widget.Widget, pos image.Point) {
	var target widget.Widget
	if root != nil {
		target = widget.DeepestAt(root, pos)
	}
	if target == a.hovered {
		return
	}
	if a.hovered != nil && a.hovered.HandleEvent(event.MouseEvent{Kind: event.MouseLeave, Pos: pos}) {
		a.dirty = true
	}
	a.hovered = target
	if target != nil && target.HandleEvent(event.MouseEvent{Kind: event.MouseEnter, Pos: pos}) {
		a.dirty = true
	}

	// Aplica o formato de cursor desejado pelo widget sob o ponteiro
	// (I-beam em campos de texto, mãozinha em clicáveis).
	shape := widget.CursorDefault
	if target != nil {
		shape = widget.CursorShapeOf(target)
	}
	a.applyCursor(shape)

	// Tooltip: cancela o do alvo anterior e, se o novo tem dica, agenda a
	// exibição após a pausa do tema.
	a.hideTooltip()
	if target != nil && a.overlay == nil {
		if text := widget.TooltipTextOf(target); text != "" {
			a.tipCancel = a.schedule(a.theme.TooltipDelay, func() {
				a.tipCancel = nil
				a.showTooltip(text)
			})
		}
	}
}

// applyCursor troca o cursor da janela para o formato dado, criando (e
// cacheando) os cursores padrão do GLFW sob demanda.
func (a *App) applyCursor(shape widget.CursorShape) {
	if shape == a.cursorShape {
		return
	}
	a.cursorShape = shape
	if shape == widget.CursorDefault {
		a.window.SetCursor(nil)
		return
	}
	if a.stdCursors == nil {
		a.stdCursors = make(map[widget.CursorShape]*glfw.Cursor)
	}
	cur, ok := a.stdCursors[shape]
	if !ok {
		switch shape {
		case widget.CursorText:
			cur = glfw.CreateStandardCursor(glfw.IBeamCursor)
		case widget.CursorHand:
			cur = glfw.CreateStandardCursor(glfw.HandCursor)
		}
		a.stdCursors[shape] = cur
	}
	a.window.SetCursor(cur)
}

// setFocus move o foco de teclado para w (ou o limpa, se w for nil),
// notificando os widgets envolvidos com event.FocusEvent. Focar um widget
// fora da overlay aberta a fecha (Tab, clique em outro campo).
func (a *App) setFocus(w widget.Widget) {
	if a.overlay != nil && w != nil && !widget.Contains(a.overlay, w) {
		a.closeOverlay()
	}
	if a.focused == w {
		return
	}
	if a.focused != nil {
		a.focused.HandleEvent(event.FocusEvent{Gained: false})
	}
	a.focused = w
	if w != nil {
		w.HandleEvent(event.FocusEvent{Gained: true})
	}
	a.dirty = true
}

// focusNext avança o foco para o próximo widget focável na ordem da árvore,
// com wraparound. Sem widgets focáveis, não faz nada.
func (a *App) focusNext() {
	order := widget.Focusables(a.root)
	if len(order) == 0 {
		return
	}
	next := 0
	for i, w := range order {
		if w == a.focused {
			next = (i + 1) % len(order)
			break
		}
	}
	a.setFocus(order[next])
}

// focusPrev recua o foco para o widget focável anterior na ordem da árvore
// (Shift+Tab), com wraparound.
func (a *App) focusPrev() {
	order := widget.Focusables(a.root)
	if len(order) == 0 {
		return
	}
	prev := len(order) - 1
	for i, w := range order {
		if w == a.focused {
			prev = (i - 1 + len(order)) % len(order)
			break
		}
	}
	a.setFocus(order[prev])
}

// mapMods converte os modificadores do GLFW para o tipo do JUIGo.
func mapMods(m glfw.ModifierKey) event.Modifiers {
	var mods event.Modifiers
	if m&glfw.ModShift != 0 {
		mods |= event.ModShift
	}
	if m&glfw.ModControl != 0 {
		mods |= event.ModControl
	}
	if m&glfw.ModAlt != 0 {
		mods |= event.ModAlt
	}
	if m&glfw.ModSuper != 0 {
		mods |= event.ModSuper
	}
	return mods
}

// mapKey converte teclas do GLFW para as teclas reconhecidas pelo JUIGo.
func mapKey(key glfw.Key) event.Key {
	switch key {
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return event.KeyEnter
	case glfw.KeySpace:
		return event.KeySpace
	case glfw.KeyBackspace:
		return event.KeyBackspace
	case glfw.KeyDelete:
		return event.KeyDelete
	case glfw.KeyLeft:
		return event.KeyLeft
	case glfw.KeyRight:
		return event.KeyRight
	case glfw.KeyHome:
		return event.KeyHome
	case glfw.KeyEnd:
		return event.KeyEnd
	case glfw.KeyUp:
		return event.KeyUp
	case glfw.KeyDown:
		return event.KeyDown
	case glfw.KeyEscape:
		return event.KeyEscape
	case glfw.KeyA:
		return event.KeyA
	case glfw.KeyC:
		return event.KeyC
	case glfw.KeyV:
		return event.KeyV
	case glfw.KeyX:
		return event.KeyX
	default:
		return event.KeyUnknown
	}
}

// mapMouseButton converte o botão do GLFW para o tipo do JUIGo.
func mapMouseButton(b glfw.MouseButton) event.MouseButton {
	switch b {
	case glfw.MouseButtonRight:
		return event.MouseButtonRight
	case glfw.MouseButtonMiddle:
		return event.MouseButtonMiddle
	default:
		return event.MouseButtonLeft
	}
}

// resize realoca o buffer RGBA para o novo tamanho do framebuffer, em pixels
// físicos. Esta é a única situação em que o buffer é realocado.
func (a *App) resize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	a.width, a.height = w, h
	a.buf = image.NewRGBA(image.Rect(0, 0, w, h))
	a.pixelRatio = pixelRatio(a.window, w)
	// Popups e tooltips têm posição absoluta ancorada no layout antigo.
	a.closeOverlay()
	a.hideTooltip()
	a.dirty = true
}

// Theme devolve o tema da aplicação, usado na construção dos widgets.
// Bus devolve o barramento de eventos da aplicação (Publish síncrono), para
// comunicação entre partes do código do usuário.
func (a *App) Bus() *event.Bus {
	return a.bus
}

func (a *App) Theme() *theme.Theme {
	return a.theme
}

// SetRoot define o widget raiz da árvore de interface, injeta o tema da
// aplicação na árvore (mount) e agenda um redesenho. A raiz recebe Layout
// com os limites do buffer a cada renderização.
func (a *App) SetRoot(w widget.Widget) {
	a.root = w
	if w != nil {
		widget.Mount(w, a.theme)
	}
	a.dirty = true
}

// Invalidate marca a interface como suja e acorda o loop de eventos, forçando
// uma nova renderização. Útil para mudanças de estado feitas fora do fluxo de
// eventos da janela.
func (a *App) Invalidate() {
	a.dirty = true
	glfw.PostEmptyEvent()
}

// render redesenha o frame no buffer RGBA, envia para a textura e apresenta.
// Não aloca: o buffer é reutilizado entre frames.
func (a *App) render() {
	if !a.dirty || a.width <= 0 || a.height <= 0 {
		return
	}
	render.FillRect(a.buf, a.buf.Bounds(), a.theme.Background)
	if a.root != nil {
		// Re-injeta o tema antes do layout para cobrir widgets adicionados
		// dinamicamente à árvore (idempotente, sem alocações).
		widget.Mount(a.root, a.theme)
		a.root.Layout(a.buf.Bounds())
		a.root.Draw(a.buf)
	}
	// Camadas superiores: overlay (popups) e, por cima, o tooltip.
	if a.overlay != nil {
		widget.Mount(a.overlay, a.theme)
		a.overlay.Layout(a.overlay.Bounds())
		a.overlay.Draw(a.buf)
	}
	if a.tipShown && a.tipView != nil {
		a.tipView.Draw(a.buf)
	}

	a.blitter.Upload(a.buf)
	fbw, fbh := a.window.GetFramebufferSize()
	a.blitter.Draw(fbw, fbh)
	a.window.SwapBuffers()
	a.dirty = false
}

// Run executa o loop de eventos até a janela ser fechada e então libera os
// recursos da aplicação. Usa glfw.WaitEvents (bloqueante): só há trabalho de
// CPU quando chegam eventos, e só há renderização quando o estado está sujo.
// Falhas ocorridas dentro de callbacks (ex.: reescalar o tema ao trocar de
// monitor) encerram o loop e são devolvidas aqui.
func (a *App) Run() error {
	a.render()
	for !a.window.ShouldClose() {
		// Com timers pendentes (piscada do cursor, tooltip), o loop acorda
		// no vencimento mais próximo; sem timers, bloqueia como sempre.
		if wait, ok := a.nextTimerWait(); ok {
			if wait > 0 {
				glfw.WaitEventsTimeout(wait.Seconds())
			}
		} else {
			glfw.WaitEvents()
		}
		a.runDueTimers()
		if a.fatalErr != nil {
			a.destroy()
			return a.fatalErr
		}
		if a.dirty {
			a.render()
		}
	}
	a.destroy()
	return nil
}

// destroy libera os recursos GL e a janela.
func (a *App) destroy() {
	hooks.Repaint = nil
	hooks.ClipboardRead = nil
	hooks.ClipboardWrite = nil
	hooks.Schedule = nil
	hooks.OpenOverlay = nil
	hooks.CloseOverlay = nil
	a.blitter.Destroy()
	a.window.Destroy()
	glfw.Terminate()
}
