package widget

import (
	"image"
	"image/color"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
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
//   - event.MouseDown dentro: passa a pressed.
//   - event.MouseUp dentro E pressed: dispara OnClick.
//   - event.MouseLeave enquanto pressed: cancela sem disparar.
//
// É focável; Enter ou Espaço disparam OnClick quando focado.
type Button struct {
	BaseWidget
	// Label é o texto do botão.
	Label string

	// padding é o espaço interno entre o rótulo e as bordas, em unidades
	// lógicas (ver Pad); negativo usa o padrão do tema (Theme.Padding).
	padding int
	// onClick é chamado quando o botão é acionado (ver OnClick).
	onClick func()

	state   ButtonState
	pressed bool
	focused bool
	// clip é a visão recortada reutilizada pelo Draw (sem alocação).
	clip image.RGBA

	// loading exibe o indicador animado no lugar do rótulo e implica
	// desabilitado (sem cliques, fora do Tab) enquanto ativo.
	loading      bool
	spinnerPhase int
	spinnerStop  func()
}

// NewButton cria um botão com o rótulo e o callback dados. O tema é herdado
// no mount; o padding padrão vem do tema.
func NewButton(label string, onClick func()) *Button {
	return &Button{
		Label:   label,
		padding: -1,
		onClick: onClick,
	}
}

// OnClick define a ação do clique, substituindo a passada no construtor.
// Encadeável.
func (b *Button) OnClick(fn func()) *Button {
	b.onClick = fn
	return b
}

// Pad define o espaço interno entre o rótulo e as bordas, em unidades
// lógicas; negativo volta ao padrão do tema. Encadeável.
func (b *Button) Pad(padding int) *Button {
	b.padding = padding
	return b
}

// padPx resolve o padding para pixels na escala do tema.
func (b *Button) padPx() int {
	if b.theme == nil {
		return 0
	}
	if b.padding >= 0 {
		return b.theme.Px(b.padding)
	}
	return b.theme.PaddingPx()
}

// State devolve o estado visual atual do botão.
func (b *Button) State() ButtonState {
	return b.state
}

// Loading informa se o botão está no estado de carregamento.
func (b *Button) Loading() bool {
	return b.loading
}

// SetLoading liga/desliga o estado de carregamento: um indicador animado
// substitui o rótulo e o botão fica efetivamente desabilitado. Combine com
// App.Post para trabalho assíncrono:
//
//	btn.SetLoading(true)
//	go func() {
//	    resultado := buscar()
//	    app.Post(func() { btn.SetLoading(false); usar(resultado) })
//	}()
func (b *Button) SetLoading(v bool) {
	if b.loading == v {
		return
	}
	b.loading = v
	b.stopSpinner()
	if v {
		b.state = ButtonStateNormal
		b.pressed = false
		b.spinnerPhase = 0
		b.scheduleSpinner()
	}
	b.Invalidate()
}

// BindLoading vincula o estado de carregamento ao State. Encadeável.
func (b *Button) BindLoading(s *state.State[bool]) *Button {
	b.SetLoading(s.Get())
	s.Watch(func(v bool) {
		b.SetLoading(v)
	})
	return b
}

// isDisabled: loading implica desabilitado para o roteamento central.
func (b *Button) isDisabled() bool {
	return b.BaseWidget.isDisabled() || b.loading
}

// scheduleSpinner agenda o próximo quadro da animação de carregamento.
func (b *Button) scheduleSpinner() {
	if b.theme == nil || b.theme.SpinnerStep <= 0 {
		return
	}
	b.spinnerStop = hooks.ScheduleAfter(b.theme.SpinnerStep, func() {
		if !b.loading {
			return
		}
		b.spinnerPhase = (b.spinnerPhase + 1) % 3
		b.Invalidate()
		b.scheduleSpinner()
	})
}

// stopSpinner cancela o quadro pendente da animação.
func (b *Button) stopSpinner() {
	if b.spinnerStop != nil {
		b.spinnerStop()
		b.spinnerStop = nil
	}
}

// HandleEvent implementa a máquina de estados do clique, o acionamento por
// teclado (Enter/Espaço quando focado) e o registro de foco.
func (b *Button) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEnter || e.Key == event.KeySpace {
			b.fire()
			return true
		}
		return false
	case event.FocusEvent:
		b.focused = e.Gained
		return true
	case event.MouseEvent:
		return b.handleMouse(e)
	}
	return false
}

