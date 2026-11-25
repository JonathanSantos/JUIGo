package widget

import (
	"image"
	"image/color"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
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

// textRole é o papel tipográfico do Text (corpo por padrão).
type textRole int

const (
	roleBody textRole = iota
	roleTitle
	roleSubtitle
	roleCaption
)

// Text é um widget somente-leitura que desenha uma linha de texto com a
// fonte do tema. Não é focável e não consome eventos.
//
// Além do corpo (padrão), o Text assume os papéis tipográficos do tema:
// Title e Subtitle usam a fonte de DISPLAY (Theme.UseDisplayFont — Go
// Mono Bold no Nanquim, Go Bold no Classic) e Caption usa a fonte do corpo
// em tamanho menor. Os tamanhos vêm de Theme.TitleSize/SubtitleSize/
// CaptionSize e seguem trocas de tema e de escala em runtime.
type Text struct {
	BaseWidget
	// align controla o alinhamento horizontal dentro dos bounds (ver
	// Align/Left/Center/Right).
	align Align

	// role é o papel tipográfico (ver Title/Subtitle/Caption).
	role textRole
	// col sobrescreve a cor do texto (ver Color); o valor zero usa a cor do
	// tema. tone escolhe uma cor de ESTADO do tema (Danger/Success/Warning)
	// — segue trocas de tema em runtime.
	col  color.RGBA
	tone textTone
	text string
	// clip é a visão recortada reutilizada pelo Draw (sem alocação).
	clip image.RGBA
}

// NewText cria um Text com o conteúdo dado, alinhado à esquerda. O tema é
// herdado no mount.
func NewText(s string) *Text {
	return &Text{text: s}
}

// Align define o alinhamento horizontal do texto dentro dos bounds —
// Left, Center e Right são os atalhos. Encadeável.
func (t *Text) Align(a Align) *Text {
	if t.align != a {
		t.align = a
		t.Invalidate()
	}
	return t
}

// Left alinha o texto à esquerda (o padrão). Encadeável.
func (t *Text) Left() *Text { return t.Align(AlignLeft) }

// Center alinha o texto ao centro. Encadeável.
func (t *Text) Center() *Text { return t.Align(AlignCenter) }

// Right alinha o texto à direita. Encadeável.
func (t *Text) Right() *Text { return t.Align(AlignRight) }

// Title dá ao texto o papel de TÍTULO: fonte de display do tema no tamanho
// Theme.TitleSize. Encadeável.
func (t *Text) Title() *Text {
	t.role = roleTitle
	t.Invalidate()
	return t
}

// Subtitle dá ao texto o papel de SUBTÍTULO: fonte de display no tamanho
// Theme.SubtitleSize. Encadeável.
func (t *Text) Subtitle() *Text {
	t.role = roleSubtitle
	t.Invalidate()
	return t
}

// Caption dá ao texto o papel de LEGENDA: a fonte do corpo no tamanho
// Theme.CaptionSize — para notas e metadados. Combina bem com Color
// (Theme.Placeholder) para apagar junto. Encadeável.
func (t *Text) Caption() *Text {
	t.role = roleCaption
	t.Invalidate()
	return t
}

// Color sobrescreve a cor do texto com um valor fixo; para a cor de erro que
// acompanha o tema, prefira Danger. Encadeável.
func (t *Text) Color(c color.RGBA) *Text {
	t.col = c
	t.tone = toneNone
	t.Invalidate()
	return t
}

// Danger desenha o texto na cor de erro do tema (Theme.Danger), seguindo
// trocas de tema em runtime — ideal para mensagens de validação. Encadeável.
func (t *Text) Danger() *Text {
	t.tone = toneDanger
	t.Invalidate()
	return t
}

// Success desenha o texto na cor de sucesso do tema (Theme.Success), como
// Danger. Encadeável.
func (t *Text) Success() *Text {
	t.tone = toneSuccess
	t.Invalidate()
	return t
}

// Warning desenha o texto na cor de aviso do tema (Theme.Warning), como
// Danger. Encadeável.
func (t *Text) Warning() *Text {
	t.tone = toneWarning
	t.Invalidate()
	return t
}

// textTone é a cor de ESTADO de um texto (ver Text.Danger/Success/Warning).
type textTone uint8

const (
	toneNone textTone = iota
	toneDanger
	toneSuccess
	toneWarning
)

// color devolve a cor do tom no tema; sem tom, a cor sobrescrita (Color)
// ou, se ela for zero, o texto do tema.
func (tone textTone) color(th *theme.Theme, override color.RGBA) color.RGBA {
	switch tone {
	case toneDanger:
		return th.Danger
	case toneSuccess:
		return th.Success
	case toneWarning:
		return th.Warning
	}
	if override == (color.RGBA{}) {
		return th.Text
	}
	return override
}

// BindText vincula o conteúdo ao State: o Text passa a exibir o valor atual
// e é atualizado — com redesenho automático — a cada Set. Encadeável.
func (t *Text) BindText(s *state.State[string]) *Text {
	t.SetText(s.Get())
	state.WatchWeak(s, t, func(t *Text, v string) { t.SetText(v) })
	return t
}

// Text devolve o conteúdo atual.
func (t *Text) Text() string {
	return t.text
}

// SetText troca o conteúdo e agenda um redesenho se ele mudou. Encadeável.
func (t *Text) SetText(s string) *Text {
	if t.text == s {
		return t
	}
	t.text = s
	t.Invalidate()
	return t
}

// roleFont devolve a fonte do papel tipográfico atual, ou nil para o corpo
// (que desenha pela face principal do tema).
func (t *Text) roleFont() *theme.TextFont {
	if t.theme == nil {
		return nil
	}
	switch t.role {
	case roleTitle:
		return t.theme.Title()
	case roleSubtitle:
		return t.theme.Subtitle()
	case roleCaption:
		return t.theme.Caption()
	}
	return nil
}

// PreferredSize devolve a largura medida do texto e a altura de uma linha,
// pelo papel tipográfico atual. Antes do mount (sem tema), devolve zero.
func (t *Text) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	if f := t.roleFont(); f != nil {
		return image.Point{X: f.Measure(t.text), Y: f.LineHeight()}
	}
	return image.Point{
		X: t.theme.MeasureString(t.text),
		Y: t.theme.LineHeight(),
	}
}

// Draw desenha o texto alinhado horizontalmente e centralizado na vertical,
// recortado aos bounds do widget, com a fonte do papel tipográfico atual.
func (t *Text) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	bounds := t.Bounds()
	c := t.tone.color(t.theme, t.col)

	var w, lineH, ascent int
	f := t.roleFont()
	if f != nil {
		w, lineH, ascent = f.Measure(t.text), f.LineHeight(), f.Ascent()
	} else {
		w, lineH, ascent = t.theme.MeasureString(t.text), t.theme.LineHeight(), t.theme.Ascent()
	}
	x := bounds.Min.X
	switch t.align {
	case AlignCenter:
		x = bounds.Min.X + (bounds.Dx()-w)/2
	case AlignRight:
		x = bounds.Max.X - w
	}
	y := bounds.Min.Y + (bounds.Dy()-lineH)/2 + ascent
	view := render.Clip(dst, bounds, &t.clip)
	if f != nil {
		f.Draw(view, t.text, image.Pt(x, y), c)
	} else {
		t.theme.DrawText(view, t.text, image.Pt(x, y), c)
	}
}
