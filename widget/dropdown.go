package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// Dropdown é um seletor de uma opção entre várias: um gatilho com o valor
// atual que, ao ser acionado (clique, Enter ou Espaço), abre um popup na
// camada de overlay com a lista de opções. No popup: mover o mouse realça,
// clique ou Enter selecionam, setas cima/baixo e Home/End navegam, Escape ou
// clique fora fecham sem selecionar. É focável.
//
// Nasce pronto para reatividade: BindValue vincula a opção selecionada a um
// State[string] em duas vias.
type Dropdown struct {
	BaseWidget
	// options são as opções exibidas, na ordem (ver Options/SetOptions).
	options []string

	// placeholder é exibido quando nada está selecionado (ver Placeholder).
	placeholder string
	// onChange é chamado com a opção escolhida a cada seleção feita pelo
	// usuário (ver OnChange).
	onChange func(string)
	// onIndex recebe o índice escolhido pelo usuário (ver BindChoice).
	onIndex  func(int)
	selected int
	open     bool
	hover    bool
	pressed  bool
	focused  bool
	list     *dropdownList
	bound    *state.State[string]
	clip     image.RGBA
}

// NewDropdown cria um dropdown com as opções dadas; a primeira vem
// selecionada (ou nenhuma, se não houver opções). O tema é herdado no mount.
func NewDropdown(options ...string) *Dropdown {
	d := &Dropdown{options: options, selected: -1}
	if len(options) > 0 {
		d.selected = 0
	}
	d.list = &dropdownList{owner: d}
	return d
}

// Placeholder devolve o texto exibido quando nada está selecionado.
func (d *Dropdown) Placeholder() string {
	return d.placeholder
}

// SetPlaceholder define o texto exibido quando nada está selecionado e
// agenda um redesenho. Encadeável.
func (d *Dropdown) SetPlaceholder(s string) *Dropdown {
	if d.placeholder != s {
		d.placeholder = s
		d.Invalidate()
	}
	return d
}

// Options devolve as opções exibidas, na ordem. A fatia é a do próprio
// Dropdown: não a modifique — troque as opções com SetOptions.
func (d *Dropdown) Options() []string {
	return d.options
}

// SetOptions troca as opções e agenda um redesenho. A seleção é preservada
// pelo TEXTO: se a opção selecionada ainda existir, segue selecionada (no
// novo índice); senão a seleção é limpa — sem disparar OnChange nem tocar o
// State vinculado. Encadeável.
func (d *Dropdown) SetOptions(options ...string) *Dropdown {
	current := d.Value()
	d.options = options
	d.selected = -1
	if current != "" {
		d.selected = d.indexOf(current)
	}
	d.Invalidate()
	return d
}

// OnChange define o callback chamado com a opção escolhida a cada seleção
// feita pelo usuário. Encadeável.
func (d *Dropdown) OnChange(fn func(string)) *Dropdown {
	d.onChange = fn
	return d
}

// Value devolve a opção selecionada ("" se nenhuma).
func (d *Dropdown) Value() string {
	if d.selected < 0 || d.selected >= len(d.options) {
		return ""
	}
	return d.options[d.selected]
}

// Selected devolve o índice selecionado (-1 se nenhum).
func (d *Dropdown) Selected() int {
	return d.selected
}

// Select define a seleção programaticamente (limitada às opções; -1 limpa),
// sincroniza o State vinculado (se houver) e agenda um redesenho. Não
// dispara OnChange.
func (d *Dropdown) Select(i int) {
	if i < -1 || i >= len(d.options) {
		return
	}
	if i != d.selected {
		d.selected = i
		d.Invalidate()
	}
	if v := d.Value(); d.bound != nil && d.bound.Get() != v {
		d.bound.Set(v)
	}
}

// BindValue vincula a seleção ao State em DUAS vias: seleções do usuário
// fazem Set no State, e um Set externo seleciona a opção de mesmo texto
// (valores fora de Options são ignorados). Encadeável.
func (d *Dropdown) BindValue(s *state.State[string]) *Dropdown {
	d.bound = s
	if i := d.indexOf(s.Get()); i >= 0 {
		d.Select(i)
	}
	state.WatchWeak(s, d, func(d *Dropdown, v string) {
		if i := d.indexOf(v); i >= 0 && i != d.selected {
			d.Select(i)
		}
	})
	return d
}

func (d *Dropdown) indexOf(v string) int {
	for i, o := range d.options {
		if o == v {
			return i
		}
	}
	return -1
}

// Focusable devolve true: o dropdown participa da cadeia de foco.
func (d *Dropdown) Focusable() bool {
	return true
}

// CursorShape do Dropdown: mãozinha.
func (d *Dropdown) CursorShape() CursorShape {
	return CursorHand
}

