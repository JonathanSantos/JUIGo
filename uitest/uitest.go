// Package uitest é o harness de testes de UI do JUIGo: monta uma árvore de
// widgets em uma widget.Session headless — o MESMO núcleo de interação que o
// App real usa — e a dirige sinteticamente: cliques, digitação, teclas,
// hover, arraste, rolagem e um RELÓGIO VIRTUAL para comportamentos temporais
// (tooltip, cursor piscante), sem janela, sem GLFW e sem sleeps.
//
//	h := uitest.New(t, ui, 480, 320)
//	h.Click(uitest.Text("Enviar"))
//	h.Type("olá, ação")
//	h.Key(juigo.KeyTab)
//	h.Advance(600 * time.Millisecond) // tooltip aparece, caret pisca
//	img := h.Screenshot()             // determinístico (golden tests)
//
// O harness assume o modelo single-threaded do JUIGo: use-o apenas na
// goroutine do teste, um harness por vez (os hooks de processo são
// redirecionados durante o teste e restaurados no Cleanup).
package uitest

import (
	"image"
	"testing"
	"time"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/theme"
	"juigo/widget"
)

// Harness dirige uma Session headless para testes.
type Harness struct {
	t       testing.TB
	session *widget.Session
	theme   *theme.Theme
	size    image.Point

	// Relógio virtual: Advance move o tempo e dispara os agendamentos.
	now      time.Duration
	timers   []vtimer
	timerSeq int

	// Repaints conta quantas vezes a interface pediu redesenho.
	Repaints int
}

type vtimer struct {
	id int
	at time.Duration
	fn func()
}

// New cria um harness com o tema padrão (escala 1) e a árvore dada, montada
// e com o tamanho dado. Os hooks do processo (timers, overlay, repaint,
// clipboard) são redirecionados para o harness e restaurados no Cleanup.
func New(t testing.TB, root widget.Widget, width, height int) *Harness {
	t.Helper()
	th, err := theme.Default()
	if err != nil {
		t.Fatalf("uitest: theme.Default: %v", err)
	}
	return newWith(t, root, th, width, height)
}

// NewWithTheme é como New, com um tema específico (ex.: escala 2).
func NewWithTheme(t testing.TB, root widget.Widget, th *theme.Theme, width, height int) *Harness {
	t.Helper()
	return newWith(t, root, th, width, height)
}

func newWith(t testing.TB, root widget.Widget, th *theme.Theme, width, height int) *Harness {
	h := &Harness{t: t, theme: th, size: image.Pt(width, height)}
	h.session = widget.NewSession(th)
	h.session.OnDirty = func() { h.Repaints++ }

	prevRepaint := hooks.Repaint
	prevSchedule := hooks.Schedule
	prevOpen := hooks.OpenOverlay
	prevClose := hooks.CloseOverlay
	hooks.Repaint = func() { h.Repaints++ }
	hooks.Schedule = h.schedule
	hooks.OpenOverlay = func(v any) {
		if w, ok := v.(widget.Widget); ok {
			h.session.OpenOverlay(w)
		}
	}
	hooks.CloseOverlay = func(v any) {
		if w, ok := v.(widget.Widget); ok {
			h.session.CloseOverlayIf(w)
		}
	}
	t.Cleanup(func() {
		hooks.Repaint = prevRepaint
		hooks.Schedule = prevSchedule
		hooks.OpenOverlay = prevOpen
		hooks.CloseOverlay = prevClose
	})

	h.session.Resize(h.size)
	h.session.SetRoot(root)
	h.Layout()
	return h
}

// schedule implementa hooks.Schedule sobre o relógio virtual.
func (h *Harness) schedule(d time.Duration, fn func()) func() {
	h.timerSeq++
	id := h.timerSeq
	h.timers = append(h.timers, vtimer{id: id, at: h.now + d, fn: fn})
	return func() {
		for i := range h.timers {
			if h.timers[i].id == id {
				h.timers = append(h.timers[:i], h.timers[i+1:]...)
				return
			}
		}
	}
}

