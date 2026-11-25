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
	// kids é a fatia reutilizada de Children (sem alocação por chamada).
	kids [1]Widget

	child  Widget
	offset int     // deslocamento vertical do conteúdo, em pixels
	accum  float64 // resto fracionário de rolagens de trackpad
	clip   image.RGBA

	// horizontal habilita o eixo X (ver Horizontal): o filho recebe a
	// própria largura preferida e rola com o delta horizontal do trackpad.
	horizontal bool
	offsetX    int
	accumX     float64
	// axis trava cada gesto de trackpad no eixo dominante (rolagem 2D).
	axis axisLock

	// stick mantém a rolagem colada ao FIM enquanto o conteúdo cresce (ver
	// StickToEnd); atEnd registra se o último layout terminou no fim, e
	// toEnd pede o salto ao fim no próximo layout (ver ScrollToEnd).
	stick bool
	atEnd bool
	toEnd bool
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
	// Fatia de um array do próprio Scroll: consultado a cada mount e a cada
	// evento de mouse, não pode alocar.
	s.kids[0] = s.child
	return s.kids[:]
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
	s.toEnd = false
	s.atEnd = false // o layout recalcula; um salto para cima solta o StickToEnd
	s.Invalidate()
}

// ScrollToEnd rola até o fim do conteúdo no próximo layout — inclusive o
// conteúdo acrescentado nesta mesma rodada de eventos (a mensagem recém
// adicionada a um chat, a linha nova de um log).
func (s *Scroll) ScrollToEnd() {
	s.toEnd = true
	s.Invalidate()
}

// StickToEnd liga o modo "colado ao fim": enquanto a rolagem estiver no fim,
// conteúdo novo a mantém no fim (chats, logs, terminais); se o usuário
// rolar para cima, a posição dele é respeitada até voltar ao fim.
// Encadeável.
func (s *Scroll) StickToEnd(on bool) *Scroll {
	s.stick = on
	s.atEnd = on // nasce colado: o primeiro conteúdo já aparece no fim
	s.Invalidate()
	return s
}

// OffsetX devolve o deslocamento horizontal atual, em pixels (zero sem
// Horizontal).
func (s *Scroll) OffsetX() int {
	return s.offsetX
}

// ScrollToX define o deslocamento horizontal, em pixels (limitado ao
// intervalo válido no próximo layout; sem efeito sem Horizontal).
func (s *Scroll) ScrollToX(x int) {
	s.offsetX = x
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
	contentH := max(pref.Y, bounds.Dy())
	maxOff := contentH - bounds.Dy()
	if s.toEnd || (s.stick && s.atEnd) {
		s.offset = maxOff
		s.toEnd = false
	}
	s.offset = min(max(s.offset, 0), maxOff)
	s.atEnd = s.offset >= maxOff
	contentW := bounds.Dx()
	if s.horizontal && pref.X > contentW {
		contentW = pref.X
	}
	if limit := contentW - bounds.Dx(); s.offsetX > limit {
		s.offsetX = limit
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
	stepPx := float64(s.theme.Px(s.theme.ScrollStep))

	// Trava de eixo do trackpad: só quando os dois eixos rolam.
	dxIn, dyIn := e.DX, e.DY
	if s.horizontal {
		dxIn, dyIn = s.axis.filter(dxIn, dyIn, s.theme.ScrollAxisLock)
	}

	// Eixo vertical.
	consumed := false
	if limit := s.child.Bounds().Dy() - s.Bounds().Dy(); limit > 0 {
		// Acumula deltas fracionários (trackpad) até renderem pixels
		// inteiros.
		delta := dyIn*stepPx + s.accum
		step := int(delta)
		s.accum = delta - float64(step)
		next := s.offset - step
		if next < 0 {
			next = 0
		}
		if next > limit {
			next = limit
		}
		switch {
		case next != s.offset:
			s.offset = next
			s.atEnd = next >= limit // rolar para cima solta o StickToEnd
			consumed = true
		case step != 0:
			// Já estava no limite nesta direção: propaga para o ancestral.
			s.accum = 0
		case dyIn != 0:
			consumed = true // delta minúsculo: aguardando acumular
		}
	}

	// Eixo horizontal (quando habilitado).
	if s.horizontal {
		if limit := s.child.Bounds().Dx() - s.Bounds().Dx(); limit > 0 {
			delta := dxIn*stepPx + s.accumX
			step := int(delta)
			s.accumX = delta - float64(step)
			next := s.offsetX - step
			if next < 0 {
				next = 0
			}
			if next > limit {
				next = limit
			}
			switch {
			case next != s.offsetX:
				s.offsetX = next
				consumed = true
			case step != 0:
				s.accumX = 0
			case dxIn != 0:
				consumed = true
			}
		}
	}
	return consumed
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
		thumbH := max(viewportH*viewportH/contentH, minThumb)
		maxOff := contentH - viewportH
		thumbY := bounds.Min.Y + (viewportH-thumbH)*s.offset/maxOff
		// Com raio, o indicador vira uma pílula (raio limitado à metade da
		// largura); com raio zero segue o retângulo reto de antes.
		render.FillRoundRect(view, image.Rect(bounds.Max.X-w-margin, thumbY, bounds.Max.X-margin, thumbY+thumbH), th.RadiusPx(), th.Placeholder)
	}
	contentW := s.child.Bounds().Dx()
	viewportW := bounds.Dx()
	if s.horizontal && contentW > viewportW {
		thumbW := max(viewportW*viewportW/contentW, minThumb)
		maxOff := contentW - viewportW
		thumbX := bounds.Min.X + (viewportW-thumbW)*s.offsetX/maxOff
		render.FillRoundRect(view, image.Rect(thumbX, bounds.Max.Y-w-margin, thumbX+thumbW, bounds.Max.Y-margin), th.RadiusPx(), th.Placeholder)
	}
}
