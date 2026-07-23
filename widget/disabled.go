package widget

import (
	"image"

	"juigo/internal/hooks"
	"juigo/render"
	"juigo/state"
)

// O estado DESABILITADO é aplicado centralmente, no roteamento: um widget
// desabilitado (e toda a sua subárvore) fica fora do hit-test de mouse e
// rolagem, do hover (sem realce, cursor nem tooltip), da ordem do Tab e da
// entrega de teclado — os widgets não precisam se defender individualmente.
// O visual é uma lavagem translúcida (Theme.DisabledWash) aplicada no Draw.

// SetDisabled habilita/desabilita o widget e agenda um redesenho. Em um
// container, desabilita funcionalmente toda a subárvore (a lavagem visual é
// aplicada nos bounds do widget marcado).
func (b *BaseWidget) SetDisabled(v bool) {
	if b.disabled == v {
		return
	}
	b.disabled = v
	hooks.RequestRepaint()
}

// Disabled informa se o widget está desabilitado.
func (b *BaseWidget) Disabled() bool {
	return b.disabled
}

// isDisabled é a consulta usada pelo roteamento; widgets podem sobrescrever
// para estados que implicam desabilitado (ex.: Button em loading).
func (b *BaseWidget) isDisabled() bool {
	return b.disabled
}

// disabledQuery é satisfeito por todo widget que embute BaseWidget.
type disabledQuery interface {
	isDisabled() bool
}

// DisabledOf informa se w está efetivamente desabilitado para interação
// (inclui estados que implicam desabilitado, como loading).
func DisabledOf(w Widget) bool {
	if q, ok := w.(disabledQuery); ok {
		return q.isDisabled()
	}
	return false
}

// BindDisabled vincula o estado desabilitado de w ao State: true no State
// desabilita. Devolve o próprio w (tipo concreto preservado), para uso
// inline na montagem da árvore:
//
//	juigo.BindDisabled(botaoEnviar, juigo.Map(valor, estaVazio))
func BindDisabled[W Widget](w W, s *state.State[bool]) W {
	if p, ok := any(w).(interface{ SetDisabled(bool) }); ok {
		p.SetDisabled(s.Get())
		s.Watch(func(v bool) {
			p.SetDisabled(v)
		})
	}
	return w
}

// drawDisabledOverlay aplica a lavagem de desabilitado sobre os bounds do
// widget; chamado ao fim do Draw dos widgets interativos e containers.
func (b *BaseWidget) drawDisabledOverlay(dst *image.RGBA) {
	if b.disabled && b.theme != nil {
		render.FillRectOver(dst, b.bounds, b.theme.DisabledWash)
	}
}
