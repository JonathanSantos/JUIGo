package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/theme"
)

// Sized fixa o tamanho PREFERIDO do filho em unidades lógicas — o jeito de
// dar dimensões a conteúdos que medem demais (uma lista virtualizada dentro
// de um Modal) ou de menos. Zero em um eixo mantém a medida natural do
// filho nesse eixo. O filho ocupa a área inteira que o layout conceder.
//
//	juigo.NewSized(juigo.NewScroll(lista), 340, 220)
type Sized struct {
	BaseWidget
	child Widget
	w, h  int
	// kid é o scratch devolvido por Children (sem alocação por chamada).
	kid [1]Widget
}

// NewSized envolve child com o tamanho preferido dado, em unidades lógicas
// (zero em um eixo herda a medida do filho). O tema é herdado no mount.
func NewSized(child Widget, w, h int) *Sized {
	return &Sized{child: child, w: w, h: h}
}

// Children devolve o filho.
func (s *Sized) Children() []Widget {
	s.kid[0] = s.child
	return s.kid[:]
}

// SetTheme define um tema explícito e o propaga ao filho, como os
// containers.
func (s *Sized) SetTheme(th *theme.Theme) {
	s.BaseWidget.SetTheme(th)
	Mount(s, th)
}

// Layout entrega ao filho a área inteira.
func (s *Sized) Layout(bounds image.Rectangle) {
	s.BaseWidget.Layout(bounds)
	s.child.Layout(bounds)
}

// PreferredSize devolve o tamanho fixado (em pixels), com a medida do filho
// nos eixos deixados em zero.
func (s *Sized) PreferredSize() image.Point {
	if s.theme == nil {
		return s.child.PreferredSize()
	}
	p := s.child.PreferredSize()
	if s.w > 0 {
		p.X = s.theme.Px(s.w)
	}
	if s.h > 0 {
		p.Y = s.theme.Px(s.h)
	}
	return p
}

// Draw desenha o filho.
func (s *Sized) Draw(dst *image.RGBA) {
	if s.child.Bounds().Overlaps(dst.Bounds()) {
		s.child.Draw(dst)
	}
	s.drawDisabledOverlay(dst)
}
