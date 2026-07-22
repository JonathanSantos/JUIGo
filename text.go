package juigo

import (
	"image"
	"image/color"
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
	// Color sobrescreve a cor do texto; o valor zero usa a cor do tema.
	Color color.RGBA

	theme *Theme
	text  string
}

// NewText cria um Text com o tema e o conteúdo dados, alinhado à esquerda.
func NewText(theme *Theme, s string) *Text {
	return &Text{theme: theme, text: s}
}

// Text devolve o conteúdo atual.
func (t *Text) Text() string {
	return t.text
}

// SetText troca o conteúdo. A mudança aparece no próximo redesenho; fora do
// fluxo de eventos, use App.Invalidate para forçá-lo.
func (t *Text) SetText(s string) {
	t.text = s
}

// PreferredSize devolve a largura medida do texto e a altura de uma linha.
func (t *Text) PreferredSize() image.Point {
	return image.Point{
		X: t.theme.MeasureString(t.text),
		Y: t.theme.LineHeight(),
	}
}

// Draw desenha o texto alinhado horizontalmente e centralizado na vertical.
func (t *Text) Draw(dst *image.RGBA) {
	bounds := t.Bounds()
	c := t.Color
	if c == (color.RGBA{}) {
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
	t.theme.DrawText(dst, t.text, image.Pt(x, y), c)
}
