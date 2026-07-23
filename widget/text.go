package widget

import (
	"image"
	"image/color"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// Align define o alinhamento horizontal do texto dentro dos bounds.
type Align int

const (
	// AlignLeft alinha à esquerda.
	AlignLeft Align = iota
	// AlignCenter centraliza.
	AlignCenter
	// AlignRight alinha à direita.
	AlignRight
)

// Text é um widget somente-leitura que desenha uma linha de texto com a
// fonte do tema. Não é focável e não consome eventos.
type Text struct {
	BaseWidget
	// Align controla o alinhamento horizontal dentro dos bounds.
	Align Align

	// col sobrescreve a cor do texto (ver Color); o valor zero usa a cor do
	// tema. danger usa Theme.Danger — segue trocas de tema em runtime.
	col    color.RGBA
	danger bool
	text   string
	// clip é a visão recortada reutilizada pelo Draw (sem alocação).
	clip image.RGBA
}

// NewText cria um Text com o conteúdo dado, alinhado à esquerda. O tema é
// herdado no mount.
func NewText(s string) *Text {
	return &Text{text: s}
}

// Center alinha o texto ao centro. Encadeável.
func (t *Text) Center() *Text {
	t.Align = AlignCenter
	return t
}

// Right alinha o texto à direita. Encadeável.
func (t *Text) Right() *Text {
	t.Align = AlignRight
	return t
}

// Color sobrescreve a cor do texto com um valor fixo; para a cor de erro que
// acompanha o tema, prefira Danger. Encadeável.
func (t *Text) Color(c color.RGBA) *Text {
	t.col = c
	t.danger = false
	return t
}

// Danger desenha o texto na cor de erro do tema (Theme.Danger), seguindo
// trocas de tema em runtime — ideal para mensagens de validação. Encadeável.
func (t *Text) Danger() *Text {
	t.danger = true
	return t
}

// BindText vincula o conteúdo ao State: o Text passa a exibir o valor atual
// e é atualizado — com redesenho automático — a cada Set. Encadeável.
func (t *Text) BindText(s *state.State[string]) *Text {
	t.SetText(s.Get())
	s.Watch(func(v string) {
		t.SetText(v)
	})
	return t
}

// Text devolve o conteúdo atual.
func (t *Text) Text() string {
	return t.text
}

// SetText troca o conteúdo e agenda um redesenho se ele mudou.
func (t *Text) SetText(s string) {
	if t.text == s {
		return
	}
	t.text = s
	t.Invalidate()
}

// PreferredSize devolve a largura medida do texto e a altura de uma linha.
// Antes do mount (sem tema), devolve zero.
func (t *Text) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: t.theme.MeasureString(t.text),
		Y: t.theme.LineHeight(),
	}
}

// Draw desenha o texto alinhado horizontalmente e centralizado na vertical,
// recortado aos bounds do widget.
func (t *Text) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	bounds := t.Bounds()
	c := t.col
	if t.danger {
		c = t.theme.Danger
	} else if c == (color.RGBA{}) {
		c = t.theme.Text
	}

	w := t.theme.MeasureString(t.text)
	x := bounds.Min.X
	switch t.Align {
	case AlignCenter:
		x = bounds.Min.X + (bounds.Dx()-w)/2
	case AlignRight:
		x = bounds.Max.X - w
	}
	y := bounds.Min.Y + (bounds.Dy()-t.theme.LineHeight())/2 + t.theme.Ascent()
	view := render.Clip(dst, bounds, &t.clip)
	t.theme.DrawText(view, t.text, image.Pt(x, y), c)
}
