package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Tabs organiza páginas de conteúdo em abas: uma barra de rótulos no topo e
// a página ativa abaixo. Só a página ativa participa do desenho, do
// hit-test e da cadeia de foco — as demais ficam dormentes até serem
// selecionadas.
//
//	juigo.NewTabs().
//	    Add("Dados", formDados).
//	    Add("Notas", notas)
//
// A barra é focável: setas Esquerda/Direita movem a aba ativa (Home/End vão
// aos extremos) e o clique seleciona. BindSelected vincula o índice ativo a
// um State[int] em duas vias; OnChange observa trocas.
type Tabs struct {
	BaseWidget
	labels []string
	pages  []Widget

	selected int
	hover    int // aba sob o ponteiro; -1 = nenhuma
	focused  bool
	bound    *state.State[int]
	onChange func(int)
	// child é o scratch devolvido por Children (sem alocação por chamada).
	child [1]Widget
	// clip é a visão recortada reutilizada ao desenhar a barra.
	clip image.RGBA
}

// NewTabs cria um conjunto de abas vazio; adicione páginas com Add. O tema
// é herdado no mount.
func NewTabs() *Tabs {
	return &Tabs{hover: -1}
}

// Add anexa uma aba com o rótulo e a página dados. Encadeável.
func (t *Tabs) Add(label string, page Widget) *Tabs {
	t.labels = append(t.labels, label)
	t.pages = append(t.pages, page)
	t.Invalidate()
	return t
}

// OnChange define o callback chamado quando a aba ativa muda, com o índice
// novo. Encadeável.
func (t *Tabs) OnChange(fn func(int)) *Tabs {
	t.onChange = fn
	return t
}

// BindSelected vincula o índice da aba ativa ao State em duas vias: trocar
// a aba faz Set, e um Set externo troca a aba (índices fora do intervalo
// são ajustados ao mais próximo). Encadeável.
func (t *Tabs) BindSelected(s *state.State[int]) *Tabs {
	t.bound = s
	t.Select(s.Get())
	s.Watch(func(v int) {
		t.Select(v)
	})
	return t
}

// Selected devolve o índice da aba ativa (zero em abas recém-criadas).
func (t *Tabs) Selected() int {
	return t.selected
}

// Select troca a aba ativa para o índice dado, ajustado ao intervalo
// válido. Dispara OnChange e sincroniza o State vinculado quando a aba
// efetivamente muda.
func (t *Tabs) Select(i int) {
	if len(t.pages) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= len(t.pages) {
		i = len(t.pages) - 1
	}
	if i == t.selected {
		// Mantém o State vinculado em sincronia quando o Set externo veio
		// fora do intervalo e foi ajustado.
		if t.bound != nil && t.bound.Get() != i {
			t.bound.Set(i)
		}
		return
	}
	t.selected = i
	t.Invalidate()
	if t.bound != nil && t.bound.Get() != i {
		t.bound.Set(i)
	}
	if t.onChange != nil {
		t.onChange(i)
	}
}

// Children devolve apenas a página ativa: as dormentes ficam fora do
// roteamento, do foco e do mount até serem selecionadas.
func (t *Tabs) Children() []Widget {
	p := t.page()
	if p == nil {
		return nil
	}
	t.child[0] = p
	return t.child[:]
}

// SetTheme define um tema explícito e o propaga à subárvore atual, como os
// containers.
func (t *Tabs) SetTheme(th *theme.Theme) {
	t.BaseWidget.SetTheme(th)
	Mount(t, th)
}

// Focusable devolve true: a barra de abas participa da cadeia de foco.
func (t *Tabs) Focusable() bool {
	return true
}

// CursorShape devolve a mãozinha — sobre a página ativa vale o cursor do
// widget mais profundo dela, não este.
func (t *Tabs) CursorShape() CursorShape {
	return CursorHand
}

// page devolve a página ativa, ou nil.
func (t *Tabs) page() Widget {
	if t.selected < 0 || t.selected >= len(t.pages) {
		return nil
	}
	return t.pages[t.selected]
}

// barHeight devolve a altura da barra de abas em pixels.
func (t *Tabs) barHeight() int {
	return t.theme.LineHeight() + 2*t.theme.PaddingPx()
}

