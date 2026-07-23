package juigo

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"sync"
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
// (via render.Blitter), dos timers e do loop de eventos. Toda a lógica de
// INTERAÇÃO (roteamento, foco, captura, hover, overlay, tooltip) vive na
// widget.Session — o App apenas traduz os eventos do GLFW para ela, o que
// permite testar o comportamento real headless com juigo/uitest.
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
	session *widget.Session
	width   int // dimensões do buffer, em pixels físicos
	height  int
	// pixelRatio converte coordenadas lógicas da janela (mouse) em pixels
	// do framebuffer. 2 em telas retina; 1 em telas comuns.
	pixelRatio float64
	dirty      bool
	// fatalErr registra uma falha ocorrida dentro de um callback (que não
	// pode devolver erro); Run a detecta e encerra.
	fatalErr error
	// appliedCursor é o formato de cursor aplicado na janela; stdCursors
	// cacheia os cursores padrão do GLFW já criados.
	appliedCursor widget.CursorShape
	stdCursors    map[widget.CursorShape]*glfw.Cursor
	// timers são os agendamentos pendentes (hooks.Schedule): o loop usa
	// WaitEventsTimeout para acordar no vencimento mais próximo.
	timers   []appTimer
	timerSeq int
	// postMu protege posted — a ÚNICA estrutura do JUIGo tocada por outras
	// goroutines (ver Post).
	postMu sync.Mutex
	posted []func()
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
		session:    widget.NewSession(th),
		buf:        image.NewRGBA(image.Rect(0, 0, fbw, fbh)),
		width:      fbw,
		height:     fbh,
		pixelRatio: pixelRatio(window, fbw),
		dirty:      true,
	}
	a.session.OnDirty = func() { a.dirty = true }
	a.session.Resize(image.Pt(fbw, fbh))
	a.installCallbacks()
	// Setters de widgets e State.Set redesenham através deste hook; o Input
	// copia/cola através da área de transferência do sistema; popups abrem
	// pela Session; timers acordam o loop.
	hooks.Repaint = a.Invalidate
	hooks.ClipboardRead = window.GetClipboardString
	hooks.ClipboardWrite = window.SetClipboardString
	hooks.Schedule = a.schedule
	hooks.OpenOverlay = func(v any) {
		if w, ok := v.(widget.Widget); ok {
			a.session.OpenOverlay(w)
		}
	}
	hooks.CloseOverlay = func(v any) {
		if w, ok := v.(widget.Widget); ok {
			a.session.CloseOverlayIf(w)
		}
	}
	return a, nil
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

// pixelRatio calcula a razão framebuffer/janela (pixels físicos por unidade
// lógica de janela), usada para converter coordenadas de mouse.
func pixelRatio(window *glfw.Window, fbWidth int) float64 {
	winW, _ := window.GetSize()
	if winW <= 0 || fbWidth <= 0 {
		return 1
	}
	return float64(fbWidth) / float64(winW)
}

// installCallbacks registra os callbacks de janela do GLFW, traduzindo cada
// evento para a Session. Tudo executa na main thread dentro de WaitEvents.
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
		if action == glfw.Release {
			a.session.PointerUp(pos, mapMouseButton(button))
			return
		}
		a.session.PointerDown(pos, mapMouseButton(button))
	})
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, rawMods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		a.session.KeyPress(mapKey(key), mapMods(rawMods))
	})
	a.window.SetCharCallback(func(_ *glfw.Window, r rune) {
		a.session.Char(r)
	})
	a.window.SetScrollCallback(func(_ *glfw.Window, dx, dy float64) {
		x, y := a.window.GetCursorPos()
		a.session.Scroll(a.toBufferCoords(x, y), dx, dy)
	})
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		a.session.PointerMove(a.toBufferCoords(x, y))
		a.applyCursor(a.session.CursorShape())
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

// applyCursor troca o cursor da janela para o formato dado, criando (e
// cacheando) os cursores padrão do GLFW sob demanda.
func (a *App) applyCursor(shape widget.CursorShape) {
	if shape == a.appliedCursor {
		return
	}
	a.appliedCursor = shape
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
	case glfw.KeyTab:
		return event.KeyTab
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
	a.session.Resize(image.Pt(w, h))
	a.dirty = true
}

// Bus devolve o barramento de eventos da aplicação (Publish síncrono), para
// comunicação entre partes do código do usuário.
func (a *App) Bus() *event.Bus {
	return a.bus
}

// Theme devolve o tema da aplicação, usado na construção dos widgets.
func (a *App) Theme() *theme.Theme {
	return a.theme
}

// Session devolve o núcleo de interação da aplicação (foco, hover, overlay).
func (a *App) Session() *widget.Session {
	return a.session
}

// SetRoot define o widget raiz da árvore de interface, injeta o tema da
// aplicação na árvore (mount) e agenda um redesenho.
func (a *App) SetRoot(w widget.Widget) {
	a.session.SetRoot(w)
	a.dirty = true
}

// Invalidate marca a interface como suja e acorda o loop de eventos, forçando
// uma nova renderização. Útil para mudanças de estado feitas fora do fluxo de
// eventos da janela.
func (a *App) Invalidate() {
	a.dirty = true
	glfw.PostEmptyEvent()
}

// Post agenda fn para executar na MAIN THREAD, na próxima volta do loop de
// eventos. É o ÚNICO método do JUIGo seguro para chamar de outras goroutines
// — a ponte para trabalho assíncrono (rede, disco): faça o trabalho pesado
// em uma goroutine e entregue o resultado à interface via Post; DENTRO de fn,
// State.Set e os setters de widgets são seguros como sempre.
//
//	btn.SetLoading(true)
//	go func() {
//	    resultado := buscar()
//	    app.Post(func() {
//	        btn.SetLoading(false)
//	        estado.Set(resultado)
//	    })
//	}()
func (a *App) Post(fn func()) {
	if fn == nil {
		return
	}
	a.postMu.Lock()
	a.posted = append(a.posted, fn)
	a.postMu.Unlock()
	glfw.PostEmptyEvent() // thread-safe: acorda o loop
}

// runPosted executa, na main thread, os callbacks entregues via Post.
func (a *App) runPosted() {
	a.postMu.Lock()
	batch := a.posted
	a.posted = nil
	a.postMu.Unlock()
	for _, fn := range batch {
		fn()
	}
}

// render compõe o frame pela Session, envia para a textura e apresenta.
// Não aloca: o buffer é reutilizado entre frames.
func (a *App) render() {
	if !a.dirty || a.width <= 0 || a.height <= 0 {
		return
	}
	a.session.Render(a.buf)
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
		a.runPosted()
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
