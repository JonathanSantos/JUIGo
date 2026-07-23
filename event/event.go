package event

import "image"

// Event é a interface marcadora dos eventos internos do JUIGo.
//
// Regras de roteamento (implementadas pelo App):
//   - Eventos de mouse roteiam por GEOMETRIA: hit-test da raiz até o widget
//     mais profundo que contém o ponto; propagam para cima se não consumidos.
//   - Eventos de teclado (KeyEvent) e texto (CharEvent) roteiam por FOCO:
//     vão direto ao widget focado, sem hit-test.
type Event interface {
	isEvent()
}

// MouseEventKind identifica o tipo de um MouseEvent.
type MouseEventKind int

const (
	// MouseDown indica que um botão do mouse foi pressionado.
	MouseDown MouseEventKind = iota
	// MouseUp indica que um botão do mouse foi solto.
	MouseUp
	// MouseMove indica movimento do cursor.
	MouseMove
	// MouseEnter indica que o cursor entrou nos limites do widget.
	// Entregue diretamente pelo App ao widget afetado (sem hit-test).
	MouseEnter
	// MouseLeave indica que o cursor saiu dos limites do widget.
	// Entregue diretamente pelo App ao widget afetado (sem hit-test).
	MouseLeave
)

// MouseButton identifica um botão físico do mouse.
type MouseButton int

const (
	// MouseButtonLeft é o botão esquerdo.
	MouseButtonLeft MouseButton = iota
	// MouseButtonRight é o botão direito.
	MouseButtonRight
	// MouseButtonMiddle é o botão do meio.
	MouseButtonMiddle
)

// MouseEvent é um evento de mouse, com posição em coordenadas da janela.
type MouseEvent struct {
	Kind   MouseEventKind
	Pos    image.Point
	Button MouseButton
}

func (MouseEvent) isEvent() {}

// Key identifica as teclas não-textuais reconhecidas pelo JUIGo. Entrada de
// texto não passa por Key: chega como CharEvent (rune) via CharCallback.
type Key int

const (
	// KeyUnknown é uma tecla não mapeada.
	KeyUnknown Key = iota
	// KeyEnter é a tecla Enter/Return (inclui o Enter do teclado numérico).
	KeyEnter
	// KeySpace é a barra de espaço.
	KeySpace
	// KeyTab é a tecla Tab (usada pelo App para avançar o foco).
	KeyTab
	// KeyBackspace apaga o caractere anterior ao cursor.
	KeyBackspace
	// KeyDelete apaga o caractere sob o cursor.
	KeyDelete
	// KeyLeft move o cursor à esquerda.
	KeyLeft
	// KeyRight move o cursor à direita.
	KeyRight
	// KeyHome move o cursor ao início.
	KeyHome
	// KeyEnd move o cursor ao fim.
	KeyEnd
	// KeyUp e KeyDown navegam verticalmente (ex.: itens de um Dropdown).
	KeyUp
	KeyDown
	// KeyEscape cancela/fecha (ex.: popup do Dropdown).
	KeyEscape
	// KeyA, KeyC, KeyV e KeyX são as letras dos atalhos de edição —
	// selecionar tudo, copiar, colar e recortar — quando combinadas com o
	// modificador de comando. Texto comum não passa por Key: chega como
	// CharEvent.
	KeyA
	KeyC
	KeyV
	KeyX
)

// Modifiers é o conjunto (bitmask) de teclas modificadoras ativas em um
// KeyEvent.
type Modifiers int

const (
	// ModShift indica Shift pressionado.
	ModShift Modifiers = 1 << iota
	// ModControl indica Ctrl pressionado.
	ModControl
	// ModAlt indica Alt/Option pressionado.
	ModAlt
	// ModSuper indica Super (Cmd no macOS, Windows no Windows) pressionado.
	ModSuper
)

// Shift informa se Shift está ativo (ex.: seleção de texto com as setas).
func (m Modifiers) Shift() bool {
	return m&ModShift != 0
}

// Command informa se um modificador de "comando" está ativo — Ctrl (Windows/
// Linux) ou Cmd/Super (macOS). É a base dos atalhos de edição, aceitando os
// dois para funcionar igual em qualquer plataforma.
func (m Modifiers) Command() bool {
	return m&(ModControl|ModSuper) != 0
}

// KeyEvent é um evento de tecla pressionada (ou repetida), com os
// modificadores ativos no momento. Roteado por foco: entregue apenas ao
// widget focado.
type KeyEvent struct {
	Key  Key
	Mods Modifiers
}

func (KeyEvent) isEvent() {}

// CharEvent é um caractere de texto digitado (já decodificado pelo SO,
// incluindo acentuação e composição). Roteado por foco.
type CharEvent struct {
	Rune rune
}

func (CharEvent) isEvent() {}

// FocusEvent notifica um widget que ele ganhou (Gained=true) ou perdeu o
// foco de teclado. Entregue diretamente pelo App.
type FocusEvent struct {
	Gained bool
}

func (FocusEvent) isEvent() {}

// ScrollEvent é a rolagem da roda do mouse ou do trackpad, roteada por
// GEOMETRIA para o widget sob o cursor (propaga para cima se não consumida —
// um container de rolagem no limite devolve false e deixa um ancestral
// rolar). DX e DY são os deltas como entregues pelo GLFW: a roda do mouse dá
// passos inteiros; trackpads dão valores fracionários contínuos.
type ScrollEvent struct {
	Pos image.Point
	DX  float64
	DY  float64
}

func (ScrollEvent) isEvent() {}

// Bus é um barramento publish/subscribe simples para comunicação entre
// partes da aplicação. O Publish é SÍNCRONO: os handlers executam na mesma
// goroutine, antes do retorno. Não é seguro para uso concorrente — como todo
// o JUIGo, deve ser usado apenas na main thread.
type Bus struct {
	subs map[string][]func(any)
}

// NewBus cria um barramento vazio.
func NewBus() *Bus {
	return &Bus{subs: make(map[string][]func(any))}
}

// Subscribe registra fn para receber publicações do tópico dado.
func (b *Bus) Subscribe(topic string, fn func(any)) {
	b.subs[topic] = append(b.subs[topic], fn)
}

// Publish entrega data, sincronamente, a todos os inscritos do tópico, na
// ordem de inscrição.
func (b *Bus) Publish(topic string, data any) {
	for _, fn := range b.subs[topic] {
		fn(data)
	}
}
