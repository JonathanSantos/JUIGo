package juigo

import (
	"image"

	"juigo/render"
)

// Slider é um controle deslizante horizontal para escolher um float64 no
// intervalo [Min, Max]. Clicar posiciona o valor; arrastar o ajusta
// continuamente — inclusive com o cursor fora dos bounds, graças à captura
// de mouse do App. É focável; setas ajustam por Step, Home/End vão aos
// extremos.
//
// Nasce pronto para reatividade: BindValue vincula o valor a um
// State[float64] em duas vias.
type Slider struct {
	BaseWidget
	// Min e Max delimitam o intervalo de valores (Min <= Max).
	Min, Max float64
	// Step é o incremento das setas do teclado; zero usa 5% do intervalo.
	Step float64
	// OnChange é chamado com o novo valor a cada ajuste feito pelo usuário.
	// Pode ser nil.
	OnChange func(float64)

	value    float64
	dragging bool
	hover    bool
	focused  bool
	bound    *State[float64]
}

// NewSlider cria um slider com o intervalo dado e o valor inicial em min.
// O tema é herdado no mount.
func NewSlider(min, max float64) *Slider {
	if max < min {
		min, max = max, min
	}
	return &Slider{Min: min, Max: max, value: min}
}

// Value devolve o valor atual.
func (s *Slider) Value() float64 {
	return s.value
}

// SetValue define o valor programaticamente (limitado ao intervalo) e agenda
// um redesenho. Não dispara OnChange; com binding, prefira State.Set.
func (s *Slider) SetValue(v float64) {
	v = s.clamp(v)
	if s.value == v {
		return
	}
	s.value = v
	requestRepaint()
}

// BindValue vincula o valor ao State em DUAS vias: ajustes do usuário fazem
// Set no State, e um Set externo move o slider. Encadeável.
func (s *Slider) BindValue(st *State[float64]) *Slider {
	s.bound = st
	s.SetValue(st.Get())
	st.Watch(func(v float64) {
		if s.value != s.clamp(v) {
			s.SetValue(v)
		}
	})
	return s
}

// Focusable devolve true: o slider participa da cadeia de foco.
func (s *Slider) Focusable() bool {
	return true
}

// PreferredSize devolve a largura mínima do tema e a altura da alça. Antes
// do mount (sem tema), devolve zero.
func (s *Slider) PreferredSize() image.Point {
	if s.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: s.theme.Px(s.theme.SliderMinWidth),
		Y: s.theme.Px(s.theme.SliderHandleSize) + 4*s.theme.BorderPx(),
	}
}

// HandleEvent implementa clique/arraste (com captura), teclado e foco.
func (s *Slider) HandleEvent(ev Event) bool {
	switch e := ev.(type) {
	case KeyEvent:
		return s.handleKey(e.Key)
	case FocusEvent:
		s.focused = e.Gained
		return true
	case MouseEvent:
		switch e.Kind {
		case MouseEnter:
			s.hover = true
			return true
		case MouseLeave:
			// O arraste continua fora dos bounds (captura); só o realce sai.
			s.hover = false
			return true
		case MouseDown:
			if e.Button != MouseButtonLeft {
				return false
			}
			s.dragging = true
			s.setFromUser(s.valueAt(e.Pos.X))
			return true
		case MouseMove:
			if !s.dragging {
				return false
			}
			return s.setFromUser(s.valueAt(e.Pos.X))
		case MouseUp:
			if e.Button != MouseButtonLeft || !s.dragging {
				return false
			}
			s.dragging = false
			return true
		}
	}
	return false
}

// handleKey ajusta o valor pelo teclado. Devolve true se algo mudou.
func (s *Slider) handleKey(k Key) bool {
	step := s.Step
	if step <= 0 {
		step = (s.Max - s.Min) / 20
	}
	switch k {
	case KeyLeft:
		return s.setFromUser(s.value - step)
	case KeyRight:
		return s.setFromUser(s.value + step)
	case KeyHome:
		return s.setFromUser(s.Min)
	case KeyEnd:
		return s.setFromUser(s.Max)
	}
	return false
}

// setFromUser aplica um ajuste feito pelo usuário, propagando ao State
// vinculado e ao OnChange. Devolve true se o valor mudou.
func (s *Slider) setFromUser(v float64) bool {
	v = s.clamp(v)
	if s.value == v {
		return false
	}
	s.value = v
	if s.bound != nil && s.bound.Get() != v {
		s.bound.Set(v)
	}
	if s.OnChange != nil {
		s.OnChange(v)
	}
	return true
}

// clamp limita v ao intervalo [Min, Max].
func (s *Slider) clamp(v float64) float64 {
	if v < s.Min {
		return s.Min
	}
	if v > s.Max {
		return s.Max
	}
	return v
}

// ratio devolve a posição relativa do valor no intervalo, em [0, 1].
func (s *Slider) ratio() float64 {
	span := s.Max - s.Min
	if span <= 0 {
		return 0
	}
	return (s.value - s.Min) / span
}

// trackSpan devolve os X inicial e final do curso da alça (centro da alça
// nos extremos), descontando meia alça de cada lado.
func (s *Slider) trackSpan() (x0, x1 int) {
	half := s.theme.Px(s.theme.SliderHandleSize) / 2
	b := s.Bounds()
	return b.Min.X + half, b.Max.X - half
}

// valueAt converte uma coordenada X absoluta no valor correspondente.
func (s *Slider) valueAt(x int) float64 {
	if s.theme == nil {
		return s.value
	}
	x0, x1 := s.trackSpan()
	if x1 <= x0 {
		return s.Min
	}
	rel := float64(x-x0) / float64(x1-x0)
	return s.Min + rel*(s.Max-s.Min)
}

// Draw desenha o trilho (ativo até a alça, na cor de destaque) e a alça.
func (s *Slider) Draw(dst *image.RGBA) {
	if s.theme == nil {
		return
	}
	th := s.theme
	b := s.Bounds()

	x0, x1 := s.trackSpan()
	cy := b.Min.Y + b.Dy()/2
	half := th.Px(th.SliderTrackThickness) / 2
	if half < 1 {
		half = 1
	}
	hx := x0 + int(s.ratio()*float64(x1-x0))

	render.FillRect(dst, image.Rect(x0, cy-half, hx, cy+half), th.Accent)
	render.FillRect(dst, image.Rect(hx, cy-half, x1, cy+half), th.InputBorder)

	handle := th.Px(th.SliderHandleSize)
	hr := image.Rect(hx-handle/2, cy-handle/2, hx+handle/2, cy+handle/2)
	fill := th.Accent
	switch {
	case s.dragging:
		fill = th.ButtonPressed
	case s.hover:
		fill = th.ButtonHover
	}
	render.FillRect(dst, hr, fill)
	if s.focused {
		render.StrokeRect(dst, hr, 2*th.BorderPx(), th.FocusOutline)
	}
}
