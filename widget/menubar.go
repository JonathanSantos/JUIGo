package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
)

// MenuBar é a barra de menus da aplicação: títulos na horizontal que abrem
// menus de Commands — clicar abre, passar o ponteiro com um menu aberto
// troca, e os atalhos dos itens aparecem à direita e se REGISTRAM sozinhos
// como comandos globais no mount (inclusive na paleta Ctrl/Cmd+K).
//
//	juigo.NewMenuBar().
//	    Menu("Arquivo",
//	        juigo.Command{Title: "Novo", Key: juigo.LetterKey('n'),
//	            Mods: juigo.ModControl, Action: novo},
//	        widget.MenuSeparator(),
//	        juigo.Command{Title: "Salvar", Key: juigo.LetterKey('s'),
//	            Mods: juigo.ModControl, Action: salvar},
//	    ).
//	    Menu("Ajuda", juigo.Command{Title: "Sobre", Action: sobre})
//
// No menu aberto: Cima/Baixo navegam, Enter aciona, Esquerda/Direita
// trocam de menu, Escape (ou clique fora) fecha. A ação roda com o menu já
// fechado.
type MenuBar struct {
	BaseWidget
	menus []menuDef
	// open é o índice do menu aberto (-1 = nenhum); hover, o título sob o
	// ponteiro.
	open  int
	hover int
	pane  *menuPane
	clip  image.RGBA
}

// menuDef é um menu da barra: o título e os itens (Commands; ver
// MenuSeparator).
type menuDef struct {
	title string
	items []Command
}

// MenuSeparator devolve o item separador de grupos dentro de um menu.
func MenuSeparator() Command {
	return Command{Title: "-"}
}

// isSeparator informa se o item é o separador.
func isSeparator(c Command) bool {
	return c.Title == "-" && c.Action == nil
}

// NewMenuBar cria uma barra de menus vazia; adicione menus com Menu. O tema
// é herdado no mount.
func NewMenuBar() *MenuBar {
	return &MenuBar{open: -1, hover: -1}
}

// Menu acrescenta um menu com o título e os itens dados. Encadeável.
func (m *MenuBar) Menu(title string, items ...Command) *MenuBar {
	m.menus = append(m.menus, menuDef{title: title, items: items})
	m.Invalidate()
	return m
}

// attachSession registra os atalhos dos itens como comandos globais da
// sessão (idempotente: AddCommand substitui pelo título).
func (m *MenuBar) attachSession(s *Session) {
	m.BaseWidget.attachSession(s)
	for _, menu := range m.menus {
		for _, it := range menu.items {
			if !isSeparator(it) {
				s.AddCommand(it)
			}
		}
	}
}

// SetTheme define um tema explícito, como os containers.
func (m *MenuBar) SetTheme(th *theme.Theme) {
	m.BaseWidget.SetTheme(th)
	Mount(m, th)
}

// barHeight devolve a altura da barra em pixels.
func (m *MenuBar) barHeight() int {
	return m.theme.LineHeight() + 2*m.theme.Px(6)
}

// titleRect devolve o retângulo do título i na barra.
func (m *MenuBar) titleRect(i int) image.Rectangle {
	b := m.Bounds()
	pad := m.theme.PaddingPx()
	x := b.Min.X + m.theme.Px(4)
	for j := 0; j < i; j++ {
		x += m.theme.MeasureString(m.menus[j].title) + 2*pad
	}
	return image.Rect(x, b.Min.Y, x+m.theme.MeasureString(m.menus[i].title)+2*pad, b.Min.Y+m.barHeight())
}

// titleAt devolve o índice do título sob o ponto, ou -1.
func (m *MenuBar) titleAt(p image.Point) int {
	if m.theme == nil || !p.In(m.Bounds()) {
		return -1
	}
	for i := range m.menus {
		if p.In(m.titleRect(i)) {
			return i
		}
	}
	return -1
}

// PreferredSize devolve a largura dos títulos e a altura da barra.
func (m *MenuBar) PreferredSize() image.Point {
	if m.theme == nil {
		return image.Point{}
	}
	w := m.theme.Px(8)
	for i := range m.menus {
		w += m.theme.MeasureString(m.menus[i].title) + 2*m.theme.PaddingPx()
	}
	return image.Pt(w, m.barHeight())
}

// openMenu abre (ou troca para) o menu i.
func (m *MenuBar) openMenu(i int) {
	if i < 0 || i >= len(m.menus) || m.session == nil {
		return
	}
	if m.pane != nil {
		m.session.CloseOverlayIf(m.pane)
	}
	m.open = i
	r := m.titleRect(i)
	m.pane = newMenuPane(m, i, image.Pt(r.Min.X, r.Max.Y))
	m.session.OpenOverlay(m.pane)
	m.Invalidate()
}

// paneClosed avisa que o menu aberto fechou (a pane saiu da pilha).
func (m *MenuBar) paneClosed(p *menuPane) {
	if m.pane == p {
		m.pane = nil
		m.open = -1
		m.Invalidate()
	}
}

