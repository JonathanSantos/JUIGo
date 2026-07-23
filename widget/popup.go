package widget

import (
	"image"

	"juigo/event"
	"juigo/render"
	"juigo/theme"
)

// Popup é um painel ANCORADO num ponto da tela, exibido na camada de
// overlay SEM escurecer o fundo — a base de menus de contexto e diálogos
// leves ("ajustar…", "renomear…"):
//
//	p := juigo.NewPopup(conteudo)
//	p.ShowAt(pontoDoClique) // clique fora, Escape ou Close fecham
//
// O painel é posicionado no ponto (deslocado para caber na janela); clicar
// fora fecha E engole o clique; Escape fecha (entrega de fallback da
// overlay); o primeiro focável do conteúdo abre focado. Como o Modal, é de
// camada única — abrir outro overlay substitui.
type Popup struct {
	BaseWidget
	// onClose é chamado quando o popup fecha, por qualquer caminho (ver
	// OnClose).
	onClose func()

	content Widget
	at      image.Point
	panel   image.Rectangle
	shown   bool
	clip    image.RGBA
}

// NewPopup cria um popup com o conteúdo dado. O tema é herdado ao abrir.
func NewPopup(content Widget) *Popup {
	return &Popup{content: content}
}

// OnClose define o callback chamado quando o popup fecha, por qualquer
// caminho (Escape, clique fora ou Close). Encadeável.
func (p *Popup) OnClose(fn func()) *Popup {
	p.onClose = fn
	return p
}

// Children devolve o conteúdo (roteamento, mount e foco).
func (p *Popup) Children() []Widget {
	if p.content == nil {
		return nil
	}
	return []Widget{p.content}
}

// SetTheme define um tema explícito e o propaga ao conteúdo.
func (p *Popup) SetTheme(t *theme.Theme) {
	p.BaseWidget.SetTheme(t)
	Mount(p, t)
}

// SpansWindow devolve true: o popup recebe a janela inteira no layout (para
// ancorar o painel com recuo garantido) e intercepta cliques fora dele.
func (p *Popup) SpansWindow() bool {
	return true
}

// Focusable devolve true: sem focáveis no conteúdo, o próprio popup foca
// (Escape fecha direto).
func (p *Popup) Focusable() bool {
	return true
}

// Shown informa se o popup está aberto.
func (p *Popup) Shown() bool {
	return p.shown
}

// ShowAt exibe o popup ancorado no ponto dado (reposiciona se já aberto).
func (p *Popup) ShowAt(at image.Point) {
	p.at = at
	if p.shown {
		p.Invalidate()
		return
	}
	p.shown = true
	OpenOverlay(p)
}

// Close fecha o popup (restaurando o foco anterior) e chama OnClose.
func (p *Popup) Close() {
	if !p.shown {
		return
	}
	p.shown = false
	CloseOverlay(p)
	if p.onClose != nil {
		p.onClose()
	}
}

// Layout ancora o painel no ponto (tamanho preferido do conteúdo mais o
// respiro), deslocando-o para caber na janela.
func (p *Popup) Layout(bounds image.Rectangle) {
	p.BaseWidget.Layout(bounds)
	if p.content == nil || p.theme == nil {
		return
	}
	pad := p.theme.PaddingPx()
	pref := p.content.PreferredSize()
	w := pref.X + 2*pad
	h := pref.Y + 2*pad
	pos := p.at
	if pos.X+w > bounds.Max.X {
		pos.X = bounds.Max.X - w
	}
	if pos.Y+h > bounds.Max.Y {
		pos.Y = bounds.Max.Y - h
	}
	if pos.X < bounds.Min.X {
		pos.X = bounds.Min.X
	}
	if pos.Y < bounds.Min.Y {
		pos.Y = bounds.Min.Y
	}
	p.panel = image.Rectangle{Min: pos, Max: pos.Add(image.Pt(w, h))}
	p.content.Layout(p.panel.Inset(pad))
}

// HandleEvent fecha com Escape ou clique fora do painel (engolindo o
// clique, como o Modal).
func (p *Popup) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEscape {
			p.Close()
			return true
		}
		return false
	case event.FocusEvent:
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseDown:
			if e.Button == event.MouseButtonLeft && !e.Pos.In(p.panel) {
				p.Close()
			}
			return true
		case event.MouseUp:
			return true
		}
	}
	return false
}

// Draw desenha o painel (fundo, borda e conteúdo recortado) — sem pano de
// fundo escurecido: a interface por baixo continua visível.
func (p *Popup) Draw(dst *image.RGBA) {
	if p.theme == nil {
		return
	}
	th := p.theme
	render.FillRect(dst, p.panel, th.InputBackground)
	render.StrokeRect(dst, p.panel, th.BorderPx(), th.InputBorder)
	if p.content != nil {
		view := render.Clip(dst, p.panel, &p.clip)
		p.content.Draw(view)
	}
}
