package widget

import (
	"image"
	"strings"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// ShowCommandPalette abre a paleta de comandos da sessão: um painel no
// topo da janela com um campo de busca e a lista dos comandos registrados
// (título + atalho). Digitar filtra, Cima/Baixo movem a seleção, Enter (ou
// clique) executa — a paleta fecha ANTES da ação rodar, então comandos que
// abrem diálogos funcionam. Escape e clique fora fecham. Com comandos
// registrados, Ctrl/Cmd+K chama isto sozinho.
func (s *Session) ShowCommandPalette() {
	if s.palette != nil || len(s.commands) == 0 {
		return
	}
	p := newPaletteView(s)
	th := s.theme
	w := min(th.Px(380), s.size.X-2*th.PaddingPx())
	h := min(th.Px(280), s.size.Y-2*th.PaddingPx())
	x := (s.size.X - w) / 2
	y := th.Px(48)
	p.Layout(image.Rect(x, y, x+w, y+h))
	s.palette = p
	s.OpenOverlay(p)
}

// cmdRow é uma linha da paleta: título à esquerda e o atalho à direita (em
// legenda apagada). Consome o clique para executar direto.
type cmdRow struct {
	BaseWidget
	title, hint string
	onClick     func()
	pressed     bool
}

func (r *cmdRow) PreferredSize() image.Point {
	if r.theme == nil {
		return image.Point{}
	}
	return image.Pt(r.theme.Px(200), r.theme.LineHeight())
}

func (r *cmdRow) CursorShape() CursorShape { return CursorHand }

func (r *cmdRow) HandleEvent(ev event.Event) bool {
	e, ok := ev.(event.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft {
			return false
		}
		r.pressed = true
		return true
	case event.MouseUp:
		if e.Button != event.MouseButtonLeft || !r.pressed {
			return false
		}
		r.pressed = false
		if e.Pos.In(r.Bounds()) && r.onClick != nil {
			r.onClick()
		}
		return true
	case event.MouseLeave:
		r.pressed = false
	}
	return false
}

func (r *cmdRow) Draw(dst *image.RGBA) {
	if r.theme == nil {
		return
	}
	th := r.theme
	b := r.Bounds()
	pad := th.Px(6)
	y := b.Min.Y + (b.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(dst, r.title, image.Pt(b.Min.X+pad, y), th.Text)
	if r.hint != "" {
		f := th.Caption()
		hw := f.Measure(r.hint)
		hy := b.Min.Y + (b.Dy()-f.LineHeight())/2 + f.Ascent()
		f.Draw(dst, r.hint, image.Pt(b.Max.X-pad-hw, hy), th.Placeholder)
	}
}

// paletteInput é o campo de busca da paleta: intercepta Cima/Baixo para
// mover a seleção da lista e delega o resto ao Input (padrão "decorar
// widget pronto").
type paletteInput struct {
	*Input
	pal *paletteView
}

func (pi *paletteInput) HandleEvent(ev event.Event) bool {
	if e, ok := ev.(event.KeyEvent); ok {
		switch e.Key {
		case event.KeyDown:
			pi.pal.move(1)
			return true
		case event.KeyUp:
			pi.pal.move(-1)
			return true
		}
	}
	return pi.Input.HandleEvent(ev)
}

// paletteView é o painel da paleta de comandos, exibido como overlay leve
// (sem pano de fundo; clique fora fecha pela regra da pilha).
type paletteView struct {
	BaseWidget
	session  *Session
	all      []Command
	filtered []int
	filtro   *state.State[string]
	selIdx   *state.State[int]
	lista    *List[*cmdRow]
	box      *VBox
	clip     image.RGBA
}

// newPaletteView monta a paleta com um retrato dos comandos registrados.
func newPaletteView(s *Session) *paletteView {
	p := &paletteView{
		session: s,
		all:     s.Commands(),
		filtro:  state.New(""),
		selIdx:  state.New(0),
	}
	p.lista = NewList(0,
		func() *cmdRow { return &cmdRow{} },
		func(row *cmdRow, i int) {
			if i < 0 || i >= len(p.filtered) {
				row.title, row.hint, row.onClick = "", "", nil
				return
			}
			cmd := p.all[p.filtered[i]]
			row.title, row.hint = cmd.Title, cmd.ShortcutLabel()
			row.onClick = func() { p.run(cmd) }
		},
	).BindSelected(p.selIdx)
	campo := &paletteInput{pal: p}
	campo.Input = NewInput("Comando…").BindValue(p.filtro).OnSubmit(p.runSelected)
	p.filtro.Watch(func(string) { p.refilter() })
	p.box = NewVBox(campo, NewSized(NewScroll(p.lista), 0, 200)).Gap(6)
	p.refilter()
	return p
}

// refilter recalcula os índices filtrados (busca case-insensitive por
// trecho do título) e volta a seleção ao topo.
func (p *paletteView) refilter() {
	q := strings.ToLower(strings.TrimSpace(p.filtro.Get()))
	p.filtered = p.filtered[:0]
	for i := range p.all {
		if q == "" || strings.Contains(strings.ToLower(p.all[i].Title), q) {
			p.filtered = append(p.filtered, i)
		}
	}
	p.lista.SetCount(len(p.filtered))
	p.lista.Refresh()
	if p.selIdx.Get() != 0 {
		p.selIdx.Set(0)
	}
	p.Invalidate()
}

// move desloca a seleção pelo teclado, presa ao intervalo.
func (p *paletteView) move(delta int) {
	if len(p.filtered) == 0 {
		return
	}
	i := p.selIdx.Get() + delta
	i = clampInt(i, 0, len(p.filtered)-1)
	if i != p.selIdx.Get() {
		p.selIdx.Set(i)
	}
}

// runSelected executa o comando selecionado (Enter no campo).
func (p *paletteView) runSelected() {
	i := p.selIdx.Get()
	if i < 0 || i >= len(p.filtered) {
		return
	}
	p.run(p.all[p.filtered[i]])
}

// run fecha a paleta ANTES de executar (a ação pode abrir um diálogo).
func (p *paletteView) run(cmd Command) {
	p.close()
	if cmd.Action != nil {
		cmd.Action()
	}
}

// close remove a paleta da pilha de overlays.
func (p *paletteView) close() {
	p.session.CloseOverlayIf(p)
}

// overlayClosed limpa o registro da sessão quando a paleta sai da pilha
// por qualquer caminho (Escape, clique fora, execução).
func (p *paletteView) overlayClosed() {
	if p.session.palette == p {
		p.session.palette = nil
	}
}

// Children devolve o miolo (campo + lista).
func (p *paletteView) Children() []Widget {
	return []Widget{p.box}
}

// Layout posiciona o miolo com o respiro do painel.
func (p *paletteView) Layout(bounds image.Rectangle) {
	p.BaseWidget.Layout(bounds)
	if p.theme == nil {
		return
	}
	p.box.Layout(bounds.Inset(p.theme.PaddingPx()))
}

// HandleEvent fecha com Escape e segura o mouse dentro do painel.
func (p *paletteView) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.KeyEvent:
		if e.Key == event.KeyEscape {
			p.close()
			return true
		}
	case event.MouseEvent:
		return e.Kind == event.MouseDown || e.Kind == event.MouseUp
	}
	return false
}

// Draw desenha o painel (superfície com fio, como um Card flutuante) e o
// miolo.
func (p *paletteView) Draw(dst *image.RGBA) {
	if p.theme == nil {
		return
	}
	th := p.theme
	b := p.Bounds()
	radius := th.RadiusPx()
	render.FillRoundRect(dst, b, radius, th.Surface)
	render.StrokeRoundRect(dst, b, radius, th.BorderPx(), th.SurfaceBorder)
	view := render.Clip(dst, b, &p.clip)
	p.box.Draw(view)
}