// tabAt devolve o índice da aba sob o ponto, ou -1 (fora da barra ou no
// espaço livre à direita dos rótulos).
func (t *Tabs) tabAt(p image.Point) int {
	if t.theme == nil {
		return -1
	}
	b := t.Bounds()
	if p.Y < b.Min.Y || p.Y >= b.Min.Y+t.barHeight() {
		return -1
	}
	pad := t.theme.PaddingPx()
	x := b.Min.X
	for i, label := range t.labels {
		w := t.theme.MeasureString(label) + 2*pad
		if p.X >= x && p.X < x+w {
			return i
		}
		x += w
	}
	return -1
}

// inBar informa se o ponto está na faixa da barra de abas.
func (t *Tabs) inBar(p image.Point) bool {
	b := t.Bounds()
	return p.In(b) && p.Y < b.Min.Y+t.barHeight()
}

// Layout posiciona a página ativa abaixo da barra de abas.
func (t *Tabs) Layout(bounds image.Rectangle) {
	t.BaseWidget.Layout(bounds)
	if t.theme == nil {
		return
	}
	if p := t.page(); p != nil {
		p.Layout(image.Rect(bounds.Min.X, bounds.Min.Y+t.barHeight(), bounds.Max.X, bounds.Max.Y))
	}
}

// PreferredSize devolve a largura da barra (soma dos rótulos com respiro)
// ou da página mais larga, e a altura da barra mais a página mais alta —
// o máximo entre TODAS as páginas, para o tamanho não pular ao trocar de
// aba.
func (t *Tabs) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	pad := t.theme.PaddingPx()
	var barW int
	var page image.Point
	for i, label := range t.labels {
		barW += t.theme.MeasureString(label) + 2*pad
		pref := t.pages[i].PreferredSize()
		page.X = max(page.X, pref.X)
		page.Y = max(page.Y, pref.Y)
	}
	return image.Pt(max(barW, page.X), t.barHeight()+page.Y)
}

// HandleEvent troca a aba por clique e teclado e mantém o hover da barra.
func (t *Tabs) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if len(t.pages) == 0 {
			return false
		}
		switch e.Key {
		case event.KeyLeft:
			t.Select(t.selected - 1)
		case event.KeyRight:
			t.Select(t.selected + 1)
		case event.KeyHome:
			t.Select(0)
		case event.KeyEnd:
			t.Select(len(t.pages) - 1)
		default:
			return false
		}
		return true
	case event.FocusEvent:
		t.focused = e.Gained
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseMove, event.MouseEnter:
			if i := t.tabAt(e.Pos); i != t.hover {
				t.hover = i
				return true
			}
			return false
		case event.MouseLeave:
			if t.hover != -1 {
				t.hover = -1
				return true
			}
			return false
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft || !t.inBar(e.Pos) {
				return false
			}
			if i := t.tabAt(e.Pos); i >= 0 {
				t.Select(i)
			}
			return true
		case event.MouseUp:
			return t.inBar(e.Pos)
		}
	}
	return false
}

// Draw desenha a barra (separador, rótulos, realce de hover, sublinhado da
// ativa e anel de foco) e a página ativa.
func (t *Tabs) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	th := t.theme
	b := t.Bounds()
	pad := th.PaddingPx()
	barH := t.barHeight()
	radius := th.RadiusPx()
	view := render.Clip(dst, b, &t.clip)

	// Separador na base da barra; o sublinhado da aba ativa passa por cima.
	render.FillRect(view, image.Rect(b.Min.X, b.Min.Y+barH-th.BorderPx(), b.Max.X, b.Min.Y+barH), th.InputBorder)

	under := max(th.Px(2), 2)
	x := b.Min.X
	for i, label := range t.labels {
		w := th.MeasureString(label) + 2*pad
		tab := image.Rect(x, b.Min.Y, x+w, b.Min.Y+barH)
		if i == t.hover && i != t.selected {
			render.FillRoundRect(view, image.Rect(tab.Min.X, tab.Min.Y, tab.Max.X, tab.Max.Y-th.BorderPx()), radius, th.HoverBackground)
		}
		c := th.Placeholder
		if i == t.selected {
			c = th.Text
		}
		th.DrawText(view, label, image.Pt(x+pad, b.Min.Y+(barH-th.LineHeight())/2+th.Ascent()), c)
		if i == t.selected {
			render.FillRoundRect(view, image.Rect(tab.Min.X, tab.Max.Y-under, tab.Max.X, tab.Max.Y), radius, th.Accent)
			if t.focused {
				render.StrokeRoundRect(view, tab, radius, 2*th.BorderPx(), th.FocusOutline)
			}
		}
		x += w
	}

	if p := t.page(); p != nil && p.Bounds().Overlaps(dst.Bounds()) {
		p.Draw(dst)
	}
	t.drawDisabledOverlay(dst)
}
