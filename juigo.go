// Package juigo é uma biblioteca minimalista de interface gráfica para Go.
//
// A renderização é feita por software (CPU) sobre um buffer *image.RGBA;
// GLFW fornece a janela e os eventos do sistema operacional, e o OpenGL é
// usado apenas para apresentar o buffer na tela (blit de textura em um quad
// fullscreen). Toda a biblioteca é single-threaded: janela, eventos e
// desenho vivem na main thread do processo.
package juigo

import (
	"fmt"
	"image"
	"math"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"

	"juigo/render"
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
	theme   *Theme
	root    Widget
	hovered Widget
	focused Widget
	width   int // dimensões do buffer, em pixels físicos
	height  int
	// pixelRatio converte coordenadas lógicas da janela (mouse) em pixels
	// do framebuffer. 2 em telas retina; 1 em telas comuns.
	pixelRatio float64
	dirty      bool
	// fatalErr registra uma falha ocorrida dentro de um callback (que não
	// pode devolver erro); Run a detecta e encerra.
	fatalErr error
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

	theme, err := DefaultTheme()
	if err != nil {
		blitter.Destroy()
		window.Destroy()
		glfw.Terminate()
		return nil, err
	}

	// Escala o tema (fonte, métricas) para o monitor onde a janela abriu.
	scaleX, _ := window.GetContentScale()
	if scaleX > 0 && scaleX != 1 {
		if err := theme.SetScale(float64(scaleX)); err != nil {
			blitter.Destroy()
			window.Destroy()
			glfw.Terminate()
			return nil, err
		}
	}

	a := &App{
		window:     window,
		blitter:    blitter,
		theme:      theme,
		buf:        image.NewRGBA(image.Rect(0, 0, fbw, fbh)),
		width:      fbw,
		height:     fbh,
		pixelRatio: pixelRatio(window, fbw),
		dirty:      true,
	}
	a.installCallbacks()
	// Setters de widgets e State.Set redesenham através deste hook.
	repaintHook = a.Invalidate
	return a, nil
}

// Run cria a aplicação com New, define root como raiz da árvore e executa o
// loop de eventos até a janela ser fechada. É o caminho curto para a maioria
// das aplicações; use New quando precisar do *App (tema, Invalidate, Bus).
func Run(title string, width, height int, root Widget) error {
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
		kind := MouseDown
		if action == glfw.Release {
			kind = MouseUp
		}
		if kind == MouseDown {
			// Clique em widget focável muda o foco; em área sem widget
			// focável, limpa o foco.
			a.setFocus(focusableAt(a.root, pos))
		}
		a.dispatch(MouseEvent{Kind: kind, Pos: pos, Button: mapMouseButton(button)})
	})
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if key == glfw.KeyTab {
			a.focusNext()
			return
		}
		k := mapKey(key)
		if k == KeyUnknown || a.focused == nil {
			return
		}
		// Teclado roteia por FOCO: direto ao widget focado, sem hit-test.
		if a.focused.HandleEvent(KeyEvent{Key: k}) {
			a.dirty = true
		}
	})
	a.window.SetCharCallback(func(_ *glfw.Window, r rune) {
		if a.focused == nil {
			return
		}
		if a.focused.HandleEvent(CharEvent{Rune: r}) {
			a.dirty = true
		}
	})
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		pos := a.toBufferCoords(x, y)
		a.updateHover(pos)
		a.dispatch(MouseEvent{Kind: MouseMove, Pos: pos, Button: MouseButtonLeft})
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

// dispatch roteia um evento de mouse pela árvore (por geometria) e marca a
// interface como suja se algum widget o consumir.
func (a *App) dispatch(ev MouseEvent) {
	if a.root == nil {
		return
	}
	if dispatchMouse(a.root, ev) {
		a.dirty = true
	}
}

// updateHover mantém o widget sob o cursor, entregando MouseLeave ao widget
// anterior e MouseEnter ao novo quando o alvo muda.
func (a *App) updateHover(pos image.Point) {
	target := widgetAt(a.root, pos)
	if target == a.hovered {
		return
	}
	if a.hovered != nil && a.hovered.HandleEvent(MouseEvent{Kind: MouseLeave, Pos: pos}) {
		a.dirty = true
	}
	a.hovered = target
	if target != nil && target.HandleEvent(MouseEvent{Kind: MouseEnter, Pos: pos}) {
		a.dirty = true
	}
}

// setFocus move o foco de teclado para w (ou o limpa, se w for nil),
// notificando os widgets envolvidos com FocusEvent.
func (a *App) setFocus(w Widget) {
	if a.focused == w {
		return
	}
	if a.focused != nil {
		a.focused.HandleEvent(FocusEvent{Gained: false})
	}
	a.focused = w
	if w != nil {
		w.HandleEvent(FocusEvent{Gained: true})
	}
	a.dirty = true
}

// focusNext avança o foco para o próximo widget focável na ordem da árvore,
// com wraparound. Sem widgets focáveis, não faz nada.
func (a *App) focusNext() {
	var order []Widget
	collectFocusable(a.root, &order)
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

// mapKey converte teclas do GLFW para as teclas reconhecidas pelo JUIGo.
func mapKey(key glfw.Key) Key {
	switch key {
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return KeyEnter
	case glfw.KeySpace:
		return KeySpace
	case glfw.KeyBackspace:
		return KeyBackspace
	case glfw.KeyDelete:
		return KeyDelete
	case glfw.KeyLeft:
		return KeyLeft
	case glfw.KeyRight:
		return KeyRight
	case glfw.KeyHome:
		return KeyHome
	case glfw.KeyEnd:
		return KeyEnd
	default:
		return KeyUnknown
	}
}

// mapMouseButton converte o botão do GLFW para o tipo do JUIGo.
func mapMouseButton(b glfw.MouseButton) MouseButton {
	switch b {
	case glfw.MouseButtonRight:
		return MouseButtonRight
	case glfw.MouseButtonMiddle:
		return MouseButtonMiddle
	default:
		return MouseButtonLeft
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
	a.dirty = true
}

// Theme devolve o tema da aplicação, usado na construção dos widgets.
func (a *App) Theme() *Theme {
	return a.theme
}

// SetRoot define o widget raiz da árvore de interface, injeta o tema da
// aplicação na árvore (mount) e agenda um redesenho. A raiz recebe Layout
// com os limites do buffer a cada renderização.
func (a *App) SetRoot(w Widget) {
	a.root = w
	if w != nil {
		propagateTheme(w, a.theme)
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
		propagateTheme(a.root, a.theme)
		a.root.Layout(a.buf.Bounds())
		a.root.Draw(a.buf)
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
		glfw.WaitEvents()
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
	repaintHook = nil
	a.blitter.Destroy()
	a.window.Destroy()
	glfw.Terminate()
}
