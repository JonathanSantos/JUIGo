package juigo

import (
	"image"

	"juigo/render"
)

// ButtonState identifica o estado visual do Button.
type ButtonState int

const (
	// ButtonStateNormal é o estado de repouso.
	ButtonStateNormal ButtonState = iota
	// ButtonStateHover indica o cursor sobre o botão.
	ButtonStateHover
	// ButtonStatePressed indica o botão pressionado (mouse down dentro).
	ButtonStatePressed
)

// Button é um botão de ação com rótulo. Semântica de clique:
//   - MouseDown dentro: passa a pressed.
//   - MouseUp dentro E pressed: dispara OnClick.
//   - MouseLeave enquanto pressed: cancela sem disparar.
//
// É focável; Enter ou Espaço disparam OnClick quando focado.
type Button struct {
	BaseWidget
	// Label é o texto do botão.
	Label string
	// Padding é o espaço interno entre o rótulo e as bordas.
	Padding int
	// OnClick é chamado quando o botão é acionado. Pode ser nil.
	OnClick func()

	theme   *Theme
	state   ButtonState
	pressed bool
	focused bool
}

// NewButton cria um botão com o tema, rótulo e callback dados. O padding
// padrão vem do tema.
func NewButton(theme *Theme, label string, onClick func()) *Button {
	return &Button{
		Label:   label,
		Padding: theme.Padding,
		OnClick: onClick,
		theme:   theme,
	}
}

// State devolve o estado visual atual do botão.
func (b *Button) State() ButtonState {
	return b.state
}

// Focusable devolve true: o botão participa da cadeia de foco.
func (b *Button) Focusable() bool {
	return true
}

// PreferredSize devolve o tamanho do rótulo mais o padding interno.
func (b *Button) PreferredSize() image.Point {
	return image.Point{
		X: b.theme.MeasureString(b.Label) + 2*b.Padding,
		Y: b.theme.LineHeight() + 2*b.Padding,
	}
}

// Draw desenha o fundo conforme o estado, o contorno de foco quando focado e
// o rótulo centralizado.
func (b *Button) Draw(dst *image.RGBA) {
	bounds := b.Bounds()

	bg := b.theme.ButtonNormal
	switch b.state {
	case ButtonStateHover:
		bg = b.theme.ButtonHover
	case ButtonStatePressed:
		bg = b.theme.ButtonPressed
	}
	render.FillRect(dst, bounds, bg)

	if b.focused {
		render.StrokeRect(dst, bounds, 2*b.theme.BorderWidth, b.theme.FocusOutline)
	}

	labelW := b.theme.MeasureString(b.Label)
	x := bounds.Min.X + (bounds.Dx()-labelW)/2
	y := bounds.Min.Y + (bounds.Dy()-b.theme.LineHeight())/2 + b.theme.Ascent()
	render.DrawText(dst, b.theme.Face, b.Label, image.Pt(x, y), b.theme.ButtonText)
}
