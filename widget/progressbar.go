package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// ProgressBar é o indicador só-leitura de progresso no intervalo [Min, Max]
// — o irmão passivo do Slider. Não é focável e não consome eventos.
//
// Nasce pronto para reatividade: BindValue espelha um State[float64].
type ProgressBar struct {
	BaseWidget
	// min e max delimitam o intervalo (Min <= Max).
	min, max float64

	value float64
}

// NewProgressBar cria uma barra com o intervalo dado e o valor em min.
// O tema é herdado no mount.
func NewProgressBar(min, max float64) *ProgressBar {
	if max < min {
		min, max = max, min
	}
	return &ProgressBar{min: min, max: max, value: min}
}

// Min devolve o limite inferior do intervalo.
func (p *ProgressBar) Min() float64 {
	return p.min
}

// Max devolve o limite superior do intervalo.
func (p *ProgressBar) Max() float64 {
	return p.max
}

// SetRange troca o intervalo (invertido se max < min), reajusta o valor
// atual para caber nele e agenda um redesenho. Não dispara callbacks.
// Encadeável.
func (p *ProgressBar) SetRange(min, max float64) *ProgressBar {
	if max < min {
		min, max = max, min
	}
	p.min, p.max = min, max
	p.SetValue(p.value)
	p.Invalidate()
	return p
}

// Value devolve o valor atual.
func (p *ProgressBar) Value() float64 {
	return p.value
}

// SetValue define o valor (limitado ao intervalo) e agenda um redesenho.
// Encadeável.
func (p *ProgressBar) SetValue(v float64) *ProgressBar {
	if v < p.min {
		v = p.min
	}
	if v > p.max {
		v = p.max
	}
	if p.value == v {
		return p
	}
	p.value = v
	p.Invalidate()
	return p
}

// BindValue espelha o State na barra (uma via: a barra só exibe).
// Encadeável.
func (p *ProgressBar) BindValue(s *state.State[float64]) *ProgressBar {
	p.SetValue(s.Get())
	state.WatchWeak(s, p, func(p *ProgressBar, v float64) { p.SetValue(v) })
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
	radius := th.RadiusPx()
	render.FillRoundRect(dst, b, radius, th.InputBorder)
	span := p.max - p.min
	if span <= 0 {
		return
	}
	fill := int(float64(b.Dx()) * (p.value - p.min) / span)
	render.FillRoundRect(dst, image.Rect(b.Min.X, b.Min.Y, b.Min.X+fill, b.Max.Y), radius, th.Accent)
}
