package quick

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/widget"
)

// MenuItem é uma entrada de Menu: rótulo e a ação disparada ao selecionar.
type MenuItem struct {
	// Label é o texto da entrada.
	Label string
	// OnSelect é chamado quando a entrada é selecionada, depois de o menu
	// fechar — pode abrir diálogos ou outros menus.
	OnSelect func()
}

// Item constrói um MenuItem — açúcar para chamadas de Menu enxutas.
func Item(label string, onSelect func()) MenuItem {
	return MenuItem{Label: label, OnSelect: onSelect}
}

// Menu abre um menu de contexto ancorado no ponto dado (coordenadas de
// janela — o Pos do event.MouseEvent de botão direito, por exemplo):
//
//	quick.Menu(pos,
//	    quick.Item("Renomear…", renomear),
//	    quick.Item("Excluir", excluir),
//	)
//
// Setas Cima/Baixo (Home/End nos extremos) navegam, Enter ou clique
// selecionam — o menu fecha e então OnSelect é chamado —, Escape ou clique
// fora fecham sem selecionar. Uma chamada por abertura, como os diálogos.
func Menu(at image.Point, items ...MenuItem) *widget.Popup {
	l := &menuList{items: items}
	p := widget.NewPopup(l)
	l.popup = p
	p.ShowAt(at)
	return p
}

// menuList é o conteúdo do popup do Menu: linhas selecionáveis por mouse e
// teclado, no padrão do popup do Dropdown.
type menuList struct {
	widget.BaseWidget
	items     []MenuItem
	highlight int
	popup     *widget.Popup
}

// Focusable devolve true: o menu abre focado e recebe o teclado.
func (l *menuList) Focusable() bool {
	return true
}

// CursorShape devolve a mãozinha.
func (l *menuList) CursorShape() widget.CursorShape {
	return widget.CursorHand
}

// itemHeight devolve a altura de uma linha do menu em pixels.
func (l *menuList) itemHeight() int {
	th := l.Theme()
	return th.LineHeight() + th.PaddingPx()
}

// itemAt devolve o índice da entrada sob o ponto, ou -1.
func (l *menuList) itemAt(p image.Point) int {
	b := l.Bounds()
	if !p.In(b) {
		return -1
	}
	i := (p.Y - b.Min.Y) / l.itemHeight()
	if i < 0 || i >= len(l.items) {
		return -1
	}
	return i
}

// choose fecha o menu e dispara a ação da entrada i.
func (l *menuList) choose(i int) {
	if i < 0 || i >= len(l.items) {
		return
	}
	fn := l.items[i].OnSelect
	l.popup.Close()
	if fn != nil {
		fn()
	}
}

// PreferredSize devolve a largura do rótulo mais largo com respiro e a
// altura de todas as linhas.
func (l *menuList) PreferredSize() image.Point {
	th := l.Theme()
	if th == nil {
		return image.Point{}
	}
	var w int
	for _, it := range l.items {
		w = max(w, th.MeasureString(it.Label))
	}
	return image.Pt(w+2*th.PaddingPx(), len(l.items)*l.itemHeight())
}

// HandleEvent navega com o teclado e seleciona por Enter ou clique.
func (l *menuList) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		switch e.Key {
		case event.KeyUp:
			if l.highlight > 0 {
				l.highlight--
			}
		case event.KeyDown:
			if l.highlight < len(l.items)-1 {
				l.highlight++
			}
		case event.KeyHome:
			l.highlight = 0
		case event.KeyEnd:
			l.highlight = len(l.items) - 1
		case event.KeyEnter, event.KeySpace:
			l.choose(l.highlight)
		default:
			return false
		}
		return true
	case event.FocusEvent:
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
			l.choose(l.itemAt(e.Pos))
			return true
		}
	}
	return false
}

// Draw desenha as linhas com o realce da entrada corrente.
func (l *menuList) Draw(dst *image.RGBA) {
	th := l.Theme()
	if th == nil {
		return
	}
	b := l.Bounds()
	itemH := l.itemHeight()
	y := b.Min.Y
	for i, it := range l.items {
		if i == l.highlight {
			render.FillRoundRect(dst, image.Rect(b.Min.X, y, b.Max.X, y+itemH), th.RadiusPx(), th.HoverBackground)
		}
		baseline := y + (itemH-th.LineHeight())/2 + th.Ascent()
		th.DrawText(dst, it.Label, image.Pt(b.Min.X+th.PaddingPx(), baseline), th.Text)
		y += itemH
	}
}

// Toast exibe um aviso transitório na base da janela — a confirmação leve
// que não pede interação ("Contato salvo"). Some sozinho após
// Theme.ToastDuration; um novo Toast substitui o atual. Para controlar a
// duração, use widget.ShowToast.
func Toast(message string) {
	widget.ShowToast(message, 0)
}
