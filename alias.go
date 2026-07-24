package juigo

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
)

// Este arquivo é a fachada do JUIGo: reexporta, por alias, os tipos e os
// construtores dos subpacotes, para que aplicações comuns importem apenas
// "github.com/JonathanSantos/JUIGo". A documentação de cada tipo vive no subpacote correspondente.

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
	// Scroll é o container de rolagem vertical com recorte.
	Scroll = widget.Scroll
	// Dropdown é o seletor de uma opção com popup em overlay.
	Dropdown = widget.Dropdown
	// ProgressBar é o indicador só-leitura de progresso.
	ProgressBar = widget.ProgressBar
	// Image exibe uma image.Image com proporção preservada.
	Image = widget.Image
	// Radio é a opção de escolha exclusiva (grupo = State compartilhado).
	Radio = widget.Radio
	// TextArea é o editor de texto multilinha.
	TextArea = widget.TextArea
	// Modal é o diálogo centralizado sobre pano de fundo translúcido.
	Modal = widget.Modal
	// Popup é o painel ancorado num ponto, sem escurecer o fundo (menus de
	// contexto, diálogos leves).
	Popup = widget.Popup
	// Tabs organiza páginas de conteúdo em abas.
	Tabs = widget.Tabs
	// Navigator gerencia uma pilha de telas com transições animadas
	// (Push/Pop/Replace).
	Navigator = widget.Navigator
	// SplitPane divide a área entre dois painéis com um divisor arrastável.
	SplitPane = widget.SplitPane
	// Transition é o efeito visual da troca de telas do Navigator.
	Transition = widget.Transition
	// DropTarget é implementado por widgets que aceitam soltar um arrasto
	// (ver StartDrag).
	DropTarget = widget.DropTarget
	// PreeditEvent é o estado da pré-edição de um IME (composição de
	// texto), roteado por foco.
	PreeditEvent = event.PreeditEvent
	// TextCaret é implementado por editores de texto: o retângulo do
	// cursor, âncora da janela de candidatos do IME.
	TextCaret = widget.TextCaret
	// CodeEditor é o editor de texto para código: fonte mono, gutter,
	// tabs, undo coalescido e rolagem virtualizada.
	CodeEditor = widget.CodeEditor
	// Highlighter colore uma linha de código por vez, com estado entre
	// linhas (lexers prontos em juigo/syntax).
	Highlighter = widget.Highlighter
	// HighlightSpan é um trecho contíguo de linha com a mesma classe.
	HighlightSpan = widget.HighlightSpan
	// HighlightState é o estado do lexer carregado entre linhas.
	HighlightState = widget.HighlightState
	// SyntaxStyle é a classe léxica de um trecho (a cor vem de
	// Theme.Syntax).
	SyntaxStyle = widget.SyntaxStyle
	// Table é a tabela de células de texto com cabeçalho fixo e seleção.
	Table = widget.Table
	// Grid distribui filhos em grade de N colunas (formulários alinhados).
	Grid = widget.Grid
	// Align é o alinhamento horizontal do Text.
	Align = widget.Align
	// ButtonState é o estado visual do Button.
	ButtonState = widget.ButtonState
	// CursorShape é o formato do cursor do mouse desejado por um widget.
	CursorShape = widget.CursorShape
)

// Tema (juigo/theme).
type (
	// Theme centraliza cores, fonte, métricas e a escala HiDPI.
	Theme = theme.Theme
)

// List é a lista virtualizada de linhas uniformes com pool reciclado.
type List[W Widget] = widget.List[W]

// History é o valor com pilhas de desfazer/refazer (ver state.History).
type History[T any] = state.History[T]

