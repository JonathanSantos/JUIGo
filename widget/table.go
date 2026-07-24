package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// Table é a tabela de células de TEXTO: colunas com título, linhas
// uniformes desenhadas sob demanda (só as visíveis) e seleção opcional de
// linha. As células vêm de um callback — a tabela não guarda dados:
//
//	tabela := juigo.NewTable([]string{"Nome", "Sobrenome"}, len(pessoas),
//	    func(linha, coluna int) string { … },
//	).BindSelected(selecionada)
//	juigo.Grow(juigo.NewScroll(tabela), 1)
//
// Dentro de um Scroll, o CABEÇALHO FICA FIXO: ele é desenhado no topo da
// viewport (informada pelo Scroll no layout), por cima das linhas roladas.
// Depois de mudar os dados, chame Refresh; o total, SetCount. Células são
// texto puro (recortadas à célula) — conteúdo rico fica para a composição
// com List ou widgets custom.
type Table struct {
	BaseWidget

	titles []string
	count  int
	cell   func(row, col int) string

	// widths são larguras LÓGICAS por coluna; 0 divide a sobra igualmente
	// (ver Widths). colX é o cache das bordas de coluna do último Draw.
	widths []int
	colX   []int

	selected *state.State[int]
	viewport image.Rectangle
	clip     image.RGBA
	// onReorder habilita a reordenação por arrasto (ver OnReorder); arm é o
	// estado do gesto entre o MouseDown e o limiar.
	onReorder func(de, para int)
	arm       reorderArm
}

// NewTable cria a tabela com os títulos de coluna, o total de linhas e o
// callback de célula. O tema é herdado no mount.
func NewTable(titles []string, count int, cell func(row, col int) string) *Table {
	return &Table{titles: titles, count: count, cell: cell}
}

// Widths define larguras lógicas por coluna (na ordem); 0 divide a sobra
// igualmente entre as colunas zeradas. Encadeável.
func (t *Table) Widths(larguras ...int) *Table {
	t.widths = larguras
	return t
}

// BindSelected habilita a seleção de linha e a vincula ao State (índice;
// -1 = nenhuma): clicar seleciona, um Set externo move o realce. Encadeável.
func (t *Table) BindSelected(s *state.State[int]) *Table {
	t.selected = s
	s.Watch(func(int) { t.Invalidate() })
	return t
}

// Count devolve o total de linhas.
func (t *Table) Count() int {
	return t.count
}

// SetCount muda o total de linhas (limpando a seleção fora do intervalo) e
// agenda um redesenho.
func (t *Table) SetCount(n int) {
	if n < 0 {
		n = 0
	}
	if t.count == n {
		return
	}
	t.count = n
	if t.selected != nil && t.selected.Get() >= n {
		t.selected.Set(-1)
	}
	t.Invalidate()
}

// Refresh agenda um redesenho (chame após mudar os DADOS das células).
func (t *Table) Refresh() {
	t.Invalidate()
}

// SetViewport recebe a região visível do Scroll: o cabeçalho é desenhado no
// topo dela e só as linhas visíveis são pintadas.
func (t *Table) SetViewport(r image.Rectangle) {
	t.viewport = r
}

// rowH devolve a altura de linha (e do cabeçalho).
func (t *Table) rowH() int {
	return t.theme.LineHeight() + t.theme.Px(6)
}

// RowRect devolve o retângulo absoluto da linha i (útil em testes).
func (t *Table) RowRect(i int) image.Rectangle {
	b := t.Bounds()
	h := t.rowH()
	top := b.Min.Y + h + i*h // primeira linha abaixo do cabeçalho
	return image.Rect(b.Min.X, top, b.Max.X, top+h)
}

// PreferredSize soma as larguras (títulos medidos, com padding) e as linhas
// mais o cabeçalho.
func (t *Table) PreferredSize() image.Point {
	if t.theme == nil {
		return image.Point{}
	}
	w := 0
	for i, título := range t.titles {
		colW := t.theme.MeasureString(título) + 2*t.theme.PaddingPx()
		if i < len(t.widths) && t.widths[i] > 0 {
			colW = t.theme.Px(t.widths[i])
		}
		w += colW
	}
	return image.Point{X: w, Y: (t.count + 1) * t.rowH()}
}

