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
	// Options são as opções exibidas, na ordem.
	Options []string

	// placeholder é exibido quando nada está selecionado (ver Placeholder).
	placeholder string
	// onChange é chamado com a opção escolhida a cada seleção feita pelo
	// usuário (ver OnChange).
	onChange func(string)
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
	d := &Dropdown{Options: options, selected: -1}
	if len(options) > 0 {
		d.selected = 0
	}
	d.list = &dropdownList{owner: d}
	return d
}

// Placeholder define o texto exibido quando nada está selecionado (Options
// vazio ou seleção -1). Encadeável.
func (d *Dropdown) Placeholder(s string) *Dropdown {
	d.placeholder = s
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
	if d.selected < 0 || d.selected >= len(d.Options) {
		return ""
	}
	return d.Options[d.selected]
}

// SelectedIndex devolve o índice selecionado (-1 se nenhum).
func (d *Dropdown) SelectedIndex() int {
	return d.selected
}

// Select define a seleção programaticamente (limitada às opções; -1 limpa) e
// agenda um redesenho. Não dispara OnChange; com binding, prefira State.Set.
func (d *Dropdown) Select(i int) {
	if i < -1 || i >= len(d.Options) || i == d.selected {
		return
	}
	d.selected = i
	d.Invalidate()
}

// BindValue vincula a seleção ao State em DUAS vias: seleções do usuário
// fazem Set no State, e um Set externo seleciona a opção de mesmo texto
// (valores fora de Options são ignorados). Encadeável.
func (d *Dropdown) BindValue(s *state.State[string]) *Dropdown {
	d.bound = s
	if i := d.indexOf(s.Get()); i >= 0 {
		d.Select(i)
	}
	s.Watch(func(v string) {
		if i := d.indexOf(v); i >= 0 && i != d.selected {
			d.Select(i)
		}
	})
	return d
}

func (d *Dropdown) indexOf(v string) int {
	for i, o := range d.Options {
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
	for _, o := range d.Options {
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
			if i := d.selected + delta; i >= 0 && i < len(d.Options) {
				d.selectFromUser(i)
			}
			return true
		}
		return false
	case event.FocusEvent:
		d.focused = e.Gained
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
	if len(d.Options) == 0 || d.theme == nil {
		return
	}
	d.open = true
	d.list.highlight = d.selected
	if d.list.highlight < 0 {
		d.list.highlight = 0
	}
	b := d.Bounds()
	h := d.list.itemHeight()*len(d.Options) + 2*d.theme.BorderPx()
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
	if i < 0 || i >= len(d.Options) {
		return
	}
	changed := i != d.selected
	d.selected = i
	if changed {
		if d.bound != nil && d.bound.Get() != d.Options[i] {
			d.bound.Set(d.Options[i])
		}
		if d.onChange != nil {
			d.onChange(d.Options[i])
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
	if d.selected >= 0 && d.selected < len(d.Options) {
		th.DrawText(view, d.Options[d.selected], image.Pt(bounds.Min.X+th.PaddingPx(), baseline), th.Text)
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
	if i < 0 || i >= len(l.owner.Options) {
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
			if l.highlight < len(d.Options)-1 {
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
			l.highlight = len(d.Options) - 1
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
	for i, opt := range d.Options {
		row := image.Rect(bounds.Min.X+th.BorderPx(), y, bounds.Max.X-th.BorderPx(), y+itemH)
		if i == l.highlight {
			// Realce arredondado como o painel: com raio zero volta a ser
			// o retângulo reto de antes.
			render.FillRoundRect(view, row, radius, th.HoverBackground)
		}
		c := th.Text
		if i == d.selected {
			c = th.Accent
		}
		baseline := y + (itemH-th.LineHeight())/2 + th.Ascent()
		th.DrawText(view, opt, image.Pt(row.Min.X+th.PaddingPx(), baseline), c)
		y += itemH
	}
}