// Advance avança o relógio virtual, disparando na ordem os agendamentos que
// vencerem no caminho (que podem agendar novos — ex.: a piscada do cursor).
func (h *Harness) Advance(d time.Duration) {
	target := h.now + d
	for {
		best := -1
		for i, tm := range h.timers {
			if tm.at <= target && (best < 0 || tm.at < h.timers[best].at) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		tm := h.timers[best]
		h.timers = append(h.timers[:best], h.timers[best+1:]...)
		h.now = tm.at
		tm.fn()
	}
	h.now = target
}

// Session expõe a sessão para asserções avançadas (Focused, Overlay, ...).
func (h *Harness) Session() *widget.Session {
	return h.session
}

// Layout força um passe de layout (Render sem capturar a imagem) — útil após
// mudanças de estado, antes de calcular posições de clique.
func (h *Harness) Layout() {
	buf := image.NewRGBA(image.Rectangle{Max: h.size})
	h.session.Render(buf)
}

// Screenshot compõe o frame atual (árvore + overlay + tooltip) e devolve a
// imagem — determinística: mesma árvore e estado produzem os mesmos bytes.
func (h *Harness) Screenshot() *image.RGBA {
	buf := image.NewRGBA(image.Rectangle{Max: h.size})
	h.session.Render(buf)
	return buf
}

// Focused devolve o widget focado (nil se nenhum).
func (h *Harness) Focused() widget.Widget {
	return h.session.Focused()
}

// MoveTo move o ponteiro para pos (hover, cursor, tooltip agendado).
func (h *Harness) MoveTo(pos image.Point) {
	h.session.PointerMove(pos)
}

// Hover move o ponteiro para o centro do widget selecionado.
func (h *Harness) Hover(sel Selector) {
	h.t.Helper()
	h.MoveTo(center(h.mustFind(sel).Bounds()))
}

// ClickAt faz um clique completo (move, pressiona, solta) na posição dada.
func (h *Harness) ClickAt(pos image.Point) {
	h.MoveTo(pos)
	h.session.PointerDown(pos, event.MouseButtonLeft)
	h.session.PointerUp(pos, event.MouseButtonLeft)
}

// Click faz um clique completo no centro do widget selecionado.
func (h *Harness) Click(sel Selector) {
	h.t.Helper()
	h.ClickAt(center(h.mustFind(sel).Bounds()))
}

// Drag pressiona em from, arrasta (com captura, como no App real) e solta em
// to.
func (h *Harness) Drag(from, to image.Point) {
	h.MoveTo(from)
	h.session.PointerDown(from, event.MouseButtonLeft)
	h.session.PointerMove(to)
	h.session.PointerUp(to, event.MouseButtonLeft)
}

// Type digita a string no widget focado, rune a rune.
func (h *Harness) Type(s string) {
	for _, r := range s {
		h.session.Char(r)
	}
}

// Key pressiona uma tecla (com modificadores opcionais, combinados por OU).
func (h *Harness) Key(k event.Key, mods ...event.Modifiers) {
	var m event.Modifiers
	for _, mod := range mods {
		m |= mod
	}
	h.session.KeyPress(k, m)
}

// Scroll rola sobre o centro do widget selecionado (dy>0 = para cima).
func (h *Harness) Scroll(sel Selector, dy float64) {
	h.t.Helper()
	h.session.Scroll(center(h.mustFind(sel).Bounds()), 0, dy)
}

// Find procura o primeiro widget que casa com o seletor, na overlay (se
// aberta) e depois na árvore, em pré-ordem. Devolve nil se não encontrar.
func (h *Harness) Find(sel Selector) widget.Widget {
	if ov := h.session.Overlay(); ov != nil {
		if w := findIn(ov, sel); w != nil {
			return w
		}
	}
	return findIn(h.session.Root(), sel)
}

func (h *Harness) mustFind(sel Selector) widget.Widget {
	h.t.Helper()
	w := h.Find(sel)
	if w == nil {
		h.t.Fatalf("uitest: nenhum widget encontrado para %s", sel.Desc)
	}
	return w
}

func findIn(root widget.Widget, sel Selector) widget.Widget {
	if root == nil {
		return nil
	}
	if sel.Match(root) {
		return root
	}
	if p, ok := root.(widget.ParentWidget); ok {
		for _, ch := range p.Children() {
			if w := findIn(ch, sel); w != nil {
				return w
			}
		}
	}
	return nil
}

func center(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}
