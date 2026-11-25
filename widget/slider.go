package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
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
	// min e max delimitam o intervalo de valores (Min <= Max).
	min, max float64

	// step é o incremento das setas do teclado (ver Step); zero usa 5% do
	// intervalo.
	step float64
	// onChange é chamado com o novo valor a cada ajuste feito pelo usuário
	// (ver OnChange).
	onChange func(float64)
	value    float64
	dragging bool
	hover    bool
	focused  bool
	bound    *state.State[float64]
}

// Step define o incremento das setas do teclado; zero (o padrão) usa 5% do
// intervalo. Encadeável.
func (s *Slider) Step(v float64) *Slider {
	s.step = v
	return s
}

// OnChange define o callback chamado com o novo valor a cada ajuste feito
// pelo usuário. Encadeável.
func (s *Slider) OnChange(fn func(float64)) *Slider {
	s.onChange = fn
	return s
}

// NewSlider cria um slider com o intervalo dado e o valor inicial em min.
// O tema é herdado no mount.
func NewSlider(min, max float64) *Slider {
	if max < min {
		min, max = max, min
	}
	return &Slider{min: min, max: max, value: min}
}

// Min devolve o limite inferior do intervalo.
func (s *Slider) Min() float64 {
	return s.min
}

// Max devolve o limite superior do intervalo.
func (s *Slider) Max() float64 {
	return s.max
}

// SetRange troca o intervalo (invertido se max < min), reajusta o valor
// atual para caber nele e agenda um redesenho. Não dispara callbacks.
// Encadeável.
func (s *Slider) SetRange(min, max float64) *Slider {
	if max < min {
		min, max = max, min
	}
	s.min, s.max = min, max
	s.SetValue(s.value)
	s.Invalidate()
	return s
}

// Value devolve o valor atual.
func (s *Slider) Value() float64 {
	return s.value
}

// SetValue define o valor programaticamente (limitado ao intervalo),
// sincroniza o State vinculado (se houver) e agenda um redesenho. Não
// dispara OnChange. Encadeável.
func (s *Slider) SetValue(v float64) *Slider {
	v = s.clamp(v)
	if s.value != v {
		s.value = v
		s.Invalidate()
	}
	if s.bound != nil && s.bound.Get() != v {
		s.bound.Set(v)
	}
	return s
}

// BindValue vincula o valor ao State em DUAS vias: ajustes do usuário fazem
// Set no State, e um Set externo move o slider. Encadeável.
func (s *Slider) BindValue(st *state.State[float64]) *Slider {
	s.bound = st
	s.SetValue(st.Get())
	state.WatchWeak(st, s, func(s *Slider, v float64) {
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
func (s *Slider) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		return s.handleKey(e.Key)
	case event.FocusEvent:
		s.focused = e.Gained
		s.notifyFocus(e.Gained)
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseEnter:
			s.hover = true
			return true
		case event.MouseLeave:
			// O arraste continua fora dos bounds (captura); só o realce sai.
			s.hover = false
			return true
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			s.dragging = true
			s.setFromUser(s.valueAt(e.Pos.X))
			return true
		case event.MouseMove:
			if !s.dragging {
				return false
			}
			return s.setFromUser(s.valueAt(e.Pos.X))
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft || !s.dragging {
				return false
			}
			s.dragging = false
			return true
		}
	}
	return false
}

// handleKey ajusta o valor pelo teclado. Devolve true se algo mudou.
func (s *Slider) handleKey(k event.Key) bool {
	step := s.step
	if step <= 0 {
		step = (s.max - s.min) / 20
	}
	switch k {
	case event.KeyLeft:
		return s.setFromUser(s.value - step)
	case event.KeyRight:
		return s.setFromUser(s.value + step)
	case event.KeyHome:
		return s.setFromUser(s.min)
	case event.KeyEnd:
		return s.setFromUser(s.max)
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
	if s.onChange != nil {
		s.onChange(v)
	}
	return true
}

// clamp limita v ao intervalo [Min, Max].
func (s *Slider) clamp(v float64) float64 {
	if v < s.min {
		return s.min
	}
	if v > s.max {
		return s.max
	}
	return v
}

// ratio devolve a posição relativa do valor no intervalo, em [0, 1].
func (s *Slider) ratio() float64 {
	span := s.max - s.min
	if span <= 0 {
		return 0
	}
	return (s.value - s.min) / span
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
		return s.min
	}
	rel := float64(x-x0) / float64(x1-x0)
	return s.min + rel*(s.max-s.min)
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
	half := max(th.Px(th.SliderTrackThickness)/2, 1)
	hx := x0 + int(s.ratio()*float64(x1-x0))

	// Trilho completo e trecho ativo por cima: com raio o trilho vira uma
	// pílula (o raio é limitado à metade da espessura); com raio zero o
	// resultado é pixel a pixel o mesmo dos dois segmentos retos de antes.
	radius := th.RadiusPx()
	render.FillRoundRect(dst, image.Rect(x0, cy-half, x1, cy+half), radius, th.Track)
	render.FillRoundRect(dst, image.Rect(x0, cy-half, hx, cy+half), radius, th.Accent)

	handle := th.Px(th.SliderHandleSize)
	hr := image.Rect(hx-handle/2, cy-handle/2, hx+handle/2, cy+handle/2)
	fill := th.Accent
	switch {
	case s.dragging:
		fill = th.ButtonPressed
	case s.hover:
		fill = th.ButtonHover
	}
	render.FillRoundRect(dst, hr, radius, fill)
	if s.focused {
		render.StrokeRoundRect(dst, hr, radius, 2*th.BorderPx(), th.FocusOutline)
	}
	s.drawDisabledOverlay(dst)
}
