package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
)

// SplitPane divide a área entre dois painéis com um divisor arrastável:
// lado a lado por padrão, empilhados com Vertical. A posição é uma FRAÇÃO
// do primeiro painel (Ratio, 0..1), vinculável a um State em duas vias
// (BindRatio) — persistir a posição é guardar um float64.
//
//	juigo.NewSplitPane(arvore, detalhe).Ratio(0.3).Min(80)
//
// O divisor também é focável: setas o movem pelo teclado (Home/End vão aos
// extremos), respeitando os mínimos. A espessura da faixa de pega vem de
// Theme.SplitterThickness; a linha desenhada é mais fina.
type SplitPane struct {
	BaseWidget
	a, b     Widget
	vertical bool
	ratio    float64
	// minA e minB são os tamanhos mínimos de cada painel, em unidades
	// lógicas (aplicados em pixels no layout).
	minA, minB int
	bound      *state.State[float64]

	dragging bool
	hover    bool
	focused  bool
	// child é o scratch devolvido por Children (sem alocação por chamada).
	child [2]Widget
}

// NewSplitPane cria um divisor lado a lado entre a (esquerda) e b (direita),
// meio a meio. O tema é herdado no mount.
func NewSplitPane(a, b Widget) *SplitPane {
	return &SplitPane{a: a, b: b, ratio: 0.5}
}

// Vertical empilha os painéis (a em cima, b embaixo) com o divisor
// horizontal. Encadeável.
func (s *SplitPane) Vertical() *SplitPane {
	s.vertical = true
	s.Invalidate()
	return s
}

// Ratio define a fração do primeiro painel (0..1, ajustada ao intervalo).
// Encadeável.
func (s *SplitPane) Ratio(f float64) *SplitPane {
	s.setRatio(f)
	return s
}

// Min define o tamanho mínimo dos DOIS painéis, em unidades lógicas — o
// divisor não passa disso nem no arraste, nem pelo teclado, nem por um Set
// externo. Encadeável.
func (s *SplitPane) Min(l int) *SplitPane {
	s.minA, s.minB = l, l
	s.Invalidate()
	return s
}

// BindRatio vincula a fração ao State em duas vias: arrastar faz Set, e um
// Set externo move o divisor. Encadeável.
func (s *SplitPane) BindRatio(st *state.State[float64]) *SplitPane {
	s.bound = st
	s.setRatio(st.Get())
	st.Watch(func(v float64) { s.setRatio(v) })
	return s
}

// CurrentRatio devolve a fração atual do primeiro painel.
func (s *SplitPane) CurrentRatio() float64 {
	return s.ratio
}

// setRatio ajusta a fração ao intervalo, redesenha e sincroniza o State.
func (s *SplitPane) setRatio(f float64) {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	if f != s.ratio {
		s.ratio = f
		s.Invalidate()
	}
	if s.bound != nil && s.bound.Get() != f {
		s.bound.Set(f)
	}
}

// Children devolve os dois painéis.
func (s *SplitPane) Children() []Widget {
	s.child[0], s.child[1] = s.a, s.b
	return s.child[:]
}

// SetTheme define um tema explícito e o propaga aos painéis, como os
// containers.
func (s *SplitPane) SetTheme(th *theme.Theme) {
	s.BaseWidget.SetTheme(th)
	Mount(s, th)
}

// Focusable devolve true: o divisor participa da cadeia de foco e as setas
// o movem.
func (s *SplitPane) Focusable() bool {
	return true
}

// CursorShape devolve a seta dupla do eixo do divisor — só vale sobre a
// faixa de pega (sobre os painéis, o cursor é do widget mais profundo
// deles).
func (s *SplitPane) CursorShape() CursorShape {
	if s.vertical {
		return CursorResizeV
	}
	return CursorResizeH
}

// axisSize devolve o comprimento do eixo dividido, em pixels.
func (s *SplitPane) axisSize() int {
	if s.vertical {
		return s.Bounds().Dy()
	}
	return s.Bounds().Dx()
}

// thicknessPx devolve a espessura da faixa de pega em pixels (mínimo 1).
func (s *SplitPane) thicknessPx() int {
	t := s.theme.Px(s.theme.SplitterThickness)
	return max(t, 1)
}

// firstSpan devolve o comprimento do primeiro painel em pixels, com a
// fração aplicada ao espaço útil e os mínimos respeitados (o primeiro
// painel prevalece quando não cabem os dois).
func (s *SplitPane) firstSpan() int {
	avail := s.axisSize() - s.thicknessPx()
	if avail <= 0 {
		return 0
	}
	span := int(s.ratio*float64(avail) + 0.5)
	minA := s.theme.Px(s.minA)
	minB := s.theme.Px(s.minB)
	if span > avail-minB {
		span = avail - minB
	}
	if span < minA {
		span = minA
	}
	return clampInt(span, 0, avail)
}

