package widget

import (
	"image"

	"juigo/event"
	"juigo/render"
	"juigo/theme"
)

// Scroll é um container de rolagem VERTICAL com um único filho: o filho é
// medido na altura preferida e desenhado RECORTADO à área visível; a roda do
// mouse (ou trackpad, com deltas fracionários acumulados) rola o conteúdo.
// No limite da rolagem o evento não é consumido e propaga para cima — Scrolls
// aninhados funcionam naturalmente.
//
// O tamanho preferido é o do filho; para dar ao Scroll uma altura de
// viewport menor que o conteúdo, combine com Grow:
//
//	juigo.NewVBox(cabecalho, juigo.Grow(juigo.NewScroll(lista), 1))
//
// Um indicador fino de posição é desenhado à direita quando o conteúdo é
// maior que a área visível.
type Scroll struct {
	BaseWidget

	child  Widget
	offset int     // deslocamento vertical do conteúdo, em pixels
	accum  float64 // resto fracionário de rolagens de trackpad
	clip   image.RGBA
}

// NewScroll cria um Scroll para o filho dado. O tema é herdado no mount.
func NewScroll(child Widget) *Scroll {
	return &Scroll{child: child}
}

// Children devolve o filho (satisfaz ParentWidget: roteamento, mount, foco).
func (s *Scroll) Children() []Widget {
	if s.child == nil {
		return nil
	}
	return []Widget{s.child}
}

// SetTheme define um tema explícito e o propaga imediatamente à subárvore,
// como nos demais containers.
func (s *Scroll) SetTheme(t *theme.Theme) {
	s.BaseWidget.SetTheme(t)
	Mount(s, t)
}

// Offset devolve o deslocamento atual da rolagem, em pixels (0 = topo).
func (s *Scroll) Offset() int {
	return s.offset
}

// ScrollTo define o deslocamento da rolagem, em pixels (limitado ao
// intervalo válido no próximo layout), e agenda um redesenho.
func (s *Scroll) ScrollTo(y int) {
	s.offset = y
	s.Invalidate()
}

// PreferredSize devolve o tamanho preferido do filho — normalmente o Scroll
// é combinado com Grow para receber a altura da viewport.
func (s *Scroll) PreferredSize() image.Point {
	if s.child == nil {
		return image.Point{}
	}
	return s.child.PreferredSize()
}

// Layout dá ao filho a largura da viewport e a própria altura preferida (no
// mínimo a da viewport), deslocado pela rolagem — que é limitada aqui ao
// intervalo válido.
func (s *Scroll) Layout(bounds image.Rectangle) {
	s.BaseWidget.Layout(bounds)
	if s.child == nil {
		return
	}
	contentH := s.child.PreferredSize().Y
	if contentH < bounds.Dy() {
		contentH = bounds.Dy()
	}
	if max := contentH - bounds.Dy(); s.offset > max {
		s.offset = max
	}
	if s.offset < 0 {
		s.offset = 0
	}
	top := bounds.Min.Y - s.offset
	s.child.Layout(image.Rect(bounds.Min.X, top, bounds.Max.X, top+contentH))
}

// HandleEvent consome ScrollEvent quando há para onde rolar; no limite,
// devolve false para o evento propagar (Scrolls aninhados).
func (s *Scroll) HandleEvent(ev event.Event) bool {
	e, ok := ev.(event.ScrollEvent)
	if !ok || s.child == nil || s.theme == nil {
		return false
	}
	max := s.child.Bounds().Dy() - s.Bounds().Dy()
	if max <= 0 {
		return false
	}

	// Acumula deltas fracionários (trackpad) até renderem pixels inteiros.
	delta := e.DY*float64(s.theme.Px(s.theme.ScrollStep)) + s.accum
	step := int(delta)
	s.accum = delta - float64(step)

	novo := s.offset - step
	if novo < 0 {
		novo = 0
	}
	if novo > max {
		novo = max
	}
	if novo == s.offset {
		if step != 0 {
			// Já estava no limite nesta direção: propaga para o ancestral.
			s.accum = 0
			return false
		}
		return true // delta minúsculo: consumido, aguardando acumular
	}
	s.offset = novo
	return true
}

// Draw desenha o filho recortado à área visível e o indicador de posição.
func (s *Scroll) Draw(dst *image.RGBA) {
	if s.child == nil || s.theme == nil {
		return
	}
	th := s.theme
	bounds := s.Bounds()
	view := render.Clip(dst, bounds, &s.clip)
	s.child.Draw(view)

	// Indicador: só quando o conteúdo excede a viewport.
	contentH := s.child.Bounds().Dy()
	viewportH := bounds.Dy()
	if contentH <= viewportH {
		return
	}
	thumbH := viewportH * viewportH / contentH
	if min := th.Px(4 * th.ScrollbarWidth); thumbH < min {
		thumbH = min
	}
	maxOff := contentH - viewportH
	thumbY := bounds.Min.Y + (viewportH-thumbH)*s.offset/maxOff
	w := th.Px(th.ScrollbarWidth)
	margin := 2 * th.BorderPx()
	render.FillRect(view, image.Rect(bounds.Max.X-w-margin, thumbY, bounds.Max.X-margin, thumbY+thumbH), th.Placeholder)
}