// PreferredSize acomoda a opção mais larga, o chevron e o padding.
func (d *Dropdown) PreferredSize() image.Point {
	if d.theme == nil {
		return image.Point{}
	}
	th := d.theme
	widest := th.MeasureString(d.placeholder)
	for _, o := range d.options {
		if w := th.MeasureString(o); w > widest {
			widest = w
		}
	}
	return image.Point{
		X: widest + 3*th.PaddingPx() + th.Px(8),
		Y: th.LineHeight() + 2*th.PaddingPx(),
	}
}

// HandleEvent implementa o acionamento (semântica de clique do Button), o
// teclado e o registro de foco.
func (d *Dropdown) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		switch e.Key {
		case event.KeyEnter, event.KeySpace:
			d.toggle()
			return true
		case event.KeyDown, event.KeyUp:
			// Fechado, as setas trocam a seleção diretamente.
			delta := 1
			if e.Key == event.KeyUp {
				delta = -1
			}
			if i := d.selected + delta; i >= 0 && i < len(d.options) {
				d.selectFromUser(i)
			}
			return true
		}
		return false
	case event.FocusEvent:
		d.focused = e.Gained
		d.notifyFocus(e.Gained)
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseEnter:
			d.hover = true
			return true
		case event.MouseLeave:
			d.hover = false
			d.pressed = false
			return true
		case event.MouseDown:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			d.pressed = true
			return true
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft || !d.pressed {
				return false
			}
			d.pressed = false
			if e.Pos.In(d.Bounds()) {
				d.toggle()
			}
			return true
		}
	}
	return false
}

// toggle abre ou fecha o popup.
func (d *Dropdown) toggle() {
	if d.open {
		d.closePopup()
		return
	}
	if len(d.options) == 0 || d.theme == nil {
		return
	}
	d.open = true
	d.list.highlight = max(d.selected, 0)
	b := d.Bounds()
	h := d.list.itemHeight()*len(d.options) + 2*d.theme.BorderPx()
	d.list.Layout(image.Rect(b.Min.X, b.Max.Y+d.theme.Px(2), b.Max.X, b.Max.Y+d.theme.Px(2)+h))
	OpenOverlay(d.list)
}

// closePopup fecha o popup (o App restaura o foco anterior).
func (d *Dropdown) closePopup() {
	if !d.open {
		return
	}
	d.open = false
	CloseOverlay(d.list)
}

// selectFromUser aplica uma seleção feita pelo usuário, propagando ao State
// vinculado e ao OnChange, e fecha o popup se aberto.
func (d *Dropdown) selectFromUser(i int) {
	if i < 0 || i >= len(d.options) {
		return
	}
	changed := i != d.selected
	d.selected = i
	if changed {
		if d.bound != nil && d.bound.Get() != d.options[i] {
			d.bound.Set(d.options[i])
		}
		if d.onIndex != nil {
			d.onIndex(i)
		}
		if d.onChange != nil {
			d.onChange(d.options[i])
		}
	}
	d.closePopup()
}

// Draw desenha o gatilho: fundo, borda, valor atual e o chevron.
func (d *Dropdown) Draw(dst *image.RGBA) {
	if d.theme == nil {
		return
	}
	th := d.theme
	bounds := d.Bounds()

	radius := th.RadiusPx()
	render.FillRoundRect(dst, bounds, radius, th.InputBackground)
	border := th.InputBorder
	if d.focused || d.hover || d.open {
		border = th.InputBorderFocused
	}
	render.StrokeRoundRect(dst, bounds, radius, th.BorderPx(), border)

	view := render.Clip(dst, bounds, &d.clip)
	baseline := bounds.Min.Y + (bounds.Dy()-th.LineHeight())/2 + th.Ascent()
	if d.selected >= 0 && d.selected < len(d.options) {
		th.DrawText(view, d.options[d.selected], image.Pt(bounds.Min.X+th.PaddingPx(), baseline), th.Text)
	} else if d.placeholder != "" {
		th.DrawText(view, d.placeholder, image.Pt(bounds.Min.X+th.PaddingPx(), baseline), th.Placeholder)
	}

	// Chevron ▼ desenhado com faixas de largura decrescente.
	size := th.Px(8)
	cx := bounds.Max.X - th.PaddingPx() - size
	cy := bounds.Min.Y + (bounds.Dy()-size/2)/2
	for i := 0; i < size/2; i++ {
		render.FillRect(view, image.Rect(cx+i, cy+i, cx+size-i, cy+i+1), th.Placeholder)
	}
	d.drawDisabledOverlay(dst)
}

// dropdownList é o popup de opções, exibido na camada de overlay. Recebe o
// foco enquanto aberto; perder o foco (clique fora, Tab) fecha o popup.
type dropdownList struct {
	BaseWidget
	owner     *Dropdown
	highlight int
	clip      image.RGBA
}

func (l *dropdownList) Focusable() bool {
	return true
}

func (l *dropdownList) CursorShape() CursorShape {
	return CursorHand
}

// itemHeight devolve a altura de uma linha do popup.
func (l *dropdownList) itemHeight() int {
	th := l.owner.theme
	return th.LineHeight() + th.PaddingPx()
}

