package widget

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/JonathanSantos/JUIGo/render"
)

// O INSPECTOR de depuração é uma camada desenhada acima de tudo (Ctrl/Cmd+I
// alterna) que mostra os bounds de todos os widgets, realça o widget sob o
// ponteiro — inclusive desabilitados — e exibe um crachá com tipo, posição,
// tamanho, tamanho preferido e flags. A interface continua interativa
// enquanto ele está ligado.
//
// As cores são FIXAS de alto contraste, deliberadamente fora do tema: é
// ferramenta de desenvolvimento, precisa ser visível sobre qualquer paleta.
// Enquanto ligado, o desenho do crachá aloca (fmt/medições por frame) —
// aceitável para depuração, nunca ativo em produção por padrão.
var (
	inspectorOutline   = color.RGBA{R: 0xE0, G: 0x40, B: 0xFB, A: 0x66}
	inspectorHighlight = color.RGBA{R: 0xFF, G: 0x2D, B: 0x95, A: 0xFF}
	inspectorFill      = color.RGBA{R: 0xFF, G: 0x2D, B: 0x95, A: 0x22}
	inspectorBadgeBG   = color.RGBA{R: 0x14, G: 0x16, B: 0x1A, A: 0xF0}
	inspectorBadgeFG   = color.RGBA{R: 0xF5, G: 0xF7, B: 0xFA, A: 0xFF}
	inspectorBadgeDim  = color.RGBA{R: 0x9A, G: 0xA0, B: 0xA8, A: 0xFF}
)

// SetInspector liga/desliga o inspector de depuração.
func (s *Session) SetInspector(on bool) {
	if s.inspect == on {
		return
	}
	s.inspect = on
	// Ligar OU desligar exige frame completo: a camada cobre a tela toda e,
	// ao desligar, os contornos antigos precisam ser apagados de uma vez.
	s.InvalidateAll()
}

// ToggleInspector alterna o inspector (atalho Ctrl/Cmd+I).
func (s *Session) ToggleInspector() {
	s.SetInspector(!s.inspect)
}

// InspectorEnabled informa se o inspector está ativo.
func (s *Session) InspectorEnabled() bool {
	return s.inspect
}

// inspectTargetAt devolve o widget mais profundo sob p, IGNORANDO o estado
// desabilitado — o inspector enxerga tudo.
func inspectTargetAt(w Widget, p image.Point) Widget {
	if w == nil || !p.In(w.Bounds()) {
		return nil
	}
	if pw, ok := w.(ParentWidget); ok {
		children := pw.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if deep := inspectTargetAt(children[i], p); deep != nil {
				return deep
			}
		}
	}
	return w
}

// typeName devolve o nome curto do tipo concreto ("Button", "meuapp.Card").
func typeName(w Widget) string {
	name := strings.TrimPrefix(fmt.Sprintf("%T", w), "*")
	if rest, ok := strings.CutPrefix(name, "widget."); ok {
		return rest
	}
	return name
}

// drawInspector desenha a camada de depuração: contornos de todos os
// widgets, realce do alvo sob o ponteiro e o crachá de informações.
func (s *Session) drawInspector(dst *image.RGBA) {
	if s.theme == nil {
		return
	}
	outlineTree(dst, s.root)
	outlineTree(dst, s.overlay)

	target := inspectTargetAt(s.overlay, s.lastCursor)
	if target == nil {
		target = inspectTargetAt(s.root, s.lastCursor)
	}
	if target == nil {
		return
	}

	b := target.Bounds()
	render.FillRectOver(dst, b, inspectorFill)
	strokeOver(dst, b, 2, inspectorHighlight)
	s.drawInspectorBadge(dst, target)
}

// outlineTree desenha o contorno fino de cada widget da árvore.
func outlineTree(dst *image.RGBA, w Widget) {
	if w == nil {
		return
	}
	strokeOver(dst, w.Bounds(), 1, inspectorOutline)
	if p, ok := w.(ParentWidget); ok {
		for _, ch := range p.Children() {
			outlineTree(dst, ch)
		}
	}
}

// strokeOver desenha um contorno com blending alfa (por dentro de r).
func strokeOver(dst *image.RGBA, r image.Rectangle, w int, c color.RGBA) {
	if r.Empty() || w <= 0 {
		return
	}
	render.FillRectOver(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+w), c)
	render.FillRectOver(dst, image.Rect(r.Min.X, r.Max.Y-w, r.Max.X, r.Max.Y), c)
	render.FillRectOver(dst, image.Rect(r.Min.X, r.Min.Y+w, r.Min.X+w, r.Max.Y-w), c)
	render.FillRectOver(dst, image.Rect(r.Max.X-w, r.Min.Y+w, r.Max.X, r.Max.Y-w), c)
}

// drawInspectorBadge desenha o crachá com as informações do alvo, próximo ao
// ponteiro e limitado à janela.
func (s *Session) drawInspectorBadge(dst *image.RGBA, target Widget) {
	th := s.theme
	b := target.Bounds()
	pref := target.PreferredSize()

	l1 := fmt.Sprintf("%s — %d×%d @ (%d, %d)", typeName(target), b.Dx(), b.Dy(), b.Min.X, b.Min.Y)

	flags := make([]string, 0, 4)
	flags = append(flags, fmt.Sprintf("pref %d×%d", pref.X, pref.Y))
	if target.Focusable() {
		flags = append(flags, "focável")
	}
	if target == s.focused {
		flags = append(flags, "focado")
	}
	if DisabledOf(target) {
		flags = append(flags, "desabilitado")
	}
	if s.overlay != nil && Contains(s.overlay, target) {
		flags = append(flags, "overlay")
	}
	l2 := strings.Join(flags, " · ")

	pad := th.PaddingPx()
	w := th.MeasureString(l1)
	if w2 := th.MeasureString(l2); w2 > w {
		w = w2
	}
	boxW := w + 2*pad
	boxH := 2*th.LineHeight() + pad

	pos := s.lastCursor.Add(image.Pt(pad, pad*2))
	if pos.X+boxW > s.size.X {
		pos.X = s.size.X - boxW
	}
	if pos.Y+boxH > s.size.Y {
		pos.Y = s.lastCursor.Y - pad - boxH
	}
	if pos.X < 0 {
		pos.X = 0
	}
	if pos.Y < 0 {
		pos.Y = 0
	}

	box := image.Rectangle{Min: pos, Max: pos.Add(image.Pt(boxW, boxH))}
	render.FillRectOver(dst, box, inspectorBadgeBG)
	strokeOver(dst, box, 1, inspectorHighlight)
	th.DrawText(dst, l1, image.Pt(pos.X+pad, pos.Y+pad/2+th.Ascent()), inspectorBadgeFG)
	th.DrawText(dst, l2, image.Pt(pos.X+pad, pos.Y+pad/2+th.LineHeight()+th.Ascent()), inspectorBadgeDim)
}