// step abre o menu vizinho (Esquerda/Direita com um menu aberto).
func (m *MenuBar) step(delta int) {
	if m.open < 0 {
		return
	}
	next := (m.open + delta + len(m.menus)) % len(m.menus)
	m.session.CloseOverlayIf(m.pane)
	m.openMenu(next)
}

// HandleEvent abre menus por clique e troca por hover enquanto um está
// aberto.
func (m *MenuBar) HandleEvent(ev event.Event) bool {
	e, ok := ev.(event.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case event.MouseMove, event.MouseEnter:
		i := m.titleAt(e.Pos)
		if i != m.hover {
			m.hover = i
			m.Invalidate()
		}
		return false
	case event.MouseLeave:
		if m.hover != -1 {
			m.hover = -1
			m.Invalidate()
		}
		return false
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft {
			return false
		}
		if i := m.titleAt(e.Pos); i >= 0 {
			m.openMenu(i)
		}
		return true
	case event.MouseUp:
		return true
	}
	return false
}

// Draw desenha a barra: fundo de superfície, fio inferior, títulos com
// pílula de hover e o aberto com a pílula de seleção.
func (m *MenuBar) Draw(dst *image.RGBA) {
	if m.theme == nil {
		return
	}
	th := m.theme
	b := m.Bounds()
	view := render.Clip(dst, b, &m.clip)
	render.FillRect(view, b, th.Surface)
	line := max(th.BorderPx(), 1)
	render.FillRect(view, image.Rect(b.Min.X, b.Max.Y-line, b.Max.X, b.Max.Y), th.SurfaceBorder)

	radius := th.RadiusPx()
	for i := range m.menus {
		r := m.titleRect(i)
		pill := r.Inset(th.Px(2))
		if i == m.open {
			render.FillRoundRect(view, pill, radius, th.Selection)
		} else if i == m.hover {
			render.FillRoundRect(view, pill, radius, th.HoverBackground)
		}
		y := r.Min.Y + (r.Dy()-th.LineHeight())/2 + th.Ascent()
		th.DrawText(view, m.menus[i].title, image.Pt(r.Min.X+th.PaddingPx(), y), th.Text)
	}
	m.drawDisabledOverlay(dst)
}

// menuPane é o painel suspenso de um menu aberto: itens com atalho à
// direita, separadores, navegação por teclado e execução com fechamento
// antes da ação. COBRE a janela (SpansWindow) para se comportar como menu
// de verdade: com um aberto, passar o ponteiro por outro título troca, e o
// clique fora fecha (engolido) — a caixa visível é só o painel ancorado.
type menuPane struct {
	BaseWidget
	bar       *MenuBar
	idx       int
	items     []Command
	anchor    image.Point
	panel     image.Rectangle
	highlight int
	clip      image.RGBA
}

// newMenuPane cria o painel do menu idx da barra, ancorado no ponto dado.
func newMenuPane(bar *MenuBar, idx int, anchor image.Point) *menuPane {
	return &menuPane{bar: bar, idx: idx, items: bar.menus[idx].items, anchor: anchor, highlight: -1}
}

// SpansWindow marca a pane como overlay de janela inteira (ver o godoc).
func (p *menuPane) SpansWindow() bool { return true }

// Layout guarda a janela inteira e ancora a caixa visível, presa aos
// limites.
func (p *menuPane) Layout(bounds image.Rectangle) {
	p.BaseWidget.Layout(bounds)
	th := p.bar.theme
	if th == nil {
		return
	}
	pref := p.prefSize(th)
	at := p.anchor
	if at.X+pref.X > bounds.Max.X {
		at.X = max(bounds.Max.X-pref.X, bounds.Min.X)
	}
	if at.Y+pref.Y > bounds.Max.Y {
		at.Y = max(bounds.Max.Y-pref.Y, bounds.Min.Y)
	}
	p.panel = image.Rectangle{Min: at, Max: at.Add(pref)}
}

// rowH devolve a altura de um item (ou de um separador).
func (p *menuPane) rowH(th *theme.Theme, sep bool) int {
	if sep {
		return th.SpacingPx()
	}
	return th.LineHeight() + 2*th.Px(th.RowPad)
}

// prefSize mede o painel: o item mais largo (título + atalho) e a soma das
// alturas.
func (p *menuPane) prefSize(th *theme.Theme) image.Point {
	w, h := 0, 2*th.Px(4)
	f := th.Caption()
	for _, it := range p.items {
		if isSeparator(it) {
			h += p.rowH(th, true)
			continue
		}
		iw := th.MeasureString(it.Title) + th.Px(24)
		if hint := it.ShortcutLabel(); hint != "" {
			iw += f.Measure(hint)
		}
		w = max(w, iw)
		h += p.rowH(th, false)
	}
	return image.Pt(max(w+2*th.Px(6), th.Px(160)), h)
}

