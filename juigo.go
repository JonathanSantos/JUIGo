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
type App struct {
	window  *glfw.Window
	blitter *render.Blitter
	buf     *image.RGBA
	theme   *Theme
	root    Widget
	hovered Widget
	width   int
	height  int
	dirty   bool
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

	blitter, err := render.NewBlitter(width, height)
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

	a := &App{
		window:  window,
		blitter: blitter,
		theme:   theme,
		buf:     image.NewRGBA(image.Rect(0, 0, width, height)),
		width:   width,
		height:  height,
		dirty:   true,
	}
	a.installCallbacks()
	return a, nil
}

// installCallbacks registra os callbacks de janela do GLFW. Os callbacks
// mutam o estado do App diretamente — sem mutex e sem goroutines, pois tudo
// executa na main thread dentro de glfw.WaitEvents.
func (a *App) installCallbacks() {
	a.window.SetSizeCallback(func(_ *glfw.Window, w, h int) {
		a.resize(w, h)
	})
	a.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		x, y := a.window.GetCursorPos()
		pos := image.Pt(int(x), int(y))
		kind := MouseDown
		if action == glfw.Release {
			kind = MouseUp
		}
		a.dispatch(MouseEvent{Kind: kind, Pos: pos, Button: mapMouseButton(button)})
	})
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		pos := image.Pt(int(x), int(y))
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

// resize realoca o buffer RGBA para o novo tamanho lógico da janela.
// Esta é a única situação em que o buffer é realocado.
func (a *App) resize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	a.width, a.height = w, h
	a.buf = image.NewRGBA(image.Rect(0, 0, w, h))
	a.dirty = true
}

// Theme devolve o tema da aplicação, usado na construção dos widgets.
func (a *App) Theme() *Theme {
	return a.theme
}

// SetRoot define o widget raiz da árvore de interface e agenda um redesenho.
// A raiz recebe Layout com os limites do buffer a cada renderização.
func (a *App) SetRoot(w Widget) {
	a.root = w
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
func (a *App) Run() error {
	a.render()
	for !a.window.ShouldClose() {
		glfw.WaitEvents()
		if a.dirty {
			a.render()
		}
	}
	a.destroy()
	return nil
}

// destroy libera os recursos GL e a janela.
func (a *App) destroy() {
	a.blitter.Destroy()
	a.window.Destroy()
	glfw.Terminate()
}
