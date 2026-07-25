package widget

import (
	"fmt"
	"image"
	"time"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// meses são os nomes exibidos no cabeçalho do Calendar.
var meses = [12]string{
	"janeiro", "fevereiro", "março", "abril", "maio", "junho",
	"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
}

// diasSemana são as iniciais da linha de cabeçalho (domingo primeiro).
var diasSemana = [7]string{"D", "S", "T", "Q", "Q", "S", "S"}

// diasStr são os rótulos 1..31 pré-formatados — Draw não aloca.
var diasStr = [32]string{
	"", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
	"13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23",
	"24", "25", "26", "27", "28", "29", "30", "31",
}

// Calendar é o seletor visual de data: um mês em grade (domingo primeiro),
// com ‹ › navegando os meses. Clicar num dia seleciona — a pílula do
// design system marca a seleção, o dia de hoje sai em Accent e os dias dos
// meses vizinhos completam a grade apagados (clicá-los navega e seleciona).
//
//	cal := juigo.NewCalendar().BindValue(data) // State[time.Time], duas vias
//
// O quick.Date abre um destes num popup pelo botão ao lado do campo.
type Calendar struct {
	BaseWidget
	// view é o primeiro dia do mês exibido.
	view   time.Time
	sel    time.Time
	bound  *state.State[time.Time]
	onPick func(time.Time)
	hover  int // célula sob o ponteiro (0..41; -1 = nenhuma)
	// titulo é o cabeçalho "mês ano" cacheado por mês exibido (Draw não
	// aloca).
	titulo     string
	tituloView time.Time
	clip       image.RGBA
}

// NewCalendar cria um calendário exibindo o mês atual. O tema é herdado no
// mount.
func NewCalendar() *Calendar {
	now := time.Now()
	return &Calendar{view: monthStart(now), hover: -1}
}

// monthStart devolve o primeiro dia do mês de t (mesma localidade).
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// BindValue vincula a data selecionada ao State em duas vias: clicar num
// dia faz Set, e um Set externo move a seleção (e o mês exibido).
// Encadeável.
func (c *Calendar) BindValue(s *state.State[time.Time]) *Calendar {
	c.bound = s
	c.adopt(s.Get())
	s.Watch(func(t time.Time) { c.adopt(t) })
	return c
}

// OnPick define o callback chamado quando um dia é clicado (depois do Set
// do State vinculado) — o gancho de popups que fecham na escolha.
// Encadeável.
func (c *Calendar) OnPick(fn func(time.Time)) *Calendar {
	c.onPick = fn
	return c
}

// Selected devolve a data selecionada (zero = nenhuma).
func (c *Calendar) Selected() time.Time {
	return c.sel
}

// adopt move a seleção (e o mês exibido) para t; zero limpa a seleção.
func (c *Calendar) adopt(t time.Time) {
	if t.IsZero() {
		if !c.sel.IsZero() {
			c.sel = time.Time{}
			c.Invalidate()
		}
		return
	}
	dia := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	if !c.sel.Equal(dia) || !c.view.Equal(monthStart(dia)) {
		c.sel = dia
		c.view = monthStart(dia)
		c.Invalidate()
	}
}

// pick seleciona a data, sincroniza o State e dispara OnPick.
func (c *Calendar) pick(t time.Time) {
	c.adopt(t)
	if c.bound != nil && !c.bound.Get().Equal(t) {
		c.bound.Set(t)
	}
	if c.onPick != nil {
		c.onPick(t)
	}
}

// step navega delta meses (mantendo a seleção onde está).
func (c *Calendar) step(delta int) {
	c.view = c.view.AddDate(0, delta, 0)
	c.Invalidate()
}

// Métricas da grade, em pixels.
func (c *Calendar) cellW() int   { return c.theme.Px(32) }
func (c *Calendar) cellH() int   { return c.theme.Px(26) }
func (c *Calendar) headerH() int { return c.theme.LineHeight() + 2*c.theme.Px(4) }
func (c *Calendar) weekH() int   { return c.theme.Caption().LineHeight() + c.theme.Px(4) }

// PreferredSize devolve a grade completa: cabeçalho, dias da semana e seis
// semanas.
func (c *Calendar) PreferredSize() image.Point {
	if c.theme == nil {
		return image.Point{}
	}
	return image.Pt(7*c.cellW(), c.headerH()+c.weekH()+6*c.cellH())
}

// gridStart devolve o primeiro dia EXIBIDO (domingo da primeira semana).
func (c *Calendar) gridStart() time.Time {
	return c.view.AddDate(0, 0, -int(c.view.Weekday()))
}

// cellRect devolve o retângulo da célula i (0..41) da grade.
func (c *Calendar) cellRect(i int) image.Rectangle {
	b := c.Bounds()
	top := b.Min.Y + c.headerH() + c.weekH()
	x := b.Min.X + (i%7)*c.cellW()
	y := top + (i/7)*c.cellH()
	return image.Rect(x, y, x+c.cellW(), y+c.cellH())
}

// cellAt devolve a célula sob o ponto, ou -1.
func (c *Calendar) cellAt(p image.Point) int {
	b := c.Bounds()
	top := b.Min.Y + c.headerH() + c.weekH()
	if !p.In(b) || p.Y < top {
		return -1
	}
	col := (p.X - b.Min.X) / c.cellW()
	row := (p.Y - top) / c.cellH()
	if col < 0 || col > 6 || row < 0 || row > 5 {
		return -1
	}
	return row*7 + col
}

// arrowRect devolve a área do botão de navegação (‹ esquerda, › direita).
func (c *Calendar) arrowRect(right bool) image.Rectangle {
	b := c.Bounds()
	w := c.cellW()
	if right {
		return image.Rect(b.Max.X-w, b.Min.Y, b.Max.X, b.Min.Y+c.headerH())
	}
	return image.Rect(b.Min.X, b.Min.Y, b.Min.X+w, b.Min.Y+c.headerH())
}

// CursorShape devolve a mãozinha (células e setas são clicáveis).
func (c *Calendar) CursorShape() CursorShape { return CursorHand }

// HandleEvent navega pelos ‹ ›, seleciona no clique e rastreia o hover.
func (c *Calendar) HandleEvent(ev event.Event) bool {
	e, ok := ev.(event.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case event.MouseMove, event.MouseEnter:
		if i := c.cellAt(e.Pos); i != c.hover {
			c.hover = i
			c.Invalidate()
		}
		return false
	case event.MouseLeave:
		if c.hover != -1 {
			c.hover = -1
			c.Invalidate()
		}
		return false
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft {
			return false
		}
		switch {
		case e.Pos.In(c.arrowRect(false)):
			c.step(-1)
		case e.Pos.In(c.arrowRect(true)):
			c.step(1)
		default:
			if i := c.cellAt(e.Pos); i >= 0 {
				c.pick(c.gridStart().AddDate(0, 0, i))
			}
		}
		return true
	case event.MouseUp:
		return true
	}
	return false
}

// Draw desenha o cabeçalho (‹ mês ›), as iniciais da semana e a grade.
func (c *Calendar) Draw(dst *image.RGBA) {
	if c.theme == nil {
		return
	}
	th := c.theme
	b := c.Bounds()
	view := render.Clip(dst, b, &c.clip)
	radius := th.RadiusPx()

	// Cabeçalho: setas e o título "mês ano".
	hy := b.Min.Y + (c.headerH()-th.LineHeight())/2 + th.Ascent()
	for _, right := range [2]bool{false, true} {
		r := c.arrowRect(right)
		s := "‹"
		if right {
			s = "›"
		}
		w := th.MeasureString(s)
		th.DrawText(view, s, image.Pt(r.Min.X+(r.Dx()-w)/2, hy), th.Text)
	}
	if !c.tituloView.Equal(c.view) {
		c.titulo = fmt.Sprintf("%s %d", meses[c.view.Month()-1], c.view.Year())
		c.tituloView = c.view
	}
	tw := th.MeasureString(c.titulo)
	th.DrawText(view, c.titulo, image.Pt(b.Min.X+(b.Dx()-tw)/2, hy), th.Text)

	// Iniciais da semana, em legenda apagada.
	f := th.Caption()
	wy := b.Min.Y + c.headerH() + (c.weekH()-f.LineHeight())/2 + f.Ascent()
	for i, d := range diasSemana {
		dw := f.Measure(d)
		x := b.Min.X + i*c.cellW() + (c.cellW()-dw)/2
		f.Draw(view, d, image.Pt(x, wy), th.Placeholder)
	}

	// Grade de dias: pílula de hover, pílula da seleção, hoje em Accent e
	// meses vizinhos apagados.
	hoje := time.Now()
	inicio := c.gridStart()
	for i := 0; i < 42; i++ {
		dia := inicio.AddDate(0, 0, i)
		r := c.cellRect(i)
		pill := r.Inset(th.Px(2))
		selecionado := !c.sel.IsZero() && dia.Equal(c.sel)
		if selecionado {
			render.FillRoundRect(view, pill, radius, th.Selection)
		} else if i == c.hover {
			render.FillRoundRect(view, pill, radius, th.HoverBackground)
		}
		cor := th.Text
		if dia.Month() != c.view.Month() {
			cor = th.Placeholder
		}
		if dia.Year() == hoje.Year() && dia.YearDay() == hoje.YearDay() && !selecionado {
			cor = th.Accent
		}
		s := diasStr[dia.Day()]
		sw := th.MeasureString(s)
		y := r.Min.Y + (r.Dy()-th.LineHeight())/2 + th.Ascent()
		th.DrawText(view, s, image.Pt(r.Min.X+(r.Dx()-sw)/2, y), cor)
	}
	c.drawDisabledOverlay(dst)
}
