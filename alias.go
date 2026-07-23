package juigo

import (
	"juigo/event"
	"juigo/state"
	"juigo/theme"
	"juigo/widget"
)

// Este arquivo é a fachada do JUIGo: reexporta, por alias, os tipos e os
// construtores dos subpacotes, para que aplicações comuns importem apenas
// "juigo". A documentação de cada tipo vive no subpacote correspondente.

// Tipos centrais (juigo/widget).
type (
	// Widget é o contrato central de todo elemento de interface.
	Widget = widget.Widget
	// ParentWidget é implementado por widgets com filhos (containers).
	ParentWidget = widget.ParentWidget
	// BaseWidget dá os comportamentos padrão para embutir em widgets.
	BaseWidget = widget.BaseWidget
	// Container agrupa filhos com posicionamento absoluto.
	Container = widget.Container
	// VBox distribui os filhos verticalmente.
	VBox = widget.VBox
	// HBox distribui os filhos horizontalmente.
	HBox = widget.HBox
	// Text é o widget de linha de texto.
	Text = widget.Text
	// Button é o botão de ação.
	Button = widget.Button
	// Input é o campo de texto de linha única.
	Input = widget.Input
	// Checkbox é a caixa de marcação.
	Checkbox = widget.Checkbox
	// Slider é o controle deslizante de float64.
	Slider = widget.Slider
	// Spacer é o widget invisível que expande para empurrar os irmãos.
	Spacer = widget.Spacer
	// Align é o alinhamento horizontal do Text.
	Align = widget.Align
	// ButtonState é o estado visual do Button.
	ButtonState = widget.ButtonState
)

// Tema (juigo/theme).
type (
	// Theme centraliza cores, fonte, métricas e a escala HiDPI.
	Theme = theme.Theme
)

// Reatividade (juigo/state).
type (
	// State é um valor observável tipado; Set redesenha a interface.
	State[T any] = state.State[T]
)

// Eventos (juigo/event).
type (
	// Event é a interface marcadora dos eventos internos.
	Event = event.Event
	// MouseEvent é um evento de mouse (roteado por geometria).
	MouseEvent = event.MouseEvent
	// MouseEventKind identifica o tipo de um MouseEvent.
	MouseEventKind = event.MouseEventKind
	// MouseButton identifica um botão físico do mouse.
	MouseButton = event.MouseButton
	// KeyEvent é uma tecla pressionada, com modificadores (roteado por foco).
	KeyEvent = event.KeyEvent
	// Key identifica as teclas não-textuais reconhecidas.
	Key = event.Key
	// Modifiers é o bitmask de modificadores de um KeyEvent.
	Modifiers = event.Modifiers
	// CharEvent é um caractere digitado (roteado por foco).
	CharEvent = event.CharEvent
	// FocusEvent notifica ganho/perda de foco.
	FocusEvent = event.FocusEvent
	// EventBus é o barramento publish/subscribe síncrono.
	EventBus = event.Bus
)

// Constantes reexportadas dos subpacotes.
const (
	AlignLeft   = widget.AlignLeft
	AlignCenter = widget.AlignCenter
	AlignRight  = widget.AlignRight

	ButtonStateNormal  = widget.ButtonStateNormal
	ButtonStateHover   = widget.ButtonStateHover
	ButtonStatePressed = widget.ButtonStatePressed

	MouseDown  = event.MouseDown
	MouseUp    = event.MouseUp
	MouseMove  = event.MouseMove
	MouseEnter = event.MouseEnter
	MouseLeave = event.MouseLeave

	MouseButtonLeft   = event.MouseButtonLeft
	MouseButtonRight  = event.MouseButtonRight
	MouseButtonMiddle = event.MouseButtonMiddle

	KeyUnknown   = event.KeyUnknown
	KeyEnter     = event.KeyEnter
	KeySpace     = event.KeySpace
	KeyTab       = event.KeyTab
	KeyBackspace = event.KeyBackspace
	KeyDelete    = event.KeyDelete
	KeyLeft      = event.KeyLeft
	KeyRight     = event.KeyRight
	KeyHome      = event.KeyHome
	KeyEnd       = event.KeyEnd
	KeyA         = event.KeyA
	KeyC         = event.KeyC
	KeyV         = event.KeyV
	KeyX         = event.KeyX

	ModShift   = event.ModShift
	ModControl = event.ModControl
	ModAlt     = event.ModAlt
	ModSuper   = event.ModSuper
)

// NewText cria um Text (ver widget.NewText).
func NewText(s string) *Text { return widget.NewText(s) }

// NewButton cria um Button (ver widget.NewButton).
func NewButton(label string, onClick func()) *Button { return widget.NewButton(label, onClick) }

// NewInput cria um Input (ver widget.NewInput).
func NewInput(placeholder string) *Input { return widget.NewInput(placeholder) }

// NewCheckbox cria um Checkbox (ver widget.NewCheckbox).
func NewCheckbox(label string) *Checkbox { return widget.NewCheckbox(label) }

// NewSlider cria um Slider (ver widget.NewSlider).
func NewSlider(min, max float64) *Slider { return widget.NewSlider(min, max) }

// NewContainer cria um Container absoluto (ver widget.NewContainer).
func NewContainer(children ...Widget) *Container { return widget.NewContainer(children...) }

// NewVBox cria um VBox (ver widget.NewVBox).
func NewVBox(children ...Widget) *VBox { return widget.NewVBox(children...) }

// NewHBox cria um HBox (ver widget.NewHBox).
func NewHBox(children ...Widget) *HBox { return widget.NewHBox(children...) }

// NewSpacer cria um Spacer com peso 1 (ver widget.NewSpacer).
func NewSpacer() *Spacer { return widget.NewSpacer() }

// Grow marca w para expandir no eixo principal do VBox/HBox com o peso dado,
// devolvendo o próprio w (ver widget.Grow).
func Grow[W Widget](w W, peso int) W { return widget.Grow(w, peso) }

// Centered posiciona w no eixo transversal do box com o tamanho preferido,
// centralizado (ver widget.Centered).
func Centered[W Widget](w W) W { return widget.Centered(w) }

// AtStart posiciona w no eixo transversal com o tamanho preferido, alinhado
// ao início (ver widget.AtStart).
func AtStart[W Widget](w W) W { return widget.AtStart(w) }

// AtEnd posiciona w no eixo transversal com o tamanho preferido, alinhado ao
// fim (ver widget.AtEnd).
func AtEnd[W Widget](w W) W { return widget.AtEnd(w) }

// NewState cria um State com o valor inicial dado (ver state.New).
func NewState[T any](v T) *State[T] { return state.New(v) }

// Map deriva um State de outro (ver state.Map).
func Map[A, B any](s *State[A], f func(A) B) *State[B] { return state.Map(s, f) }

// DefaultTheme constrói o tema padrão na escala 1 (ver theme.Default).
func DefaultTheme() (*Theme, error) { return theme.Default() }

// NewEventBus cria um barramento síncrono (ver event.NewBus).
func NewEventBus() *EventBus { return event.NewBus() }

// Mount injeta um tema em uma árvore de widgets sem App — útil para testes e
// renderização offscreen (ver widget.Mount).
func Mount(w Widget, t *Theme) { widget.Mount(w, t) }
