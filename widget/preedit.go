package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// preeditState é o estado e os caches de uma composição de IME,
// compartilhado pelos editores de texto (Input, TextArea). Os caches são
// atualizados por evento (set/measure), nunca por frame desenhado.
type preeditState struct {
	runes   []rune
	caret   int
	blocks  []int
	focused int
	// Derivados por measure: string da composição, largura total, X do
	// cursor interno e fronteiras dos blocos em pixels.
	text   string
	w      int
	caretX int
	blockX []int
}

// active informa se há composição em andamento.
func (p *preeditState) active() bool {
	return len(p.runes) > 0
}

// set registra o estado vindo do evento (cursor limitado ao texto).
func (p *preeditState) set(e event.PreeditEvent) {
	p.runes = append(p.runes[:0], []rune(e.Text)...)
	p.caret = e.Caret
	if p.caret < 0 {
		p.caret = 0
	}
	if p.caret > len(p.runes) {
		p.caret = len(p.runes)
	}
	p.blocks = append(p.blocks[:0], e.Blocks...)
	p.focused = e.FocusedBlock
}

// clear descarta a composição; devolve true se havia uma.
func (p *preeditState) clear() bool {
	if !p.active() {
		return false
	}
	p.runes = p.runes[:0]
	p.caret = 0
	p.blocks = p.blocks[:0]
	return true
}

// measure recalcula os caches de medida para o tema dado.
func (p *preeditState) measure(th *theme.Theme) {
	if !p.active() || th == nil {
		p.text, p.w, p.caretX = "", 0, 0
		p.blockX = p.blockX[:0]
		return
	}
	p.text = string(p.runes)
	p.w = th.MeasureString(p.text)
	p.caretX = th.MeasureString(string(p.runes[:p.caret]))
	// Fronteiras dos blocos em pixels (prefixos acumulados), para o desenho
	// dos sublinhados não medir nada por frame.
	p.blockX = p.blockX[:0]
	if len(p.blocks) > 0 {
		p.blockX = append(p.blockX, 0)
		pos := 0
		for _, n := range p.blocks {
			pos += n
			if pos > len(p.runes) {
				pos = len(p.runes)
			}
			p.blockX = append(p.blockX, th.MeasureString(string(p.runes[:pos])))
		}
	}
}

// drawUnderline desenha os sublinhados da composição a partir de compX/y:
// um traço único sem blocos; com blocos, um por bloco (com vão de 1px) e o
// bloco em conversão mais grosso, na cor de destaque. Não aloca.
func (p *preeditState) drawUnderline(view *image.RGBA, th *theme.Theme, compX, y int) {
	esp := th.BorderPx()
	if len(p.blockX) < 2 {
		render.FillRect(view, image.Rect(compX, y, compX+p.w, y+esp), th.Text)
		return
	}
	for i := 0; i+1 < len(p.blockX); i++ {
		x0 := compX + p.blockX[i]
		x1 := compX + p.blockX[i+1]
		if i+2 < len(p.blockX) && x1 > x0+1 {
			x1-- // vão entre blocos
		}
		alt, cor := esp, th.Text
		if i == p.focused {
			alt, cor = 2*esp, th.Accent
		}
		render.FillRect(view, image.Rect(x0, y, x1, y+alt), cor)
	}
}

// TextCaret é implementado por widgets de edição de texto: devolve o
// retângulo do cursor em coordenadas absolutas da janela — a âncora para a
// janela de candidatos do IME (e um ponto de verificação útil em testes).
type TextCaret interface {
	// CaretRect devolve o retângulo do cursor de texto; durante uma
	// composição, o do cursor DENTRO da pré-edição. Vazio sem tema.
	CaretRect() image.Rectangle
}

// CaretRectOf devolve o retângulo do cursor de w, se ele expõe um.
func CaretRectOf(w Widget) (image.Rectangle, bool) {
	if t, ok := w.(TextCaret); ok {
		return t.CaretRect(), true
	}
	return image.Rectangle{}, false
}

// Preedit entrega o estado de pré-edição do IME ao widget focado (ver
// event.PreeditEvent). O texto confirmado da composição chega depois pelos
// Char normais — este evento só atualiza o desenho da composição corrente.
func (s *Session) Preedit(ev event.PreeditEvent) {
	if s.focused == nil || DisabledOf(s.focused) {
		return
	}
	if f := s.focused; f.HandleEvent(ev) {
		s.AddDamage(f.Bounds())
	}
}

// CaretRect devolve o retângulo do cursor do widget focado, se ele for um
// editor de texto — é o que a camada de plataforma informa ao IME para
// posicionar a janela de candidatos.
func (s *Session) CaretRect() (image.Rectangle, bool) {
	if s.focused == nil {
		return image.Rectangle{}, false
	}
	return CaretRectOf(s.focused)
}
