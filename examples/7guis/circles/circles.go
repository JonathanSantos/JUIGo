// Package circles é o 7GUIs nº 6: desenho de círculos com undo/redo.
// Clique esquerdo em área vazia cria um círculo; passar o mouse realça o
// círculo sob o cursor; clique DIREITO no realçado abre um POPUP ANCORADO
// no ponto do clique (juigo.Popup) com um slider que ajusta o diâmetro ao
// vivo — e o ajuste inteiro vira UMA entrada de undo ao fechar. O histórico
// é um juigo.History (pilhas de desfazer/refazer com estados CanUndo/
// CanRedo prontos para os botões); o anel do círculo é um
// render.StrokeCircle.
package circles

import (
	"fmt"
	"image"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/render"
)

// circulo é um círculo do desenho, em pixels da tela.
type circulo struct {
	x, y, r int
}

// clone copia o desenho (o History não copia valores: slices mutáveis
// entram sempre clonados).
func clone(cs []circulo) []circulo {
	return append([]circulo(nil), cs...)
}

// Modelo guarda o desenho com histórico de desfazer/refazer.
type Modelo struct {
	hist *juigo.History[[]circulo]
	tela *tela
}

// Circulos devolve uma cópia do desenho (para testes).
func (m *Modelo) Circulos() []circulo {
	return clone(m.hist.Get())
}

// Undo volta um passo.
func (m *Modelo) Undo() {
	if _, ok := m.hist.Undo(); ok {
		m.tela.hover = -1
		m.tela.Invalidate()
	}
}

// Redo repete o passo desfeito.
func (m *Modelo) Redo() {
	if _, ok := m.hist.Redo(); ok {
		m.tela.hover = -1
		m.tela.Invalidate()
	}
}

// tela é o canvas custom do desenho.
type tela struct {
	juigo.BaseWidget
	m     *Modelo
	hover int // índice do círculo sob o cursor; -1 = nenhum
}

func (t *tela) PreferredSize() image.Point {
	th := t.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Point{X: th.Px(360), Y: th.Px(220)}
}

// sob devolve o índice do círculo sob o ponto (o de centro mais próximo,
// entre os que contêm o ponto), ou -1.
func (t *tela) sob(p image.Point) int {
	melhor, melhorD := -1, 0
	for i, c := range t.m.hist.Get() {
		dx, dy := p.X-c.x, p.Y-c.y
		d := dx*dx + dy*dy
		if d <= c.r*c.r && (melhor < 0 || d < melhorD) {
			melhor, melhorD = i, d
		}
	}
	return melhor
}

func (t *tela) HandleEvent(ev juigo.Event) bool {
	e, ok := ev.(juigo.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case juigo.MouseMove:
		if h := t.sob(e.Pos); h != t.hover {
			t.hover = h
			t.Invalidate()
		}
		return true
	case juigo.MouseLeave:
		if t.hover != -1 {
			t.hover = -1
			t.Invalidate()
		}
		return true
	case juigo.MouseDown:
		switch {
		case e.Button == juigo.MouseButtonLeft && t.sob(e.Pos) < 0:
			// Área vazia: cria um círculo (um ponto de undo).
			novo := append(clone(t.m.hist.Get()), circulo{x: e.Pos.X, y: e.Pos.Y, r: t.Theme().Px(18)})
			t.m.hist.Commit(novo)
			t.hover = len(novo) - 1
			t.Invalidate()
			return true
		case e.Button == juigo.MouseButtonRight && t.sob(e.Pos) >= 0:
			t.abreAjuste(t.sob(e.Pos), e.Pos)
			return true
		}
	}
	return false
}

// abreAjuste abre o popup ancorado no ponto do clique com o slider de
// diâmetro do círculo i; o ajuste inteiro vira UMA entrada de undo ao
// fechar (se o raio mudou).
func (t *tela) abreAjuste(i int, onde image.Point) {
	antes := clone(t.m.hist.Get())
	c := antes[i]
	diametro := juigo.NewState(float64(2 * c.r))
	diametro.Watch(func(v float64) {
		t.m.hist.Get()[i].r = int(v / 2) // ajuste ao vivo, sem histórico
		t.Invalidate()
	})
	juigo.NewPopup(juigo.NewVBox(
		juigo.NewText(fmt.Sprintf("Diâmetro do círculo em (%d, %d)", c.x, c.y)),
		juigo.NewSlider(8, 200).BindValue(diametro),
	)).OnClose(func() {
		if t.m.hist.Get()[i].r != c.r {
			// O estado ANTERIOR ao ajuste vira o ponto de undo.
			t.m.hist.CommitFrom(antes)
		}
	}).ShowAt(onde)
}

func (t *tela) Draw(dst *image.RGBA) {
	th := t.Theme()
	if th == nil {
		return
	}
	b := t.Bounds()
	render.FillRect(dst, b, th.InputBackground)
	render.StrokeRect(dst, b, th.BorderPx(), th.InputBorder)
	for i, c := range t.m.hist.Get() {
		centro := image.Pt(c.x, c.y)
		if i == t.hover {
			render.FillCircle(dst, centro, c.r, th.HoverBackground)
		}
		render.StrokeCircle(dst, centro, c.r, th.BorderPx(), th.Text)
	}
}

// New monta o modelo e a interface.
func New() (*Modelo, juigo.Widget) {
	m := &Modelo{hist: juigo.NewHistory([]circulo{})}
	m.tela = &tela{m: m, hover: -1}

	ui := juigo.NewVBox(
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.BindDisabled(juigo.NewButton("Desfazer", m.Undo), juigo.Not(m.hist.CanUndo())),
			juigo.BindDisabled(juigo.NewButton("Refazer", m.Redo), juigo.Not(m.hist.CanRedo())),
			juigo.NewSpacer(),
		),
		juigo.Grow(m.tela, 1),
	).Pad(16)
	return m, ui
}

// UI monta a tela (conveniência para o launcher).
func UI() juigo.Widget {
	_, w := New()
	return w
}
