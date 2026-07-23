package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/render"
)

// DropTarget é implementado por widgets que aceitam soltar um arrasto.
// Durante o arrasto, o alvo mais profundo sob o cursor cujo CanDrop aceite
// o payload ganha um realce; soltar sobre ele chama Drop.
type DropTarget interface {
	Widget
	// CanDrop informa se o widget aceita este payload — consultado a cada
	// movimento; deve ser barato e sem efeitos.
	CanDrop(payload any) bool
	// Drop recebe o payload solto sobre o widget, com a posição do cursor.
	Drop(payload any, pos image.Point)
}

// StartDrag inicia um arrasto com o payload dado; label é o texto do
// FANTASMA que segue o cursor. Chame de um widget FONTE durante uma captura
// de mouse (tipicamente no primeiro MouseMove após o MouseDown que passou
// de um pequeno limiar). Dali em diante a sessão cuida do resto: realça o
// DropTarget sob o cursor que aceitar o payload, entrega Drop ao soltar e
// cancela no Escape. A fonte continua recebendo os eventos da captura
// normalmente (MouseUp encerra o gesto dela). Fora de uma aplicação, é um
// no-op.
func StartDrag(payload any, label string) {
	hooks.RequestDrag(payload, label)
}

// StartDrag inicia um arrasto na sessão (ver a função de pacote StartDrag).
// Um arrasto por vez: chamadas durante um arrasto ativo são ignoradas.
func (s *Session) StartDrag(payload any, label string) {
	if s.theme == nil || s.dragging {
		return
	}
	s.dragging = true
	s.dragPayload = payload
	if s.dragView == nil {
		s.dragView = NewTooltipView()
	}
	Mount(s.dragView, s.theme)
	s.dragView.SetText(label)
	s.hideTooltip()
	s.updateDrag(s.lastCursor)
	s.AddDamage(s.dragView.Bounds())
}

// Dragging informa se há um arrasto em andamento.
func (s *Session) Dragging() bool {
	return s.dragging
}

// DragPayload devolve o payload do arrasto em andamento (nil sem arrasto).
func (s *Session) DragPayload() any {
	return s.dragPayload
}

// CancelDrag encerra o arrasto sem soltar em alvo nenhum (Escape, abertura
// de overlay). Sem arrasto ativo, não faz nada.
func (s *Session) CancelDrag() {
	s.endDrag()
}

// updateDrag reposiciona o fantasma junto ao cursor e recalcula o alvo sob
// ele, danificando as regiões que mudaram.
func (s *Session) updateDrag(pos image.Point) {
	if !s.dragging {
		return
	}
	off := s.theme.PaddingPx()
	pref := s.dragView.PreferredSize()
	p := pos.Add(image.Pt(off, off))
	if p.X+pref.X > s.size.X {
		p.X = s.size.X - pref.X
	}
	if p.Y+pref.Y > s.size.Y {
		p.Y = s.size.Y - pref.Y
	}
	// O diff de bounds do Layout danifica a posição antiga e a nova.
	s.dragView.Layout(image.Rectangle{Min: p, Max: p.Add(pref)})
	s.AddDamage(s.dragView.Bounds())

	alvo := s.dropTargetAt(pos)
	if alvo != s.dragTarget {
		if s.dragTarget != nil {
			s.AddDamage(s.dragTarget.Bounds())
		}
		if alvo != nil {
			s.AddDamage(alvo.Bounds())
		}
		s.dragTarget = alvo
	}
}

// finishDrag encerra o arrasto entregando Drop ao alvo sob pos, se houver.
func (s *Session) finishDrag(pos image.Point) {
	if !s.dragging {
		return
	}
	alvo := s.dropTargetAt(pos)
	payload := s.dragPayload
	s.endDrag()
	if alvo != nil {
		if t, ok := alvo.(DropTarget); ok {
			t.Drop(payload, pos)
		}
	}
}

// endDrag limpa o estado do arrasto e danifica fantasma e realce.
func (s *Session) endDrag() {
	if !s.dragging {
		return
	}
	s.dragging = false
	s.dragPayload = nil
	if s.dragView != nil {
		s.AddDamage(s.dragView.Bounds())
	}
	if s.dragTarget != nil {
		s.AddDamage(s.dragTarget.Bounds())
		s.dragTarget = nil
	}
}

// dropTargetAt devolve o widget mais profundo sob pos que aceita o payload
// corrente, buscando na overlay (se aberta e contendo pos) ou na raiz.
func (s *Session) dropTargetAt(pos image.Point) Widget {
	root := s.root
	if s.overlay != nil && pos.In(s.overlay.Bounds()) {
		root = s.overlay
	}
	return dropTargetIn(root, pos, s.dragPayload)
}

// dropTargetIn desce a árvore como DeepestAt, devolvendo o DropTarget mais
// profundo que contém pos e aceita o payload.
func dropTargetIn(w Widget, pos image.Point, payload any) Widget {
	if w == nil || !pos.In(w.Bounds()) || DisabledOf(w) {
		return nil
	}
	if pw, ok := w.(ParentWidget); ok {
		children := pw.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if t := dropTargetIn(children[i], pos, payload); t != nil {
				return t
			}
		}
	}
	if t, ok := w.(DropTarget); ok && t.CanDrop(payload) {
		return w
	}
	return nil
}

// drawDrag desenha o realce do alvo e o fantasma do arrasto — camada
// passiva acima do toast, abaixo do inspector.
func (s *Session) drawDrag(dst *image.RGBA) {
	if !s.dragging {
		return
	}
	if t := s.dragTarget; t != nil {
		render.StrokeRoundRect(dst, t.Bounds(), s.theme.RadiusPx(), 2*s.theme.BorderPx(), s.theme.Accent)
	}
	if s.dragView != nil {
		s.dragView.Draw(dst)
	}
}