// resolveCols recalcula as bordas de coluna para a largura atual: colunas
// com largura fixa primeiro, a sobra dividida igualmente entre as demais.
func (t *Table) resolveCols() {
	b := t.Bounds()
	n := len(t.titles)
	t.colX = resizeInts(t.colX, n+1)
	fixa, flexiveis := 0, 0
	for i := range t.titles {
		if i < len(t.widths) && t.widths[i] > 0 {
			fixa += t.theme.Px(t.widths[i])
		} else {
			flexiveis++
		}
	}
	sobra := b.Dx() - fixa
	if sobra < 0 {
		sobra = 0
	}
	d := distributor{leftover: sobra, weightSum: flexiveis}
	x := b.Min.X
	for i := range t.titles {
		t.colX[i] = x
		if i < len(t.widths) && t.widths[i] > 0 {
			x += t.theme.Px(t.widths[i])
		} else {
			x += d.next(1)
		}
	}
	t.colX[n] = x
}

// HandleEvent seleciona a linha sob o clique (ignorando o cabeçalho fixo)
// e arma/dispara a reordenação por arrasto (OnReorder).
func (t *Table) HandleEvent(ev event.Event) bool {
	if (t.selected == nil && t.onReorder == nil) || t.theme == nil {
		return false
	}
	e, ok := ev.(event.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft {
			return false
		}
		// Clique sobre o cabeçalho fixo não seleciona nem arma.
		cab := t.viewport.Min.Y
		if t.viewport.Empty() {
			cab = t.Bounds().Min.Y
		}
		if e.Pos.Y < cab+t.rowH() {
			return true
		}
		i := (e.Pos.Y - t.Bounds().Min.Y - t.rowH()) / t.rowH()
		if i < 0 || i >= t.count {
			return false
		}
		if t.onReorder != nil {
			t.arm.arm(i, e.Pos)
		}
		if t.selected != nil {
			if t.selected.Get() != i {
				t.selected.Set(i)
			}
			t.Invalidate()
		}
		return true
	case event.MouseMove:
		if i, fired := t.arm.fire(e.Pos, dragThreshold(t.theme)); fired {
			StartDrag(reorderPayload{owner: t, from: i}, t.cell(i, 0))
		}
		return false
	case event.MouseUp, event.MouseLeave:
		t.arm.disarm()
		return false
	}
	return false
}

// Draw pinta as linhas visíveis (com o realce da selecionada), as divisões
// e o cabeçalho fixo no topo da viewport, por cima das linhas roladas.
func (t *Table) Draw(dst *image.RGBA) {
	if t.theme == nil || len(t.titles) == 0 {
		return
	}
	th := t.theme
	b := t.Bounds()
	h := t.rowH()
	t.resolveCols()

	region := t.viewport
	if region.Empty() {
		region = dst.Bounds()
	}
	region = region.Intersect(b)

	// Linhas visíveis na região.
	first := (region.Min.Y - b.Min.Y - h) / h
	if first < 0 {
		first = 0
	}
	last := (region.Max.Y - 1 - b.Min.Y - h) / h
	if last >= t.count {
		last = t.count - 1
	}
	sel := -1
	if t.selected != nil {
		sel = t.selected.Get()
	}
	baseline := func(top int) int { return top + (h-th.LineHeight())/2 + th.Ascent() }
	for i := first; i <= last; i++ {
		r := t.RowRect(i)
		if i == sel {
			render.FillRect(dst, r, th.Selection)
		}
		for c := range t.titles {
			cel := image.Rect(t.colX[c], r.Min.Y, t.colX[c+1], r.Max.Y)
			view := render.Clip(dst, cel, &t.clip)
			th.DrawText(view, t.cell(i, c), image.Pt(cel.Min.X+th.PaddingPx(), baseline(r.Min.Y)), th.Text)
		}
		render.FillRect(dst, image.Rect(b.Min.X, r.Max.Y, b.Max.X, r.Max.Y+1), th.InputBorder)
	}

	// Cabeçalho FIXO no topo da viewport, por cima das linhas roladas.
	cabTop := region.Min.Y
	cab := image.Rect(b.Min.X, cabTop, b.Max.X, cabTop+h)
	render.FillRect(dst, cab, th.HoverBackground)
	for c, título := range t.titles {
		cel := image.Rect(t.colX[c], cab.Min.Y, t.colX[c+1], cab.Max.Y)
		view := render.Clip(dst, cel, &t.clip)
		th.DrawText(view, título, image.Pt(cel.Min.X+th.PaddingPx(), baseline(cab.Min.Y)), th.Text)
	}
	render.FillRect(dst, image.Rect(b.Min.X, cab.Max.Y-1, b.Max.X, cab.Max.Y), th.InputBorder)
	// Divisões verticais por toda a região visível.
	for c := 1; c < len(t.titles); c++ {
		render.FillRect(dst, image.Rect(t.colX[c], region.Min.Y, t.colX[c]+1, region.Max.Y), th.InputBorder)
	}
}
