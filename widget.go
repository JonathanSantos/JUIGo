package juigo

import "image"

// Widget é o contrato central de todo elemento de interface do JUIGo.
//
// O ciclo de vida em cada frame é: o App chama Layout na raiz com os limites
// do buffer, depois Draw. Eventos chegam via HandleEvent conforme as regras
// de roteamento descritas em Event.
type Widget interface {
	// Layout posiciona o widget dentro de bounds (coordenadas absolutas da
	// janela). Containers usam Layout para posicionar os filhos.
	Layout(bounds image.Rectangle)
	// Draw desenha o widget no buffer de destino. Não deve alocar.
	Draw(dst *image.RGBA)
	// HandleEvent processa um evento e devolve true se o consumiu. Consumir
	// um evento interrompe a propagação e marca a interface como suja.
	HandleEvent(ev Event) bool
	// Bounds devolve o retângulo absoluto ocupado pelo widget.
	Bounds() image.Rectangle
	// Focusable indica se o widget participa da cadeia de foco de teclado.
	Focusable() bool
	// PreferredSize devolve o tamanho natural do widget, consultado pelos
	// containers de layout (VBox/HBox) durante o Layout.
	PreferredSize() image.Point
}

// ParentWidget é implementado por widgets que possuem filhos (containers).
// O App usa Children para o hit-test de mouse, o hover e a ordem de foco.
type ParentWidget interface {
	Widget
	// Children devolve os filhos na ordem de desenho (o último fica por
	// cima e, portanto, tem prioridade no hit-test).
	Children() []Widget
}

// BaseWidget implementa os comportamentos padrão de Widget e serve para ser
// embutido nos widgets concretos: guarda os bounds definidos por Layout, não
// é focável, não consome eventos e não tem tamanho preferido.
//
// Também guarda o tema do widget: por padrão ele é HERDADO no mount — o App
// injeta o tema na árvore antes do primeiro layout (e a cada renderização,
// cobrindo widgets adicionados dinamicamente), então construtores não pedem
// tema. Use SetTheme para dar a um widget (e seus descendentes) um tema
// próprio.
type BaseWidget struct {
	bounds image.Rectangle
	theme  *Theme
	// themeExplicit protege um tema definido via SetTheme de ser
	// sobrescrito pela injeção do mount.
	themeExplicit bool
}

// Layout guarda os bounds absolutos do widget.
func (b *BaseWidget) Layout(bounds image.Rectangle) {
	b.bounds = bounds
}

// Bounds devolve o retângulo absoluto definido pelo último Layout.
func (b *BaseWidget) Bounds() image.Rectangle {
	return b.bounds
}

// Draw não desenha nada por padrão.
func (b *BaseWidget) Draw(dst *image.RGBA) {}

// HandleEvent não consome eventos por padrão.
func (b *BaseWidget) HandleEvent(ev Event) bool {
	return false
}

// Focusable devolve false por padrão.
func (b *BaseWidget) Focusable() bool {
	return false
}

// PreferredSize devolve zero por padrão (o widget aceita qualquer tamanho).
func (b *BaseWidget) PreferredSize() image.Point {
	return image.Point{}
}

// SetTheme define um tema explícito para este widget. Na injeção do mount,
// os descendentes herdam o tema explícito do ancestral mais próximo, então
// definir o tema de um container tematiza a subárvore inteira.
func (b *BaseWidget) SetTheme(t *Theme) {
	b.theme = t
	b.themeExplicit = t != nil
}

// Theme devolve o tema efetivo do widget — nil antes do mount.
func (b *BaseWidget) Theme() *Theme {
	return b.theme
}

// inheritTheme aplica o tema herdado no mount e devolve o tema efetivo que
// os descendentes devem herdar.
func (b *BaseWidget) inheritTheme(t *Theme) *Theme {
	if b.themeExplicit {
		return b.theme
	}
	b.theme = t
	return t
}

// themeHost é satisfeito por todo widget que embute BaseWidget; o App o usa
// para injetar o tema na árvore durante o mount.
type themeHost interface {
	inheritTheme(t *Theme) *Theme
}

// propagateTheme injeta o tema na árvore (mount). Widgets sem tema explícito
// herdam o do ancestral; os com SetTheme mantêm o próprio e o repassam aos
// descendentes. Idempotente e sem alocações — o App a executa a cada
// renderização para cobrir widgets adicionados dinamicamente.
func propagateTheme(w Widget, t *Theme) {
	if h, ok := w.(themeHost); ok {
		t = h.inheritTheme(t)
	}
	if p, ok := w.(ParentWidget); ok {
		for _, ch := range p.Children() {
			propagateTheme(ch, t)
		}
	}
}

// dispatchMouse roteia um evento de mouse por GEOMETRIA: desce a árvore até
// o widget mais profundo que contém o ponto (entre irmãos sobrepostos, vence
// o desenhado por último) e, se ninguém no ramo consumir, propaga para cima
// entregando ao próprio w. Devolve o widget que consumiu o evento (usado
// pelo App para a captura de mouse), ou nil.
func dispatchMouse(w Widget, ev MouseEvent) Widget {
	if p, ok := w.(ParentWidget); ok {
		children := p.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if ev.Pos.In(children[i].Bounds()) {
				if consumer := dispatchMouse(children[i], ev); consumer != nil {
					return consumer
				}
				break // só o ramo do topo; depois propaga para o pai
			}
		}
	}
	if w.HandleEvent(ev) {
		return w
	}
	return nil
}

// focusableAt devolve o widget FOCÁVEL mais profundo cujo Bounds contém p,
// ou nil. Usado pelo App para mover o foco no clique.
func focusableAt(w Widget, p image.Point) Widget {
	if w == nil || !p.In(w.Bounds()) {
		return nil
	}
	if pw, ok := w.(ParentWidget); ok {
		children := pw.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if deep := focusableAt(children[i], p); deep != nil {
				return deep
			}
		}
	}
	if w.Focusable() {
		return w
	}
	return nil
}

// collectFocusable acumula em out os widgets focáveis na ordem da árvore
// (pré-ordem, filhos na ordem de inserção). Define a ordem do Tab.
func collectFocusable(w Widget, out *[]Widget) {
	if w == nil {
		return
	}
	if w.Focusable() {
		*out = append(*out, w)
	}
	if pw, ok := w.(ParentWidget); ok {
		for _, ch := range pw.Children() {
			collectFocusable(ch, out)
		}
	}
}

// widgetAt devolve o widget mais profundo cujo Bounds contém p, ou nil.
// Usado pelo App para rastrear hover (MouseEnter/MouseLeave).
func widgetAt(w Widget, p image.Point) Widget {
	if w == nil || !p.In(w.Bounds()) {
		return nil
	}
	if pw, ok := w.(ParentWidget); ok {
		children := pw.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if deep := widgetAt(children[i], p); deep != nil {
				return deep
			}
		}
	}
	return w
}
