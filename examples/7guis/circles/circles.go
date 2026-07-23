// Package circles é o 7GUIs nº 6: desenho de círculos com undo/redo.
// Clique esquerdo em área vazia cria um círculo; passar o mouse realça o
// círculo sob o cursor; clique DIREITO no realçado abre um diálogo com um
// slider que ajusta o diâmetro ao vivo — e o ajuste inteiro vira UMA entrada
// de undo ao fechar. A tela é um widget custom (BaseWidget + Draw +
// HandleEvent + Invalidate) e o undo/redo é o padrão de snapshots no modelo.
//
// Limitações registradas (ver ../GAPS.md): sem menu de contexto nativo (o
// diálogo é um Modal), sem StrokeCircle no render (o anel são dois
// FillCircle) e sem infra de undo na lib (snapshots à mão — candidato a
// state.History).
package circles

import (
	"fmt"
	"image"

	"juigo"
	"juigo/render"
)

// circulo é um círculo do desenho, em pixels da tela.
type circulo struct {
	x, y, r int
}

// Modelo guarda o desenho e as pilhas de undo/redo (snapshots por valor).
type Modelo struct {
	circulos []circulo
	passado  [][]circulo
	futuro   [][]circulo

	temPassado *juigo.State[bool]
	temFuturo  *juigo.State[bool]

	tela *tela
}

// snapshot registra o estado atual como ponto de undo e invalida o redo.
func (m *Modelo) snapshot() {
	m.passado = append(m.passado, append([]circulo(nil), m.circulos...))
	m.futuro = nil
	m.sincroniza()
}

// sincroniza espelha as pilhas nos estados dos botões e redesenha.
func (m *Modelo) sincroniza() {
	m.temPassado.Set(len(m.passado) > 0)
	m.temFuturo.Set(len(m.futuro) > 0)
	m.tela.Invalidate()
}

// Undo volta um passo.
func (m *Modelo) Undo() {
	if len(m.passado) == 0 {
		return
	}
	m.futuro = append(m.futuro, m.circulos)
	m.circulos = m.passado[len(m.passado)-1]
	m.passado = m.passado[:len(m.passado)-1]
	m.tela.hover = -1
	m.sincroniza()
}

// Redo repete o passo desfeito.
func (m *Modelo) Redo() {
	if len(m.futuro) == 0 {
		return
	}
	m.passado = append(m.passado, m.circulos)
	m.circulos = m.futuro[len(m.futuro)-1]
	m.futuro = m.futuro[:len(m.futuro)-1]
	m.tela.hover = -1
	m.sincroniza()
}

// Circulos devolve uma cópia do desenho (para testes).
func (m *Modelo) Circulos() []circulo {
	return append([]circulo(nil), m.circulos...)
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
	for i, c := range t.m.circulos {
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
			// Área vazia: cria um círculo com raio padrão.
			t.m.snapshot()
			t.m.circulos = append(t.m.circulos, circulo{x: e.Pos.X, y: e.Pos.Y, r: t.Theme().Px(18)})
			t.hover = len(t.m.circulos) - 1
			t.Invalidate()
			return true
		case e.Button == juigo.MouseButtonRight && t.sob(e.Pos) >= 0:
			t.abreAjuste(t.sob(e.Pos))
			return true
		}
	}
	return false
}

// abreAjuste abre o diálogo com o slider de diâmetro do círculo i; o ajuste
// inteiro vira UMA entrada de undo ao fechar (se o raio mudou).
func (t *tela) abreAjuste(i int) {
	antes := append([]circulo(nil), t.m.circulos...)
	c := t.m.circulos[i]
	diametro := juigo.NewState(float64(2 * c.r))
	diametro.Watch(func(v float64) {
		t.m.circulos[i].r = int(v / 2)
		t.Invalidate()
	})
	juigo.NewModal(juigo.NewVBox(
		juigo.NewText(fmt.Sprintf("Diâmetro do círculo em (%d, %d)", c.x, c.y)),
		juigo.NewSlider(8, 200).BindValue(diametro),
	)).OnClose(func() {
		if t.m.circulos[i].r != c.r {
			// Registra o ESTADO ANTERIOR como ponto de undo.
			t.m.passado = append(t.m.passado, antes)
			t.m.futuro = nil
			t.m.sincroniza()
		}
	}).Show()
}

func (t *tela) Draw(dst *image.RGBA) {
	th := t.Theme()
	if th == nil {
		return
	}
	b := t.Bounds()
	render.FillRect(dst, b, th.InputBackground)
	render.StrokeRect(dst, b, th.BorderPx(), th.InputBorder)
	for i, c := range t.m.circulos {
		centro := image.Pt(c.x, c.y)
		render.FillCircle(dst, centro, c.r, th.Text)
		miolo := th.InputBackground
		if i == t.hover {
			miolo = th.HoverBackground
		}
		render.FillCircle(dst, centro, c.r-th.BorderPx(), miolo)
	}
}

// New monta o modelo e a interface.
func New() (*Modelo, juigo.Widget) {
	m := &Modelo{
		temPassado: juigo.NewState(false),
		temFuturo:  juigo.NewState(false),
	}
	m.tela = &tela{m: m, hover: -1}

	nao := func(s *juigo.State[bool]) *juigo.State[bool] {
		return juigo.Map(s, func(v bool) bool { return !v })
	}
	ui := juigo.NewVBox(
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.BindDisabled(juigo.NewButton("Desfazer", m.Undo), nao(m.temPassado)),
			juigo.BindDisabled(juigo.NewButton("Refazer", m.Redo), nao(m.temFuturo)),
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
