package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
)

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
