package widget

import (
	"image"

	"juigo/event"
	"juigo/render"
	"juigo/theme"
)

// OverlaySpanning é implementado por overlays que cobrem a janela INTEIRA
// (ex.: Modal): o App faz o layout deles com os bounds do buffer a cada
// renderização e não os fecha no resize.
type OverlaySpanning interface {
	SpansWindow() bool
}

// Modal é um diálogo exibido na camada de overlay: um painel centralizado
// com o conteúdo dado, sobre um pano de fundo translúcido que bloqueia a
// interface por baixo. Fecha com Escape, clique no pano de fundo (quando
// CloseOnBackdrop) ou Close(); ao fechar, o foco anterior é restaurado.
//
//	m := juigo.NewModal(juigo.NewVBox(titulo, campos, botoes))
//	m.Show()
type Modal struct {
	BaseWidget
	// closeOnBackdrop permite fechar clicando no pano de fundo (ver
	// CloseOnBackdrop; padrão true).
	closeOnBackdrop bool
	// onClose é chamado quando o modal fecha, por qualquer caminho (ver
	// OnClose).
	onClose func()

	content Widget
	panel   image.Rectangle
	shown   bool
	clip    image.RGBA
}

// NewModal cria um modal com o conteúdo dado. O tema é herdado ao abrir.
func NewModal(content Widget) *Modal {
	return &Modal{content: content, closeOnBackdrop: true}
}

// CloseOnBackdrop define se clicar no pano de fundo fecha o modal (padrão
// true); desligue para exigir uma ação explícita do conteúdo. Encadeável.
func (m *Modal) CloseOnBackdrop(v bool) *Modal {
	m.closeOnBackdrop = v
	return m
}

// OnClose define o callback chamado quando o modal fecha, por qualquer
// caminho (Escape, clique fora ou Close). Encadeável.
func (m *Modal) OnClose(fn func()) *Modal {
	m.onClose = fn
	return m
}

// Children devolve o conteúdo (roteamento, mount e foco).
func (m *Modal) Children() []Widget {
	if m.content == nil {
		return nil
	}
	return []Widget{m.content}
}

// SetTheme define um tema explícito e o propaga imediatamente à subárvore.
func (m *Modal) SetTheme(t *theme.Theme) {
	m.BaseWidget.SetTheme(t)
	Mount(m, t)
}

// SpansWindow marca o modal como overlay de janela inteira.
func (m *Modal) SpansWindow() bool {
	return true
}

// Focusable devolve true: o modal abre focado (Escape fecha direto).
func (m *Modal) Focusable() bool {
	return true
}

// Shown informa se o modal está aberto.
func (m *Modal) Shown() bool {
	return m.shown
}

// Show exibe o modal na camada de overlay.
func (m *Modal) Show() {
	if m.shown {
		return
	}
	m.shown = true
	OpenOverlay(m)
}

// Close fecha o modal (restaurando o foco anterior) e chama OnClose.
func (m *Modal) Close() {
	if !m.shown {
		return
	}
	m.shown = false
	CloseOverlay(m)
	if m.onClose != nil {
		m.onClose()
	}
}

// Layout centraliza o painel (tamanho preferido do conteúdo mais o respiro,
// limitado à janela) e posiciona o conteúdo dentro dele.
func (m *Modal) Layout(bounds image.Rectangle) {
	m.BaseWidget.Layout(bounds)
	if m.content == nil || m.theme == nil {
		return
	}
	pad := 2 * m.theme.PaddingPx()
	pref := m.content.PreferredSize()
	w := pref.X + 2*pad
	h := pref.Y + 2*pad
	if max := bounds.Dx() - 2*pad; w > max {
		w = max
	}
	if max := bounds.Dy() - 2*pad; h > max {
		h = max
	}
	x := bounds.Min.X + (bounds.Dx()-w)/2
	y := bounds.Min.Y + (bounds.Dy()-h)/2
	m.panel = image.Rect(x, y, x+w, y+h)
	m.content.Layout(m.panel.Inset(pad))
}

// HandleEvent fecha com Escape ou clique no pano de fundo e consome os
// eventos de mouse que chegam até ele (modal bloqueia o que está por baixo).
func (m *Modal) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEscape {
			m.Close()
			return true
		}
		return false
	case event.FocusEvent:
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseDown:
			if e.Button == event.MouseButtonLeft && m.closeOnBackdrop && !e.Pos.In(m.panel) {
				m.Close()
			}
			return true
		case event.MouseUp:
			return true
		}
	}
	return false
}

// Draw desenha o pano de fundo translúcido, o painel e o conteúdo (recortado
// ao painel).
func (m *Modal) Draw(dst *image.RGBA) {
	if m.theme == nil {
		return
	}
	th := m.theme
	render.FillRectOver(dst, m.Bounds(), th.Backdrop)
	render.FillRect(dst, m.panel, th.InputBackground)
	render.StrokeRect(dst, m.panel, th.BorderPx(), th.InputBorder)
	if m.content != nil {
		view := render.Clip(dst, m.panel, &m.clip)
		m.content.Draw(view)
	}
}