// itemRect devolve o retângulo do item i, dentro da caixa visível.
func (p *menuPane) itemRect(i int) image.Rectangle {
	th := p.bar.theme
	b := p.panel
	y := b.Min.Y + th.Px(4)
	for j := 0; j < i; j++ {
		y += p.rowH(th, isSeparator(p.items[j]))
	}
	return image.Rect(b.Min.X+th.Px(4), y, b.Max.X-th.Px(4), y+p.rowH(th, isSeparator(p.items[i])))
}

// itemAt devolve o índice do item ACIONÁVEL sob o ponto, ou -1.
func (p *menuPane) itemAt(pos image.Point) int {
	for i := range p.items {
		if !isSeparator(p.items[i]) && pos.In(p.itemRect(i)) {
			return i
		}
	}
	return -1
}

// Focusable devolve true: o painel navega por teclado enquanto aberto.
func (p *menuPane) Focusable() bool { return true }

// run fecha o menu e executa o item.
func (p *menuPane) run(i int) {
	if i < 0 || i >= len(p.items) || isSeparator(p.items[i]) {
		return
	}
	cmd := p.items[i]
	p.bar.session.CloseOverlayIf(p)
	if cmd.Action != nil {
		cmd.Action()
	}
}

// moveHighlight desloca o realce pulando separadores.
func (p *menuPane) moveHighlight(delta int) {
	n := len(p.items)
	if n == 0 {
		return
	}
	i := p.highlight
	for range p.items {
		i = (i + delta + n) % n
		if !isSeparator(p.items[i]) {
			p.highlight = i
			p.Invalidate()
			return
		}
	}
}

// HandleEvent navega, aciona e fecha o menu.
func (p *menuPane) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.FocusEvent:
		if !e.Gained {
			// Perder o foco (Tab, clique fora tratado pela pilha) fecha.
			p.bar.session.CloseOverlayIf(p)
		}
		return true
	case event.KeyEvent:
		switch e.Key {
		case event.KeyDown:
			p.moveHighlight(1)
		case event.KeyUp:
			p.moveHighlight(-1)
		case event.KeyEnter, event.KeySpace:
			p.run(p.highlight)
		case event.KeyEscape:
			p.bar.session.CloseOverlayIf(p)
		case event.KeyLeft:
			p.bar.step(-1)
		case event.KeyRight:
			p.bar.step(1)
		default:
			return false
		}
		return true
	case event.MouseEvent:
		switch e.Kind {
		case event.MouseMove, event.MouseEnter:
			// Fora da caixa, passar por outro título da barra TROCA o menu
			// (comportamento de menu de verdade).
			if !e.Pos.In(p.panel) {
				if i := p.bar.titleAt(e.Pos); i >= 0 && i != p.idx {
					p.bar.openMenu(i)
					return true
				}
			}
			if i := p.itemAt(e.Pos); i != p.highlight {
				p.highlight = i
				p.Invalidate()
			}
			return true
		case event.MouseDown:
			if e.Pos.In(p.panel) {
				return true
			}
			// Clique fora fecha (engolido); no PRÓPRIO título, só fecha —
			// sem reabrir no mesmo gesto.
			p.bar.session.CloseOverlayIf(p)
			return true
		case event.MouseUp:
			if e.Button == event.MouseButtonLeft && e.Pos.In(p.panel) {
				p.run(p.itemAt(e.Pos))
			}
			return true
		}
	}
	return false
}

// Draw desenha a caixa visível (superfície com fio) e os itens — o resto
// da janela fica intocado (sem pano de fundo).
func (p *menuPane) Draw(dst *image.RGBA) {
	th := p.bar.theme
	if th == nil {
		return
	}
	b := p.panel
	radius := th.RadiusPx()
	render.FillRoundRect(dst, b, radius, th.Surface)
	render.StrokeRoundRect(dst, b, radius, th.BorderPx(), th.SurfaceBorder)
	view := render.Clip(dst, b, &p.clip)
	f := th.Caption()
	for i, it := range p.items {
		r := p.itemRect(i)
		if isSeparator(it) {
			line := max(th.BorderPx(), 1)
			y := r.Min.Y + (r.Dy()-line)/2
			render.FillRect(view, image.Rect(r.Min.X, y, r.Max.X, y+line), th.SurfaceBorder)
			continue
		}
		if i == p.highlight {
			render.FillRoundRect(view, r, radius, th.HoverBackground)
		}
		y := r.Min.Y + (r.Dy()-th.LineHeight())/2 + th.Ascent()
		th.DrawText(view, it.Title, image.Pt(r.Min.X+th.Px(6), y), th.Text)
		if hint := it.ShortcutLabel(); hint != "" {
			hw := f.Measure(hint)
			hy := r.Min.Y + (r.Dy()-f.LineHeight())/2 + f.Ascent()
			f.Draw(view, hint, image.Pt(r.Max.X-th.Px(6)-hw, hy), th.Placeholder)
		}
	}
}

// overlayClosed avisa a barra quando a pane sai da pilha, por qualquer
// caminho (Escape, clique fora, troca, execução).
func (p *menuPane) overlayClosed() { p.bar.paneClosed(p) }
