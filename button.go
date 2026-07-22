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
	// Padding é o espaço interno entre o rótulo e as bordas, em unidades
	// lógicas (convertido pela escala do tema). Negativo usa o padrão do
	// tema (Theme.Padding).
	Padding int
	// OnClick é chamado quando o botão é acionado. Pode ser nil.
	OnClick func()

	state   ButtonState
	pressed bool
	focused bool
}

// NewButton cria um botão com o rótulo e o callback dados. O tema é herdado
// no mount; o padding padrão vem do tema.
func NewButton(label string, onClick func()) *Button {
	return &Button{
		Label:   label,
		Padding: -1,
		OnClick: onClick,
	}
}

// padPx resolve o padding para pixels na escala do tema.
func (b *Button) padPx() int {
	if b.theme == nil {
		return 0
	}
	if b.Padding >= 0 {
		return b.theme.Px(b.Padding)
	}
	return b.theme.PaddingPx()
}

// State devolve o estado visual atual do botão.
func (b *Button) State() ButtonState {
	return b.state
}

// HandleEvent implementa a máquina de estados do clique, o acionamento por
// teclado (Enter/Espaço quando focado) e o registro de foco.
func (b *Button) HandleEvent(ev Event) bool {
	switch e := ev.(type) {
	case KeyEvent:
		if e.Key == KeyEnter || e.Key == KeySpace {
			b.fire()
			return true
		}
		return false
	case FocusEvent:
		b.focused = e.Gained
		return true
	case MouseEvent:
		return b.handleMouse(e)
	}
	return false
}

// handleMouse trata a parte de mouse da máquina de estados.
func (b *Button) handleMouse(e MouseEvent) bool {
	switch e.Kind {
	case MouseEnter:
		if b.state == ButtonStateNormal {
			b.state = ButtonStateHover
			return true
		}
	case MouseLeave:
		// Sair com o botão pressionado cancela o clique sem disparar.
		b.pressed = false
		if b.state != ButtonStateNormal {
			b.state = ButtonStateNormal
			return true
		}
	case MouseDown:
		if e.Button != MouseButtonLeft {
			return false
		}
		b.pressed = true
		b.state = ButtonStatePressed
		return true
	case MouseUp:
		if e.Button != MouseButtonLeft || !b.pressed {
			return false
		}
		b.pressed = false
		if e.Pos.In(b.Bounds()) {
			b.state = ButtonStateHover
			b.fire()
		} else {
			b.state = ButtonStateNormal
		}
		return true
	}
	return false
}

// fire dispara o callback OnClick, se houver.
func (b *Button) fire() {
	if b.OnClick != nil {
		b.OnClick()
	}
}

// Focusable devolve true: o botão participa da cadeia de foco.
func (b *Button) Focusable() bool {
	return true
}

// PreferredSize devolve o tamanho do rótulo mais o padding interno. Antes do
// mount (sem tema), devolve zero.
func (b *Button) PreferredSize() image.Point {
	if b.theme == nil {
		return image.Point{}
	}
	pad := b.padPx()
	return image.Point{
		X: b.theme.MeasureString(b.Label) + 2*pad,
		Y: b.theme.LineHeight() + 2*pad,
	}
}

// Draw desenha o fundo conforme o estado, o contorno de foco quando focado e
// o rótulo centralizado.
func (b *Button) Draw(dst *image.RGBA) {
	if b.theme == nil {
		return
	}
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
		render.StrokeRect(dst, bounds, 2*b.theme.BorderPx(), b.theme.FocusOutline)
	}

	labelW := b.theme.MeasureString(b.Label)
	x := bounds.Min.X + (bounds.Dx()-labelW)/2
	y := bounds.Min.Y + (bounds.Dy()-b.theme.LineHeight())/2 + b.theme.Ascent()
	b.theme.DrawText(dst, b.Label, image.Pt(x, y), b.theme.ButtonText)
}