// clampInt limita v ao intervalo [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// splitterRect devolve a faixa de pega do divisor, em pixels absolutos.
func (s *SplitPane) splitterRect() image.Rectangle {
	b := s.Bounds()
	span := s.firstSpan()
	t := s.thicknessPx()
	if s.vertical {
		return image.Rect(b.Min.X, b.Min.Y+span, b.Max.X, b.Min.Y+span+t)
	}
	return image.Rect(b.Min.X+span, b.Min.Y, b.Min.X+span+t, b.Max.Y)
}

// Layout posiciona os painéis dos dois lados da faixa do divisor.
func (s *SplitPane) Layout(bounds image.Rectangle) {
	s.BaseWidget.Layout(bounds)
	if s.theme == nil {
		return
	}
	sp := s.splitterRect()
	if s.vertical {
		s.a.Layout(image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, sp.Min.Y))
		s.b.Layout(image.Rect(bounds.Min.X, sp.Max.Y, bounds.Max.X, bounds.Max.Y))
		return
	}
	s.a.Layout(image.Rect(bounds.Min.X, bounds.Min.Y, sp.Min.X, bounds.Max.Y))
	s.b.Layout(image.Rect(sp.Max.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y))
}

// PreferredSize soma os painéis e a faixa no eixo dividido e toma o máximo
// no outro.
func (s *SplitPane) PreferredSize() image.Point {
	if s.theme == nil {
		return image.Point{}
	}
	pa, pb := s.a.PreferredSize(), s.b.PreferredSize()
	t := s.thicknessPx()
	if s.vertical {
		return image.Pt(max(pa.X, pb.X), pa.Y+t+pb.Y)
	}
	return image.Pt(pa.X+t+pb.X, max(pa.Y, pb.Y))
}

// ratioAt converte a posição do ponteiro na fração correspondente (o centro
// da faixa acompanha o cursor).
func (s *SplitPane) ratioAt(pos image.Point) float64 {
	b := s.Bounds()
	avail := s.axisSize() - s.thicknessPx()
	if avail <= 0 {
		return s.ratio
	}
	var along int
	if s.vertical {
		along = pos.Y - b.Min.Y
	} else {
		along = pos.X - b.Min.X
	}
	return float64(along-s.thicknessPx()/2) / float64(avail)
}

// HandleEvent arrasta o divisor (com captura de mouse) e o move pelas setas
// quando focado.
func (s *SplitPane) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		return s.handleKey(e.Key)
	case event.FocusEvent:
		s.focused = e.Gained
		s.damage(s.splitterRect())
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseEnter:
			// O deepest só é o SplitPane sobre a própria faixa (fora dela, os
			// painéis são mais profundos): hover = ponteiro na pega.
			s.hover = true
			s.damage(s.splitterRect())
			return true
		case event.MouseLeave:
			s.hover = false
			s.damage(s.splitterRect())
			return true
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft || !e.Pos.In(s.splitterRect()) {
				return false
			}
			s.dragging = true
			return true
		case event.MouseMove:
			if !s.dragging {
				return false
			}
			s.setRatio(s.ratioAt(e.Pos))
			return true
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft || !s.dragging {
				return false
			}
			s.dragging = false
			return true
		}
	}
	return false
}

// handleKey move o divisor pelo teclado: setas do eixo em passos, Home/End
// aos extremos (os mínimos seguem valendo no layout).
func (s *SplitPane) handleKey(k event.Key) bool {
	avail := s.axisSize() - s.thicknessPx()
	if avail <= 0 {
		return false
	}
	step := float64(s.theme.Px(16)) / float64(avail)
	var dec, inc event.Key = event.KeyLeft, event.KeyRight
	if s.vertical {
		dec, inc = event.KeyUp, event.KeyDown
	}
	switch k {
	case dec:
		s.setRatio(s.ratio - step)
	case inc:
		s.setRatio(s.ratio + step)
	case event.KeyHome:
		s.setRatio(0)
	case event.KeyEnd:
		s.setRatio(1)
	default:
		return false
	}
	return true
}

// Draw desenha os painéis e o divisor: uma linha fina centrada na faixa —
// InputBorder em repouso, Accent sob o ponteiro, no arraste ou focado (com
// anel de foco na faixa).
func (s *SplitPane) Draw(dst *image.RGBA) {
	if s.theme == nil {
		return
	}
	if s.a.Bounds().Overlaps(dst.Bounds()) {
		s.a.Draw(dst)
	}
	if s.b.Bounds().Overlaps(dst.Bounds()) {
		s.b.Draw(dst)
	}
	th := s.theme
	sp := s.splitterRect()
	line := max(th.BorderPx(), 1)
	c := th.InputBorder
	if s.hover || s.dragging || s.focused {
		c = th.Accent
	}
	var lr image.Rectangle
	if s.vertical {
		mid := sp.Min.Y + (sp.Dy()-line)/2
		lr = image.Rect(sp.Min.X, mid, sp.Max.X, mid+line)
	} else {
		mid := sp.Min.X + (sp.Dx()-line)/2
		lr = image.Rect(mid, sp.Min.Y, mid+line, sp.Max.Y)
	}
	render.FillRect(dst, lr, c)
	if s.focused {
		render.StrokeRoundRect(dst, sp, th.RadiusPx(), th.BorderPx(), th.FocusOutline)
	}
	s.drawDisabledOverlay(dst)
}