// handleMouse trata a parte de mouse da máquina de estados.
func (b *Button) handleMouse(e event.MouseEvent) bool {
	switch e.Kind {
	case event.MouseEnter:
		if b.state == ButtonStateNormal {
			b.state = ButtonStateHover
			return true
		}
	case event.MouseLeave:
		// Sair com o botão pressionado cancela o clique sem disparar.
		b.pressed = false
		if b.state != ButtonStateNormal {
			b.state = ButtonStateNormal
			return true
		}
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft {
			return false
		}
		b.pressed = true
		b.state = ButtonStatePressed
		return true
	case event.MouseUp:
		if e.Button != event.MouseButtonLeft || !b.pressed {
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
	if b.onClick != nil {
		b.onClick()
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
	radius := b.theme.RadiusPx()
	render.FillRoundRect(dst, bounds, radius, bg)
	if b.theme.ButtonBorder.A > 0 {
		render.StrokeRoundRect(dst, bounds, radius, b.theme.BorderPx(), b.theme.ButtonBorder)
	}

	if b.focused {
		render.StrokeRoundRect(dst, bounds, radius, 2*b.theme.BorderPx(), b.theme.FocusOutline)
	}

	view := render.Clip(dst, bounds, &b.clip)
	if b.loading {
		b.drawSpinner(view, bounds)
	} else {
		labelW := b.theme.MeasureString(b.Label)
		x := bounds.Min.X + (bounds.Dx()-labelW)/2
		y := bounds.Min.Y + (bounds.Dy()-b.theme.LineHeight())/2 + b.theme.Ascent()
		b.theme.DrawText(view, b.Label, image.Pt(x, y), b.theme.ButtonText)
	}
	b.drawDisabledOverlay(dst)
}

// drawSpinner desenha os três pontos do indicador de carregamento, com o
// ponto ativo em destaque.
func (b *Button) drawSpinner(dst *image.RGBA, bounds image.Rectangle) {
	th := b.theme
	r := th.Px(3)
	gap := th.Px(4)
	step := 2*r + gap
	cx := bounds.Min.X + bounds.Dx()/2 - step
	cy := bounds.Min.Y + bounds.Dy()/2
	// Pontos inativos: rótulo misturado 50/50 com o fundo do estado
	// (FillCircle é sólido, então a "transparência" é pré-misturada).
	dim := mix(th.ButtonText, b.bgColor())
	for i := 0; i < 3; i++ {
		c := dim
		if i == b.spinnerPhase {
			c = th.ButtonText
		}
		render.FillCircle(dst, image.Pt(cx+i*step, cy), r, c)
	}
}

// bgColor devolve a cor de fundo do estado atual do botão.
func (b *Button) bgColor() color.RGBA {
	switch b.state {
	case ButtonStateHover:
		return b.theme.ButtonHover
	case ButtonStatePressed:
		return b.theme.ButtonPressed
	}
	return b.theme.ButtonNormal
}

// mix devolve a média aritmética de duas cores (mistura 50/50).
func mix(a, bg color.RGBA) color.RGBA {
	return color.RGBA{
		R: uint8((int(a.R) + int(bg.R)) / 2),
		G: uint8((int(a.G) + int(bg.G)) / 2),
		B: uint8((int(a.B) + int(bg.B)) / 2),
		A: 0xFF,
	}
}
