package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// Checkbox é uma caixa de marcação com rótulo. A semântica de acionamento é
// a mesma do Button: event.MouseDown dentro arma, event.MouseUp dentro alterna,
// event.MouseLeave pressionado cancela. É focável; Enter ou Espaço alternam quando
// focado.
//
// Nasce pronto para reatividade: BindChecked vincula o valor a um
// State[bool] em duas vias.
type Checkbox struct {
	BaseWidget
	// Label é o texto exibido à direita da caixa.
	label string

	// onChange é chamado com o novo valor a cada alternância feita pelo
	// usuário (ver OnChange).
	onChange func(bool)
	checked  bool
	pressed  bool
	hover    bool
	focused  bool
	bound    *state.State[bool]
}

// NewCheckbox cria um checkbox desmarcado com o rótulo dado. O tema é
// herdado no mount.
func NewCheckbox(label string) *Checkbox {
	return &Checkbox{label: label}
}

// OnChange define o callback chamado com o novo valor a cada alternância
// feita pelo usuário. Encadeável.
func (c *Checkbox) OnChange(fn func(bool)) *Checkbox {
	c.onChange = fn
	return c
}

// Label devolve o rótulo da caixa.
func (c *Checkbox) Label() string {
	return c.label
}

// SetLabel troca o rótulo e agenda um redesenho. Encadeável.
func (c *Checkbox) SetLabel(s string) *Checkbox {
	if c.label != s {
		c.label = s
		c.Invalidate()
	}
	return c
}

// Checked devolve o valor atual.
func (c *Checkbox) Checked() bool {
	return c.checked
}

// SetChecked define o valor programaticamente, sincroniza o State vinculado
// (se houver) e agenda um redesenho. Não dispara OnChange. Encadeável.
func (c *Checkbox) SetChecked(v bool) *Checkbox {
	if c.checked != v {
		c.checked = v
		c.Invalidate()
	}
	if c.bound != nil && c.bound.Get() != v {
		c.bound.Set(v)
	}
	return c
}

// BindChecked vincula o valor ao State em DUAS vias: alternâncias do usuário
// fazem Set no State, e um Set externo atualiza a caixa. Encadeável.
func (c *Checkbox) BindChecked(s *state.State[bool]) *Checkbox {
	c.bound = s
	c.SetChecked(s.Get())
	state.WatchWeak(s, c, func(c *Checkbox, v bool) {
		if c.checked != v {
			c.SetChecked(v)
		}
	})
	return c
}

// Focusable devolve true: o checkbox participa da cadeia de foco.
func (c *Checkbox) Focusable() bool {
	return true
}

// PreferredSize devolve a caixa (um quadrado de uma linha de altura) mais o
// rótulo. Antes do mount (sem tema), devolve zero.
func (c *Checkbox) PreferredSize() image.Point {
	if c.theme == nil {
		return image.Point{}
	}
	box := c.theme.LineHeight()
	return image.Point{
		X: box + c.theme.PaddingPx() + c.theme.MeasureString(c.label),
		Y: box,
	}
}

// HandleEvent implementa o acionamento por mouse e teclado e o registro de
// foco.
func (c *Checkbox) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEnter || e.Key == event.KeySpace {
			c.toggle()
			return true
		}
		return false
	case event.FocusEvent:
		c.focused = e.Gained
		c.notifyFocus(e.Gained)
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseEnter:
			c.hover = true
			return true
		case event.MouseLeave:
			// Sair com o botão pressionado cancela sem alternar.
			c.hover = false
			c.pressed = false
			return true
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			c.pressed = true
			return true
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft || !c.pressed {
				return false
			}
			c.pressed = false
			if e.Pos.In(c.Bounds()) {
				c.toggle()
			}
			return true
		}
	}
	return false
}

// toggle alterna o valor por ação do usuário, propagando ao State vinculado
// e ao OnChange.
func (c *Checkbox) toggle() {
	c.checked = !c.checked
	if c.bound != nil && c.bound.Get() != c.checked {
		c.bound.Set(c.checked)
	}
	if c.onChange != nil {
		c.onChange(c.checked)
	}
}

// Draw desenha a caixa (com a marca quando marcado) e o rótulo.
func (c *Checkbox) Draw(dst *image.RGBA) {
	if c.theme == nil {
		return
	}
	th := c.theme
	bounds := c.Bounds()

	side := th.LineHeight()
	boxTop := bounds.Min.Y + (bounds.Dy()-side)/2
	box := image.Rect(bounds.Min.X, boxTop, bounds.Min.X+side, boxTop+side)

	// O raio segue o tema mas é PRESO a 1/4 do lado: um tema de cantos
	// generosos (papel e tinta, raio 10) não pode fazer a caixa virar
	// círculo — checkbox circular é indistinguível de Radio.
	radius := min(th.RadiusPx(), side/4)
	render.FillRoundRect(dst, box, radius, th.InputBackground)
	border := th.InputBorder
	if c.hover || c.pressed {
		border = th.InputBorderFocused
	}
	render.StrokeRoundRect(dst, box, radius, th.BorderPx(), border)
	if c.focused {
		render.StrokeRoundRect(dst, box, radius, 2*th.BorderPx(), th.FocusOutline)
	}

	if c.checked {
		// A marca interna usa metade do raio: acompanha a caixa sem virar
		// círculo em caixas pequenas.
		inset := side / 4
		render.FillRoundRect(dst, box.Inset(inset), radius/2, th.Accent)
	}

	labelX := box.Max.X + th.PaddingPx()
	baseline := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(dst, c.label, image.Pt(labelX, baseline), th.Text)
	c.drawDisabledOverlay(dst)
}
