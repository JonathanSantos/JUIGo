package juigo

import (
	"fmt"
	"image"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
)

// Window é uma janela da aplicação: dona da janela GLFW, do buffer RGBA, da
// textura GL, do PRÓPRIO tema (a escala segue o monitor desta janela) e da
// própria widget.Session — foco, hover, overlay, tooltip e toast são por
// janela. A primeira nasce com New/Run; as demais com App.NewWindow. A
// aplicação termina quando TODAS as janelas fecham.
type Window struct {
	app     *App
	window  *glfw.Window
	blitter *render.Blitter
	buf     *image.RGBA
	theme   *theme.Theme
	session *widget.Session
	width   int // dimensões do buffer, em pixels físicos
	height  int
	// pixelRatio converte coordenadas lógicas da janela (mouse) em pixels
	// do framebuffer. 2 em telas retina; 1 em telas comuns.
	pixelRatio float64
	dirty      bool
	closed     bool
	onClose    func()

	// appliedCursor é o formato de cursor aplicado na janela; stdCursors
	// cacheia os cursores padrão do GLFW já criados.
	appliedCursor widget.CursorShape
	stdCursors    map[widget.CursorShape]*glfw.Cursor
}

// createWindow cria uma janela GLFW completa (contexto GL, blitter, sessão)
// com o tema dado, já escalado para o monitor onde ela abriu, e a registra
// na aplicação.
func (a *App) createWindow(title string, width, height int, th *theme.Theme) (*Window, error) {
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True) // exigido no macOS
	glfw.WindowHint(glfw.Resizable, glfw.True)

	window, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
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
		return nil, err
	}

	// Escala o tema (fonte, métricas) para o monitor onde a janela abriu.
	scaleX, _ := window.GetContentScale()
	if scaleX > 0 && float64(scaleX) != th.Scale() {
		if err := th.SetScale(float64(scaleX)); err != nil {
			blitter.Destroy()
			window.Destroy()
			return nil, err
		}
	}

	w := &Window{
		app:        a,
		window:     window,
		blitter:    blitter,
		theme:      th,
		session:    widget.NewSession(th),
		buf:        image.NewRGBA(image.Rect(0, 0, fbw, fbh)),
		width:      fbw,
		height:     fbh,
		pixelRatio: pixelRatio(window, fbw),
		dirty:      true,
	}
	w.session.OnDirty = func() { w.dirty = true }
	w.session.Resize(image.Pt(fbw, fbh))
	w.installCallbacks()
	a.windows = append(a.windows, w)
	return w, nil
}

// Theme devolve o tema desta janela.
func (w *Window) Theme() *theme.Theme {
	return w.theme
}

// Session devolve o núcleo de interação desta janela (foco, hover, overlay).
func (w *Window) Session() *widget.Session {
	return w.session
}

// SetRoot define o widget raiz da árvore desta janela e agenda um redesenho.
func (w *Window) SetRoot(root widget.Widget) {
	w.session.SetRoot(root)
	w.dirty = true
}

// SetTheme troca o tema desta janela em runtime: o novo tema é levado à
// escala atual do monitor e re-propagado pela árvore no próximo frame
// (widgets com SetTheme explícito mantêm o próprio). Cada janela precisa do
// PRÓPRIO *Theme — não compartilhe a mesma instância entre janelas (use
// Theme.Clone).
func (w *Window) SetTheme(th *theme.Theme) error {
	if th == nil || th == w.theme {
		return nil
	}
	if err := th.SetScale(w.theme.Scale()); err != nil {
		return err
	}
	w.theme = th
	w.session.SetTheme(th)
	w.dirty = true
	return nil
}

// Invalidate marca a interface DESTA janela inteira como suja e acorda o
// loop — a válvula de escape para mutação direta de campos públicos.
func (w *Window) Invalidate() {
	w.session.InvalidateAll()
	w.dirty = true
	glfw.PostEmptyEvent()
}

// OnClose registra o callback chamado quando a janela fechar (pelo botão do
// sistema ou por Close). Encadeável.
func (w *Window) OnClose(fn func()) *Window {
	w.onClose = fn
	return w
}

// Close pede o fechamento da janela: o loop a destrói na próxima volta e,
// se era a última, a aplicação termina.
func (w *Window) Close() {
	w.window.SetShouldClose(true)
	glfw.PostEmptyEvent()
}

// destroy chama OnClose e libera os recursos GL e a janela.
func (w *Window) destroy() {
	if w.closed {
		return
	}
	w.closed = true
	if w.onClose != nil {
		w.onClose()
	}
	w.window.MakeContextCurrent()
	w.blitter.Destroy()
	w.window.Destroy()
}

