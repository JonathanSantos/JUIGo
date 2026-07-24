package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Card agrupa conteúdo numa superfície elevada: fundo Theme.Surface, fio
// Theme.SurfaceBorder e os cantos do tema — a forma padrão de destacar um
// bloco do papel de fundo no design system.
//
//	juigo.NewCard(juigo.NewVBox(
//	    juigo.NewText("Fatura").Title(),
//	    juigo.NewText("Vence em 3 dias").Caption(),
//	))
type Card struct {
	BaseWidget
	child Widget
	// padding é o respiro interno em unidades lógicas; negativo usa o
	// dobro do Theme.Padding (cartões pedem mais ar que controles).
	padding int
	// kid é o scratch devolvido por Children (sem alocação por chamada).
	kid [1]Widget
}

// NewCard cria um cartão em volta de child. O tema é herdado no mount.
func NewCard(child Widget) *Card {
	return &Card{child: child, padding: -1}
}

// Pad define o respiro interno em unidades lógicas; negativo volta ao
// padrão (2 × Theme.Padding). Encadeável.
func (c *Card) Pad(l int) *Card {
	c.padding = l
	c.Invalidate()
	return c
}

// padPx resolve o respiro interno em pixels.
func (c *Card) padPx() int {
	if c.theme == nil {
		return 0
	}
	if c.padding >= 0 {
		return c.theme.Px(c.padding)
	}
	return 2 * c.theme.PaddingPx()
}

// Children devolve o conteúdo.
func (c *Card) Children() []Widget {
	c.kid[0] = c.child
	return c.kid[:]
}

// SetTheme define um tema explícito e o propaga ao conteúdo, como os
// containers.
func (c *Card) SetTheme(th *theme.Theme) {
	c.BaseWidget.SetTheme(th)
	Mount(c, th)
}

// Layout posiciona o conteúdo com o respiro interno.
func (c *Card) Layout(bounds image.Rectangle) {
	c.BaseWidget.Layout(bounds)
	pad := c.padPx()
	c.child.Layout(bounds.Inset(pad))
}

// PreferredSize devolve o tamanho do conteúdo mais o respiro.
func (c *Card) PreferredSize() image.Point {
	pad := c.padPx()
	p := c.child.PreferredSize()
	return image.Pt(p.X+2*pad, p.Y+2*pad)
}

// Draw desenha a superfície, o fio e o conteúdo.
func (c *Card) Draw(dst *image.RGBA) {
	if c.theme == nil {
		return
	}
	b := c.Bounds()
	radius := c.theme.RadiusPx()
	render.FillRoundRect(dst, b, radius, c.theme.Surface)
	if c.theme.SurfaceBorder.A > 0 {
		render.StrokeRoundRect(dst, b, radius, c.theme.BorderPx(), c.theme.SurfaceBorder)
	}
	if c.child.Bounds().Overlaps(dst.Bounds()) {
		c.child.Draw(dst)
	}
	c.drawDisabledOverlay(dst)
}
