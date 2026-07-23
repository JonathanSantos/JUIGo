package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/render"
)

// Tooltip associa a w um texto de dica, exibido pelo App quando o ponteiro
// pausa sobre o widget (atraso em Theme.TooltipDelay). Devolve o próprio w
// com o tipo concreto preservado, para uso inline na montagem da árvore:
//
//	juigo.Tooltip(juigo.NewButton("Enviar", enviar), "Envia o formulário")
//
// Em widgets que não embutem BaseWidget, é um no-op.
func Tooltip[W Widget](w W, text string) W {
	if p, ok := any(w).(interface{ setTooltip(string) }); ok {
		p.setTooltip(text)
	}
	return w
}

func (b *BaseWidget) setTooltip(text string) { b.tooltip = text }
func (b *BaseWidget) tooltipText() string    { return b.tooltip }

// TooltipTextOf devolve o texto de dica de w ("" se não houver).
func TooltipTextOf(w Widget) string {
	if p, ok := w.(interface{ tooltipText() string }); ok {
		return p.tooltipText()
	}
	return ""
}

// TooltipView é a caixa visual da dica — um retângulo escuro com o texto.
// É desenhada pelo App na camada mais alta, nunca participa de hit-test e
// não é focável. Exposta para shells alternativos.
type TooltipView struct {
	BaseWidget
	text string
	clip image.RGBA
}

// NewTooltipView cria uma caixa de dica vazia.
func NewTooltipView() *TooltipView {
	return &TooltipView{}
}

// SetText define o texto exibido.
func (t *TooltipView) SetText(s string) {
	t.text = s
}

// PreferredSize devolve o texto medido mais o respiro interno.
func (t *TooltipView) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	return image.Point{
		X: t.theme.MeasureString(t.text) + 2*t.theme.PaddingPx(),
		Y: t.theme.LineHeight() + t.theme.PaddingPx(),
	}
}

// Draw desenha a caixa e o texto, recortados aos bounds.
func (t *TooltipView) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	th := t.theme
	bounds := t.Bounds()
	view := render.Clip(dst, bounds, &t.clip)
	render.FillRect(view, bounds, th.TooltipBackground)
	x := bounds.Min.X + th.PaddingPx()
	y := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(view, t.text, image.Pt(x, y), th.TooltipText)
}