// installCallbacks registra os callbacks GLFW desta janela, traduzindo cada
// evento para a Session dela. Durante a entrega, a janela fica marcada como
// a "em interação" — é para a sessão dela que os hooks globais (overlay,
// foco programático, toast, arrasto) roteiam.
func (w *Window) installCallbacks() {
	w.window.SetFramebufferSizeCallback(func(_ *glfw.Window, largura, altura int) {
		w.app.dispatch(w, func() { w.resize(largura, altura) })
	})
	w.window.SetContentScaleCallback(func(_ *glfw.Window, scaleX, _ float32) {
		// Janela mudou de monitor: refaz fonte, métricas e cache de glyphs.
		if scaleX <= 0 {
			return
		}
		if err := w.theme.SetScale(float64(scaleX)); err != nil {
			w.app.fatalErr = err
			return
		}
		w.session.InvalidateAll()
		w.dirty = true
	})
	w.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		w.app.dispatch(w, func() {
			x, y := w.window.GetCursorPos()
			pos := w.toBufferCoords(x, y)
			if action == glfw.Release {
				w.session.PointerUp(pos, mapMouseButton(button))
				return
			}
			w.session.PointerDown(pos, mapMouseButton(button))
		})
	})
	w.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, rawMods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		w.app.dispatch(w, func() { w.session.KeyPress(mapKey(key), mapMods(rawMods)) })
	})
	w.window.SetCharCallback(func(_ *glfw.Window, r rune) {
		w.app.dispatch(w, func() { w.session.Char(r) })
	})
	w.window.SetScrollCallback(func(_ *glfw.Window, dx, dy float64) {
		w.app.dispatch(w, func() {
			x, y := w.window.GetCursorPos()
			w.session.Scroll(w.toBufferCoords(x, y), dx, dy)
		})
	})
	w.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		w.app.dispatch(w, func() {
			w.session.PointerMove(w.toBufferCoords(x, y))
			w.applyCursor(w.session.CursorShape())
		})
	})
	// Refresh dispara quando o SO precisa que a janela seja repintada
	// (ex.: durante o redimensionamento, que roda um loop modal no macOS).
	w.window.SetRefreshCallback(func(_ *glfw.Window) {
		w.dirty = true
		w.render()
	})
}

// toBufferCoords converte coordenadas lógicas de janela (como o GLFW entrega
// a posição do cursor) em pixels do buffer/framebuffer.
func (w *Window) toBufferCoords(x, y float64) image.Point {
	return image.Pt(
		int(math.Round(x*w.pixelRatio)),
		int(math.Round(y*w.pixelRatio)),
	)
}

// applyCursor troca o cursor da janela para o formato dado, criando (e
// cacheando) os cursores padrão do GLFW sob demanda.
func (w *Window) applyCursor(shape widget.CursorShape) {
	if shape == w.appliedCursor {
		return
	}
	w.appliedCursor = shape
	if shape == widget.CursorDefault {
		w.window.SetCursor(nil)
		return
	}
	if w.stdCursors == nil {
		w.stdCursors = make(map[widget.CursorShape]*glfw.Cursor)
	}
	cur, ok := w.stdCursors[shape]
	if !ok {
		switch shape {
		case widget.CursorText:
			cur = glfw.CreateStandardCursor(glfw.IBeamCursor)
		case widget.CursorHand:
			cur = glfw.CreateStandardCursor(glfw.HandCursor)
		}
		w.stdCursors[shape] = cur
	}
	w.window.SetCursor(cur)
}

// resize realoca o buffer RGBA para o novo tamanho do framebuffer, em pixels
// físicos. Esta é a única situação em que o buffer é realocado.
func (w *Window) resize(largura, altura int) {
	if largura <= 0 || altura <= 0 {
		return
	}
	w.width, w.height = largura, altura
	w.buf = image.NewRGBA(image.Rect(0, 0, largura, altura))
	w.pixelRatio = pixelRatio(w.window, largura)
	w.session.Resize(image.Pt(largura, altura))
	w.dirty = true
}

// render compõe o frame pela Session desta janela, envia para a textura e
// apresenta. Cada janela tem o próprio contexto GL — ele fica corrente só
// durante o upload/present. Não aloca: o buffer é reutilizado entre frames.
func (w *Window) render() {
	if !w.dirty || w.closed || w.width <= 0 || w.height <= 0 {
		return
	}
	region, full := w.session.Render(w.buf)
	w.dirty = false
	if region.Empty() {
		return // frame agendado sem dano visível: nada a apresentar
	}
	w.window.MakeContextCurrent()
	// Dirty regions: só a região repintada sobe para a GPU.
	if full {
		w.blitter.Upload(w.buf)
	} else {
		w.blitter.UploadRegion(w.buf, region)
	}
	fbw, fbh := w.window.GetFramebufferSize()
	w.blitter.Draw(fbw, fbh)
	w.window.SwapBuffers()
}
