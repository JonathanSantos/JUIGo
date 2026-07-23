package widget

import (
	"image"

	"juigo/render"
	"juigo/state"
)

// ProgressBar é o indicador só-leitura de progresso no intervalo [Min, Max]
// — o irmão passivo do Slider. Não é focável e não consome eventos.
//
// Nasce pronto para reatividade: BindValue espelha um State[float64].
type ProgressBar struct {
	BaseWidget
	// Min e Max delimitam o intervalo (Min <= Max).
	Min, Max float64

	value float64
}

// NewProgressBar cria uma barra com o intervalo dado e o valor em min.
// O tema é herdado no mount.
func NewProgressBar(min, max float64) *ProgressBar {
	if max < min {
		min, max = max, min
	}
	return &ProgressBar{Min: min, Max: max, value: min}
}

// Value devolve o valor atual.
func (p *ProgressBar) Value() float64 {
	return p.value
}

// SetValue define o valor (limitado ao intervalo) e agenda um redesenho.
func (p *ProgressBar) SetValue(v float64) {
	if v < p.Min {
		v = p.Min
	}
	if v > p.Max {
		v = p.Max
	}
	if p.value == v {
		return
	}
	p.value = v
	p.Invalidate()
}

// BindValue espelha o State na barra (uma via: a barra só exibe).
// Encadeável.
func (p *ProgressBar) BindValue(s *state.State[float64]) *ProgressBar {
	p.SetValue(s.Get())
	s.Watch(func(v float64) {
		p.SetValue(v)
	})
	return p
}

// PreferredSize devolve a largura mínima do Slider e o dobro da espessura do
// trilho. Antes do mount (sem tema), devolve zero.
func (p *ProgressBar) PreferredSize() image.Point {
	if p.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: p.theme.Px(p.theme.SliderMinWidth),
		Y: p.theme.Px(2 * p.theme.SliderTrackThickness),
	}
}

// Draw desenha o trilho e a fração preenchida na cor de destaque.
func (p *ProgressBar) Draw(dst *image.RGBA) {
	if p.theme == nil {
		return
	}
	th := p.theme
	b := p.Bounds()
	render.FillRect(dst, b, th.InputBorder)
	span := p.Max - p.Min
	if span <= 0 {
		return
	}
	fill := int(float64(b.Dx()) * (p.value - p.Min) / span)
	render.FillRect(dst, image.Rect(b.Min.X, b.Min.Y, b.Min.X+fill, b.Max.Y), th.Accent)
}
