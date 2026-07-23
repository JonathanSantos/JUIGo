package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Scroll é um container de rolagem VERTICAL com um único filho: o filho é
// medido na altura preferida e desenhado RECORTADO à área visível; a roda do
// mouse (ou trackpad, com deltas fracionários acumulados) rola o conteúdo.
// No limite da rolagem o evento não é consumido e propaga para cima — Scrolls
// aninhados funcionam naturalmente.
//
// O tamanho preferido é o do filho; para dar ao Scroll uma altura de
// viewport menor que o conteúdo, combine com Grow:
//
//	juigo.NewVBox(cabecalho, juigo.Grow(juigo.NewScroll(lista), 1))
//
// Um indicador fino de posição é desenhado à direita quando o conteúdo é
// maior que a área visível.
type Scroll struct {
	BaseWidget

	child  Widget
	offset int     // deslocamento vertical do conteúdo, em pixels
	accum  float64 // resto fracionário de rolagens de trackpad
	clip   image.RGBA

	// horizontal habilita o eixo X (ver Horizontal): o filho recebe a
	// própria largura preferida e rola com o delta horizontal do trackpad.
	horizontal bool
	offsetX    int
	accumX     float64
}

// NewScroll cria um Scroll para o filho dado. O tema é herdado no mount.
func NewScroll(child Widget) *Scroll {
	return &Scroll{child: child}
}

// Horizontal habilita a rolagem também no eixo X: conteúdo mais largo que a
// viewport (tabelas, planilhas) rola com o delta horizontal do trackpad.
// Encadeável.
func (s *Scroll) Horizontal() *Scroll {
	s.horizontal = true
	return s
}

// Children devolve o filho (satisfaz ParentWidget: roteamento, mount, foco).
func (s *Scroll) Children() []Widget {
	if s.child == nil {
		return nil
	}
	return []Widget{s.child}
}

// SetTheme define um tema explícito e o propaga imediatamente à subárvore,
// como nos demais containers.
func (s *Scroll) SetTheme(t *theme.Theme) {
	s.BaseWidget.SetTheme(t)
	Mount(s, t)
}

// Offset devolve o deslocamento atual da rolagem, em pixels (0 = topo).
func (s *Scroll) Offset() int {
	return s.offset
}

// ScrollTo define o deslocamento da rolagem, em pixels (limitado ao
// intervalo válido no próximo layout), e agenda um redesenho.
func (s *Scroll) ScrollTo(y int) {
	s.offset = y
	s.Invalidate()
}

// PreferredSize devolve o tamanho preferido do filho — normalmente o Scroll
// é combinado com Grow para receber a altura da viewport.
func (s *Scroll) PreferredSize() image.Point {
	if s.child == nil {
		return image.Point{}
	}
	return s.child.PreferredSize()
}

// Layout dá ao filho a largura da viewport e a própria altura preferida (no
// mínimo a da viewport), deslocado pela rolagem — que é limitada aqui ao
// intervalo válido.
func (s *Scroll) Layout(bounds image.Rectangle) {
	s.BaseWidget.Layout(bounds)
	if s.child == nil {
		return
	}
	pref := s.child.PreferredSize()
	contentH := pref.Y
	if contentH < bounds.Dy() {
		contentH = bounds.Dy()
	}
	if max := contentH - bounds.Dy(); s.offset > max {
		s.offset = max
	}
	if s.offset < 0 {
		s.offset = 0
	}
	contentW := bounds.Dx()
	if s.horizontal && pref.X > contentW {
		contentW = pref.X
	}
	if max := contentW - bounds.Dx(); s.offsetX > max {
		s.offsetX = max
	}
	if s.offsetX < 0 {
		s.offsetX = 0
	}
	top := bounds.Min.Y - s.offset
	left := bounds.Min.X - s.offsetX
	s.child.Layout(image.Rect(left, top, left+contentW, top+contentH))
	// Filhos virtualizados (List) recebem a viewport para vincular apenas
	// as linhas visíveis, já no layout.
	if vp, ok := s.child.(interface{ SetViewport(image.Rectangle) }); ok {
		vp.SetViewport(bounds)
	}
}

// HandleEvent consome ScrollEvent quando há para onde rolar; no limite,
// devolve false para o evento propagar (Scrolls aninhados).
func (s *Scroll) HandleEvent(ev event.Event) bool {
	e, ok := ev.(event.ScrollEvent)
	if !ok || s.child == nil || s.theme == nil {
		return false
	}
	passo := float64(s.theme.Px(s.theme.ScrollStep))

	// Eixo vertical.
	consumiu := false
	if max := s.child.Bounds().Dy() - s.Bounds().Dy(); max > 0 {
		// Acumula deltas fracionários (trackpad) até renderem pixels
		// inteiros.
		delta := e.DY*passo + s.accum
		step := int(delta)
		s.accum = delta - float64(step)
		novo := s.offset - step
		if novo < 0 {
			novo = 0
		}
		if novo > max {
			novo = max
		}
		switch {
		case novo != s.offset:
			s.offset = novo
			consumiu = true
		case step != 0:
			// Já estava no limite nesta direção: propaga para o ancestral.
			s.accum = 0
		case e.DY != 0:
			consumiu = true // delta minúsculo: aguardando acumular
		}
	}

	// Eixo horizontal (quando habilitado).
	if s.horizontal {
		if max := s.child.Bounds().Dx() - s.Bounds().Dx(); max > 0 {
			delta := e.DX*passo + s.accumX
			step := int(delta)
			s.accumX = delta - float64(step)
			novo := s.offsetX - step
			if novo < 0 {
				novo = 0
			}
			if novo > max {
				novo = max
			}
			switch {
			case novo != s.offsetX:
				s.offsetX = novo
				consumiu = true
			case step != 0:
				s.accumX = 0
			case e.DX != 0:
				consumiu = true
			}
		}
	}
	return consumiu
}

// Draw desenha o filho recortado à área visível e o indicador de posição.
func (s *Scroll) Draw(dst *image.RGBA) {
	if s.child == nil || s.theme == nil {
		return
	}
	th := s.theme
	bounds := s.Bounds()
	view := render.Clip(dst, bounds, &s.clip)
	s.child.Draw(view)

	// Indicadores: só quando o conteúdo excede a viewport em cada eixo.
	w := th.Px(th.ScrollbarWidth)
	margin := 2 * th.BorderPx()
	minThumb := th.Px(4 * th.ScrollbarWidth)
	contentH := s.child.Bounds().Dy()
	viewportH := bounds.Dy()
	if contentH > viewportH {
		thumbH := viewportH * viewportH / contentH
		if thumbH < minThumb {
			thumbH = minThumb
		}
		maxOff := contentH - viewportH
		thumbY := bounds.Min.Y + (viewportH-thumbH)*s.offset/maxOff
		render.FillRect(view, image.Rect(bounds.Max.X-w-margin, thumbY, bounds.Max.X-margin, thumbY+thumbH), th.Placeholder)
	}
	contentW := s.child.Bounds().Dx()
	viewportW := bounds.Dx()
	if s.horizontal && contentW > viewportW {
		thumbW := viewportW * viewportW / contentW
		if thumbW < minThumb {
			thumbW = minThumb
		}
		maxOff := contentW - viewportW
		thumbX := bounds.Min.X + (viewportW-thumbW)*s.offsetX/maxOff
		render.FillRect(view, image.Rect(thumbX, bounds.Max.Y-w-margin, thumbX+thumbW, bounds.Max.Y-margin), th.Placeholder)
	}
}