// Reatividade (juigo/state).
type (
	// State é um valor observável tipado; Set redesenha a interface.
	State[T any] = state.State[T]
	// Observable é o aspecto sem tipo de um State (base de Combine).
	Observable = state.Observable
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
	// ScrollEvent é a rolagem da roda do mouse/trackpad (por geometria).
	ScrollEvent = event.ScrollEvent
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

	CursorDefault = widget.CursorDefault
	CursorText    = widget.CursorText
	CursorHand    = widget.CursorHand
	CursorResizeH = widget.CursorResizeH
	CursorResizeV = widget.CursorResizeV

	TransitionNone       = widget.TransitionNone
	TransitionFade       = widget.TransitionFade
	TransitionSlideLeft  = widget.TransitionSlideLeft
	TransitionSlideRight = widget.TransitionSlideRight
	TransitionSlideUp    = widget.TransitionSlideUp
	TransitionSlideDown  = widget.TransitionSlideDown

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
	KeyUp        = event.KeyUp
	KeyDown      = event.KeyDown
	KeyEscape    = event.KeyEscape
	KeyI         = event.KeyI
	KeyA         = event.KeyA
	KeyC         = event.KeyC
	KeyV         = event.KeyV
	KeyX         = event.KeyX
	KeyZ         = event.KeyZ

	// Classes léxicas do highlight (ver SyntaxStyle).
	SyntaxText    = widget.SyntaxText
	SyntaxKeyword = widget.SyntaxKeyword
	SyntaxString  = widget.SyntaxString
	SyntaxNumber  = widget.SyntaxNumber
	SyntaxComment = widget.SyntaxComment
	SyntaxBuiltin = widget.SyntaxBuiltin

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

// NewScroll cria um container de rolagem vertical para o filho dado —
// normalmente combinado com Grow para definir a altura da viewport (ver
// widget.NewScroll).
func NewScroll(child Widget) *Scroll { return widget.NewScroll(child) }

// NewDropdown cria um seletor com as opções dadas (ver widget.NewDropdown).
func NewDropdown(options ...string) *Dropdown { return widget.NewDropdown(options...) }

// NewProgressBar cria uma barra de progresso (ver widget.NewProgressBar).
func NewProgressBar(min, max float64) *ProgressBar { return widget.NewProgressBar(min, max) }

// NewImage cria um widget de imagem (ver widget.NewImage).
func NewImage(src image.Image) *Image { return widget.NewImage(src) }

// NewRadio cria um rádio de escolha exclusiva; agrupe vários vinculando ao
// mesmo State com BindValue (ver widget.NewRadio).
func NewRadio(label, value string) *Radio { return widget.NewRadio(label, value) }

// NewTextArea cria um editor multilinha (ver widget.NewTextArea).
func NewTextArea(placeholder string) *TextArea { return widget.NewTextArea(placeholder) }

// NewCodeEditor cria um editor de código vazio (ver widget.CodeEditor).
func NewCodeEditor() *CodeEditor { return widget.NewCodeEditor() }

// NewModal cria um diálogo modal com o conteúdo dado; exiba com Show (ver
// widget.NewModal).
func NewModal(content Widget) *Modal { return widget.NewModal(content) }

// NewPopup cria um popup ancorado com o conteúdo dado; exiba com ShowAt
// (ver widget.NewPopup).
func NewPopup(content Widget) *Popup { return widget.NewPopup(content) }

// NewTabs cria um conjunto de abas vazio; adicione páginas com Add (ver
// widget.Tabs).
func NewTabs() *Tabs { return widget.NewTabs() }

// NewNavigator cria um navegador de telas com a pilha vazia; empilhe a tela
// inicial com Push (ver widget.Navigator).
func NewNavigator() *Navigator { return widget.NewNavigator() }

// NewSplitPane cria um divisor lado a lado entre a (esquerda) e b (direita),
// meio a meio (ver widget.SplitPane).
func NewSplitPane(a, b Widget) *SplitPane { return widget.NewSplitPane(a, b) }

// StartDrag inicia um arrasto com o payload e o rótulo do fantasma; chame
// de um widget fonte durante uma captura de mouse (ver widget.StartDrag e
// DropTarget).
func StartDrag(payload any, label string) { widget.StartDrag(payload, label) }

// NewTable cria uma tabela de texto: títulos de coluna, total de linhas e o
// callback de célula (ver widget.NewTable).
func NewTable(titulos []string, count int, celula func(linha, coluna int) string) *Table {
	return widget.NewTable(titulos, count, celula)
}

// NewGrid cria uma grade com o número de colunas e os filhos dados (ver
// widget.NewGrid).
func NewGrid(colunas int, children ...Widget) *Grid { return widget.NewGrid(colunas, children...) }

// NewList cria uma lista virtualizada: count linhas, criar() fabrica uma
// linha do pool e vincular(linha, i) a preenche com os dados do índice (ver
// widget.NewList).
func NewList[W Widget](count int, criar func() W, vincular func(W, int)) *List[W] {
	return widget.NewList(count, criar, vincular)
}

// Tooltip associa a w um texto de dica exibido quando o ponteiro pausa sobre
// o widget, devolvendo o próprio w (ver widget.Tooltip).
func Tooltip[W Widget](w W, texto string) W { return widget.Tooltip(w, texto) }

// BindDisabled vincula o estado desabilitado de w ao State (true desabilita),
// devolvendo o próprio w (ver widget.BindDisabled).
func BindDisabled[W Widget](w W, s *State[bool]) W { return widget.BindDisabled(w, s) }

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

// Combine deriva um State de várias fontes (ver state.Combine).
func Combine[T any](compute func() T, fontes ...Observable) *State[T] {
	return state.Combine(compute, fontes...)
}

// Not deriva o State booleano inverso — o par natural de BindDisabled (ver
// state.Not).
func Not(s *State[bool]) *State[bool] { return state.Not(s) }

// NewHistory cria um histórico de desfazer/refazer com o valor inicial (ver
// state.NewHistory).
func NewHistory[T any](inicial T) *History[T] { return state.NewHistory(inicial) }

// DarkTheme constrói o tema escuro na escala 1; troque em runtime com
// App.SetTheme (ver theme.Dark).
func DarkTheme() (*Theme, error) { return theme.Dark() }

// DefaultTheme constrói o tema padrão na escala 1 (ver theme.Default).
func DefaultTheme() (*Theme, error) { return theme.Default() }

// ClassicTheme constrói o tema claro com o visual clássico — cantos retos e
// botão sem borda (ver theme.Classic).
func ClassicTheme() (*Theme, error) { return theme.Classic() }

// NewEventBus cria um barramento síncrono (ver event.NewBus).
func NewEventBus() *EventBus { return event.NewBus() }

// Mount injeta um tema em uma árvore de widgets sem App — útil para testes e
// renderização offscreen (ver widget.Mount).
func Mount(w Widget, t *Theme) { widget.Mount(w, t) }

// Focus move o foco de teclado para w programaticamente (ver widget.Focus).
func Focus(w Widget) { widget.Focus(w) }

// Blur limpa o foco de teclado programaticamente (ver widget.Blur).
func Blur() { widget.Blur() }
