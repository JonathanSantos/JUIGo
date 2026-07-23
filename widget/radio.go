package widget

import (
	"image"

	"juigo/event"
	"juigo/render"
	"juigo/state"
)

// Radio é uma opção de escolha exclusiva. NÃO existe um "RadioGroup": o
// grupo É o State — vincule vários rádios ao mesmo State[string] com
// BindValue e a exclusividade emerge da reatividade (selecionar um faz Set
// no State; os watchers desmarcam os demais):
//
//	plano := juigo.NewState("pro")
//	juigo.NewRadio("Grátis", "free").BindValue(plano)
//	juigo.NewRadio("Pro", "pro").BindValue(plano)
//
// A semântica de acionamento é a do Button (leave pressionado cancela);
// Enter/Espaço selecionam quando focado. Selecionar um rádio já marcado não
// faz nada (rádios não desmarcam sozinhos).
type Radio struct {
	BaseWidget
	// Label é o texto à direita do círculo.
	Label string
	// Value é o valor que este rádio representa no grupo.
	Value string
	// OnChange é chamado com Value quando ESTE rádio é selecionado pelo
	// usuário. Pode ser nil.
	OnChange func(string)

	checked bool
	pressed bool
	hover   bool
	focused bool
	bound   *state.State[string]
}

// NewRadio cria um rádio desmarcado com o rótulo e o valor dados. O tema é
// herdado no mount.
func NewRadio(label, value string) *Radio {
	return &Radio{Label: label, Value: value}
}

// Checked devolve se este rádio é o selecionado do grupo.
func (r *Radio) Checked() bool {
	return r.checked
}

// BindValue vincula o rádio ao State do grupo: ele fica marcado quando o
// valor do State é igual ao seu Value, e selecioná-lo faz Set. Encadeável.
func (r *Radio) BindValue(s *state.State[string]) *Radio {
	r.bound = s
	r.checked = s.Get() == r.Value
	s.Watch(func(v string) {
		if want := v == r.Value; want != r.checked {
			r.checked = want
		}
	})
	return r
}

// Focusable devolve true: o rádio participa da cadeia de foco.
func (r *Radio) Focusable() bool {
	return true
}

// CursorShape do Radio: mãozinha.
func (r *Radio) CursorShape() CursorShape {
	return CursorHand
}

// PreferredSize devolve o círculo (uma linha de altura) mais o rótulo.
func (r *Radio) PreferredSize() image.Point {
	if r.theme == nil {
		return image.Point{}
	}
	side := r.theme.LineHeight()
	return image.Point{
		X: side + r.theme.PaddingPx() + r.theme.MeasureString(r.Label),
		Y: side,
	}
}

// HandleEvent implementa o acionamento por mouse e teclado e o foco.
func (r *Radio) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEnter || e.Key == event.KeySpace {
			r.selectFromUser()
			return true
		}
		return false
	case event.FocusEvent:
		r.focused = e.Gained
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseEnter:
			r.hover = true
			return true
		case event.MouseLeave:
			r.hover = false
			r.pressed = false
			return true
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			r.pressed = true
			return true
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft || !r.pressed {
				return false
			}
			r.pressed = false
			if e.Pos.In(r.Bounds()) {
				r.selectFromUser()
			}
			return true
		}
	}
	return false
}

// selectFromUser marca este rádio, propagando ao State do grupo (que
// desmarca os irmãos) e ao OnChange. Já marcado, não faz nada.
func (r *Radio) selectFromUser() {
	if r.checked {
		return
	}
	r.checked = true
	if r.bound != nil && r.bound.Get() != r.Value {
		r.bound.Set(r.Value)
	}
	if r.OnChange != nil {
		r.OnChange(r.Value)
	}
}

// Draw desenha o círculo (anel, com miolo na cor de destaque quando
// marcado) e o rótulo.
func (r *Radio) Draw(dst *image.RGBA) {
	if r.theme == nil {
		return
	}
	th := r.theme
	bounds := r.Bounds()

	side := th.LineHeight()
	radius := side / 2
	center := image.Pt(bounds.Min.X+radius, bounds.Min.Y+bounds.Dy()/2)

	border := th.InputBorder
	if r.hover || r.pressed {
		border = th.InputBorderFocused
	}
	if r.focused {
		border = th.FocusOutline
	}
	render.FillCircle(dst, center, radius, border)
	render.FillCircle(dst, center, radius-th.BorderPx(), th.InputBackground)
	if r.checked {
		render.FillCircle(dst, center, radius-th.BorderPx()-th.Px(3), th.Accent)
	}

	labelX := bounds.Min.X + side + th.PaddingPx()
	baseline := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(dst, r.Label, image.Pt(labelX, baseline), th.Text)
}
