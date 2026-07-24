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

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
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

	// buf é o framebuffer PERSISTENTE do harness: como no App real, o render
	// incremental repinta apenas as regiões danificadas sobre ele.
	buf *image.RGBA

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

// NewLazy cria o harness e SÓ ENTÃO constrói a raiz com build — para UIs
// que agendam timers ou animações na própria construção: quando build
// roda, os hooks já apontam para o relógio virtual (com uitest.New, a
// árvore é construída ANTES e os agendamentos se perdem — anim salta ao
// alvo).
func NewLazy(t testing.TB, build func() widget.Widget, width, height int) *Harness {
	t.Helper()
	th, err := theme.Default()
	if err != nil {
		t.Fatalf("uitest: theme.Default: %v", err)
	}
	h := newWith(t, widget.NewContainer(), th, width, height)
	h.session.SetRoot(build())
	h.Layout()
	return h
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

	h.buf = image.NewRGBA(image.Rectangle{Max: h.size})

	prevRepaint := hooks.Repaint
	prevDamage := hooks.Damage
	prevFrame := hooks.Frame
	prevSchedule := hooks.Schedule
	prevOpen := hooks.OpenOverlay
	prevClose := hooks.CloseOverlay
	prevFocus := hooks.Focus
	prevToast := hooks.Toast
	prevDrag := hooks.StartDrag
	hooks.Repaint = func() {
		h.Repaints++
		h.session.InvalidateAll()
	}
	hooks.Damage = func(r image.Rectangle) {
		h.session.AddDamage(r)
	}
	hooks.Frame = func() {}
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
	hooks.Focus = func(v any) {
		w, _ := v.(widget.Widget)
		h.session.Focus(w)
	}
	hooks.Toast = func(text string, d time.Duration) {
		h.session.ShowToast(text, d)
	}
	hooks.StartDrag = func(payload any, label string) {
		h.session.StartDrag(payload, label)
	}
	t.Cleanup(func() {
		hooks.Repaint = prevRepaint
		hooks.Damage = prevDamage
		hooks.Frame = prevFrame
		hooks.Schedule = prevSchedule
		hooks.OpenOverlay = prevOpen
		hooks.CloseOverlay = prevClose
		hooks.Focus = prevFocus
		hooks.Toast = prevToast
		hooks.StartDrag = prevDrag
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

// Layout força um passe de render incremental no framebuffer persistente —
// útil após mudanças de estado, antes de calcular posições de clique.
func (h *Harness) Layout() {
	h.session.Render(h.buf)
}

// sync garante um frame antes de interações por GEOMETRIA — a cadência do
// App real (um frame por evento): árvores recém-montadas ou reconstruídas
// ganham bounds antes de Find/cliques calcularem posições.
func (h *Harness) sync() {
	h.session.Render(h.buf)
}

// Screenshot compõe o frame atual como o App faria (render INCREMENTAL sobre
// o framebuffer persistente: só as regiões danificadas são repintadas) e
// devolve uma CÓPIA da imagem — determinística e segura para comparar entre
// chamadas.
func (h *Harness) Screenshot() *image.RGBA {
	h.session.Render(h.buf)
	clone := image.NewRGBA(h.buf.Bounds())
	copy(clone.Pix, h.buf.Pix)
	return clone
}

// Focused devolve o widget focado (nil se nenhum).
func (h *Harness) Focused() widget.Widget {
	return h.session.Focused()
}

// MoveTo move o ponteiro para pos (hover, cursor, tooltip agendado).
func (h *Harness) MoveTo(pos image.Point) {
	h.sync()
	h.session.PointerMove(pos)
}

// Hover move o ponteiro para o centro do widget selecionado.
func (h *Harness) Hover(sel Selector) {
	h.t.Helper()
	h.MoveTo(center(h.mustFind(sel).Bounds()))
}

// ClickAt faz um clique completo (move, pressiona, solta) na posição dada.
func (h *Harness) ClickAt(pos image.Point) {
	h.sync()
	h.MoveTo(pos)
	h.session.PointerDown(pos, event.MouseButtonLeft)
	h.session.PointerUp(pos, event.MouseButtonLeft)
}

// Click faz um clique completo no centro do widget selecionado.
func (h *Harness) Click(sel Selector) {
	h.t.Helper()
	h.ClickAt(center(h.mustFind(sel).Bounds()))
}

// RightClickAt faz um clique completo com o botão DIREITO na posição dada
// (menus de contexto, ajustes).
func (h *Harness) RightClickAt(pos image.Point) {
	h.sync()
	h.MoveTo(pos)
	h.session.PointerDown(pos, event.MouseButtonRight)
	h.session.PointerUp(pos, event.MouseButtonRight)
}

// RightClick faz um clique com o botão direito no centro do widget
// selecionado.
func (h *Harness) RightClick(sel Selector) {
	h.t.Helper()
	h.RightClickAt(center(h.mustFind(sel).Bounds()))
}

// Drag pressiona em from, arrasta (com captura, como no App real) e solta em
// to.
func (h *Harness) Drag(from, to image.Point) {
	h.sync()
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

// Preedit entrega ao widget focado uma pré-edição de IME: o texto em
// composição, a posição do cursor dentro dele (em runes) e, opcionalmente,
// os tamanhos dos blocos. Texto vazio encerra a composição; o commit é o
// Type normal. Para controlar o bloco focado, use Session().Preedit com o
// event.PreeditEvent completo.
func (h *Harness) Preedit(text string, caret int, blocks ...int) {
	h.session.Preedit(event.PreeditEvent{Text: text, Caret: caret, Blocks: blocks})
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
	h.sync()
	if ov := h.session.Overlay(); ov != nil {
		if w := findIn(ov, sel); w != nil {
			return w
		}
	}
	return findIn(h.session.Root(), sel)
}

// FindAll devolve TODOS os widgets que casam com o seletor, em ordem de
// árvore (overlay primeiro, depois a raiz). Útil quando vários widgets
// compartilham o mesmo texto ou tipo.
func (h *Harness) FindAll(sel Selector) []widget.Widget {
	h.sync()
	var out []widget.Widget
	if ov := h.session.Overlay(); ov != nil {
		collectIn(ov, sel, &out)
	}
	collectIn(h.session.Root(), sel, &out)
	return out
}

// collectIn acumula em out todos os widgets da árvore que casam com sel.
func collectIn(root widget.Widget, sel Selector, out *[]widget.Widget) {
	if root == nil {
		return
	}
	if sel.Match(root) {
		*out = append(*out, root)
	}
	if p, ok := root.(widget.ParentWidget); ok {
		for _, ch := range p.Children() {
			collectIn(ch, sel, out)
		}
	}
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
