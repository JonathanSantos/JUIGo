package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/render"
)

// Divider é o fio horizontal separador do design system (Theme.
// SurfaceBorder): ocupa uma faixa com a altura do Spacing do tema, com a
// linha centrada nela — o respiro já vem embutido.
type Divider struct {
	BaseWidget
}

// NewDivider cria um separador horizontal. O tema é herdado no mount.
func NewDivider() *Divider {
	return &Divider{}
}

// PreferredSize devolve a faixa do separador (altura = Spacing do tema).
func (d *Divider) PreferredSize() image.Point {
	if d.theme == nil {
		return image.Point{}
	}
	return image.Pt(0, d.theme.SpacingPx())
}

// Draw desenha a linha centrada na faixa.
func (d *Divider) Draw(dst *image.RGBA) {
	if d.theme == nil {
		return
	}
	b := d.Bounds()
	line := max(d.theme.BorderPx(), 1)
	y := b.Min.Y + (b.Dy()-line)/2
	render.FillRect(dst, image.Rect(b.Min.X, y, b.Max.X, y+line), d.theme.SurfaceBorder)
}
