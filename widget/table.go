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
	// onChange é chamado quando o usuário seleciona uma linha (ver OnChange).
	onChange func(int)
	viewport image.Rectangle
	clip     image.RGBA
	// onReorder habilita a reordenação por arrasto (ver OnReorder); arm é o
	// estado do gesto entre o MouseDown e o limiar.
	onReorder func(from, to int)
	arm       reorderArm
}

// NewTable cria a tabela com os títulos de coluna, o total de linhas e o
// callback de célula. O tema é herdado no mount.
func NewTable(titles []string, count int, cell func(row, col int) string) *Table {
	return &Table{titles: titles, count: count, cell: cell}
}

// Widths define larguras lógicas por coluna (na ordem); 0 divide a sobra
// igualmente entre as colunas zeradas. Encadeável.
func (t *Table) Widths(widths ...int) *Table {
	t.widths = widths
	return t
}

// BindSelected habilita a seleção de linha e a vincula ao State (índice;
// -1 = nenhuma): clicar seleciona, um Set externo move o realce. Encadeável.
func (t *Table) BindSelected(s *state.State[int]) *Table {
	t.selected = s
	state.WatchWeak(s, t, func(t *Table, _ int) { t.Invalidate() })
	return t
}

// Selectable habilita a seleção de linha SEM State externo: a Table guarda
// o índice selecionado por conta própria (leia com Selected, reaja com
// OnChange). BindSelected, Select e OnChange já a habilitam. Encadeável.
func (t *Table) Selectable() *Table {
	if t.selected == nil {
		t.BindSelected(state.New(-1))
	}
	return t
}

// Selected devolve o índice da linha selecionada (-1 = nenhuma, ou seleção
// não habilitada).
func (t *Table) Selected() int {
	if t.selected == nil {
		return -1
	}
	return t.selected.Get()
}

// Select seleciona a linha i (-1 limpa; fora do intervalo é ignorado),
// habilitando a seleção se preciso. Como todo setter programático, não
// dispara OnChange.
func (t *Table) Select(i int) {
	t.Selectable()
	if i < -1 || i >= t.count || t.selected.Get() == i {
		return
	}
	t.selected.Set(i)
}

// OnChange define o callback chamado, com o índice, quando o USUÁRIO
// seleciona uma linha (depois do Set do State vinculado) e habilita a
// seleção. Encadeável.
func (t *Table) OnChange(fn func(index int)) *Table {
	t.Selectable()
	t.onChange = fn
	return t
}

// Count devolve o total de linhas.
func (t *Table) Count() int {
	return t.count
}

// SetCount muda o total de linhas (limpando a seleção fora do intervalo) e
// agenda um redesenho. Encadeável.
func (t *Table) SetCount(n int) *Table {
	if n < 0 {
		n = 0
	}
	if t.count == n {
		return t
	}
	t.count = n
	if t.selected != nil && t.selected.Get() >= n {
		t.selected.Set(-1)
	}
	t.Invalidate()
	return t
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

// RowRect devolve o retângulo absoluto da linha i (útil em testes). Antes
// do mount (sem tema, logo sem altura de linha), devolve o retângulo vazio.
func (t *Table) RowRect(i int) image.Rectangle {
	if t.theme == nil {
		return image.Rectangle{}
	}
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
	for i, title := range t.titles {
		colW := t.theme.MeasureString(title) + 2*t.theme.PaddingPx()
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
	fixed, flexible := 0, 0
	for i := range t.titles {
		if i < len(t.widths) && t.widths[i] > 0 {
			fixed += t.theme.Px(t.widths[i])
		} else {
			flexible++
		}
	}
	rest := max(b.Dx()-fixed, 0)
	d := distributor{leftover: rest, weightSum: flexible}
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
		header := t.viewport.Min.Y
		if t.viewport.Empty() {
			header = t.Bounds().Min.Y
		}
		if e.Pos.Y < header+t.rowH() {
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
				if t.onChange != nil {
					t.onChange(i)
				}
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

	// Linhas visíveis na região E dentro do destino: num redesenho parcial
	// (dst recortado ao dano), só as linhas tocadas são formatadas.
	rows := region.Intersect(dst.Bounds())
	first := max((rows.Min.Y-b.Min.Y-h)/h, 0)
	last := (rows.Max.Y - 1 - b.Min.Y - h) / h
	if last >= t.count {
		last = t.count - 1
	}
	if rows.Empty() {
		last = first - 1
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
			cellRect := image.Rect(t.colX[c], r.Min.Y, t.colX[c+1], r.Max.Y)
			view := render.Clip(dst, cellRect, &t.clip)
			th.DrawText(view, t.cell(i, c), image.Pt(cellRect.Min.X+th.PaddingPx(), baseline(r.Min.Y)), th.Text)
		}
		render.FillRect(dst, image.Rect(b.Min.X, r.Max.Y, b.Max.X, r.Max.Y+1), th.SurfaceBorder)
	}

	// Cabeçalho FIXO no topo da viewport, por cima das linhas roladas.
	headerTop := region.Min.Y
	header := image.Rect(b.Min.X, headerTop, b.Max.X, headerTop+h)
	render.FillRect(dst, header, th.HeaderBackground)
	// Títulos no papel de LEGENDA (menores que as células): a hierarquia
	// tipográfica do design system aplicada ao dado tabular.
	caption := th.Caption()
	capBase := header.Min.Y + (h-caption.LineHeight())/2 + caption.Ascent()
	for c, title := range t.titles {
		cellRect := image.Rect(t.colX[c], header.Min.Y, t.colX[c+1], header.Max.Y)
		view := render.Clip(dst, cellRect, &t.clip)
		caption.Draw(view, title, image.Pt(cellRect.Min.X+th.PaddingPx(), capBase), th.Text)
	}
	render.FillRect(dst, image.Rect(b.Min.X, header.Max.Y-1, b.Max.X, header.Max.Y), th.InputBorder)
	// Divisões verticais por toda a região visível.
	for c := 1; c < len(t.titles); c++ {
		render.FillRect(dst, image.Rect(t.colX[c], region.Min.Y, t.colX[c]+1, region.Max.Y), th.SurfaceBorder)
	}
}