// itemAt devolve o índice do item na coordenada dada (-1 fora).
func (l *dropdownList) itemAt(p image.Point) int {
	if !p.In(l.Bounds()) {
		return -1
	}
	th := l.owner.theme
	i := (p.Y - l.Bounds().Min.Y - th.BorderPx()) / l.itemHeight()
	if i < 0 || i >= len(l.owner.options) {
		return -1
	}
	return i
}

// HandleEvent trata realce, seleção, navegação e fechamento do popup.
func (l *dropdownList) HandleEvent(ev event.Event) bool {
	d := l.owner
	switch e := ev.(type) {
	case event.KeyEvent:
		switch e.Key {
		case event.KeyDown:
			if l.highlight < len(d.options)-1 {
				l.highlight++
			}
			return true
		case event.KeyUp:
			if l.highlight > 0 {
				l.highlight--
			}
			return true
		case event.KeyHome:
			l.highlight = 0
			return true
		case event.KeyEnd:
			l.highlight = len(d.options) - 1
			return true
		case event.KeyEnter, event.KeySpace:
			d.selectFromUser(l.highlight)
			return true
		case event.KeyEscape:
			d.closePopup()
			return true
		}
		return false
	case event.FocusEvent:
		if !e.Gained {
			// Foco saiu (clique fora, Tab): fecha sem selecionar.
			d.closePopup()
		}
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseMove, event.MouseEnter:
			if i := l.itemAt(e.Pos); i >= 0 && i != l.highlight {
				l.highlight = i
				return true
			}
			return false
		case event.MouseDown:
			return e.Button == event.MouseButtonLeft
		case event.MouseUp:
			if e.Button != event.MouseButtonLeft {
				return false
			}
			if i := l.itemAt(e.Pos); i >= 0 {
				d.selectFromUser(i)
			}
			return true
		}
	}
	return false
}

// Draw desenha o popup: fundo, borda e as opções, com realce do item sob o
// ponteiro e a opção atual em destaque.
func (l *dropdownList) Draw(dst *image.RGBA) {
	d := l.owner
	th := d.theme
	if th == nil {
		return
	}
	bounds := l.Bounds()
	radius := th.RadiusPx()
	render.FillRoundRect(dst, bounds, radius, th.InputBackground)
	render.StrokeRoundRect(dst, bounds, radius, th.BorderPx(), th.InputBorder)

	view := render.Clip(dst, bounds, &l.clip)
	itemH := l.itemHeight()
	y := bounds.Min.Y + th.BorderPx()
	for i, opt := range d.options {
		row := image.Rect(bounds.Min.X+th.BorderPx(), y, bounds.Max.X-th.BorderPx(), y+itemH)
		// Realces arredondados como o painel (com raio zero, retângulos
		// retos): a opção escolhida é sempre uma pílula de seleção — forma,
		// não só cor de texto, para ler em qualquer tema —, e as demais sob
		// o ponteiro ou o teclado ganham o hover.
		switch {
		case i == d.selected:
			render.FillRoundRect(view, row, radius, th.Selection)
		case i == l.highlight:
			render.FillRoundRect(view, row, radius, th.HoverBackground)
		}
		baseline := y + (itemH-th.LineHeight())/2 + th.Ascent()
		th.DrawText(view, opt, image.Pt(row.Min.X+th.PaddingPx(), baseline), th.Text)
		y += itemH
	}
}

// BindChoice liga um Dropdown a uma escolha TIPADA: as opções exibidas são
// label(item) de cada item, e o State guarda o próprio item — sem converter
// de e para string na aplicação (números, enums, structs comparáveis).
// Duas vias: escolher faz Set(item), e um Set externo seleciona o item
// igual (valores fora de items limpam a seleção). Substitui as opções do
// Dropdown e devolve o próprio d, para encadear:
//
//	tamanho := juigo.NewState(16)
//	juigo.BindChoice(juigo.NewDropdown(), []int{14, 16, 18}, strconv.Itoa, tamanho)
func BindChoice[T comparable](d *Dropdown, items []T, label func(T) string, s *state.State[T]) *Dropdown {
	options := make([]string, len(items))
	for i, it := range items {
		options[i] = label(it)
	}
	indexOf := func(v T) int {
		for i, it := range items {
			if it == v {
				return i
			}
		}
		return -1
	}
	d.options = options
	d.selected = indexOf(s.Get())
	d.onIndex = func(i int) {
		if s.Get() != items[i] {
			s.Set(items[i])
		}
	}
	d.Invalidate()
	state.WatchWeak(s, d, func(d *Dropdown, v T) {
		if i := indexOf(v); i != d.selected {
			d.Select(i)
		}
	})
	return d
}

// SetValue seleciona a opção de texto igual a s ("" ou um texto fora das
// opções limpa a seleção) e sincroniza o State vinculado. Não dispara
// OnChange. Encadeável.
func (d *Dropdown) SetValue(s string) *Dropdown {
	d.Select(d.indexOf(s))
	return d
}
