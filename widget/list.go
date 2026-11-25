package widget

import (
	"image"

	"golang.org/x/image/font"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// List é uma lista VIRTUALIZADA de linhas uniformes: das count linhas
// lógicas, só as visíveis existem como widgets — um pool pequeno de linhas é
// criado por create() e RECICLADO conforme a rolagem, com bind(row, i)
// preenchendo cada uma com os dados do índice. Combine com Scroll:
//
//	lista := juigo.NewList(len(itens),
//	    func() *juigo.Text { return juigo.NewText("") },
//	    func(t *juigo.Text, i int) { t.SetText(itens[i]) },
//	)
//	juigo.Grow(juigo.NewScroll(lista), 1)
//
// O tamanho preferido é count × altura de uma linha de amostra (linhas devem
// ter altura uniforme) — o Scroll enxerga o conteúdo inteiro como um
// retângulo e rola normalmente, mas apenas as linhas visíveis são
// vinculadas, posicionadas e desenhadas. Eventos (clique, hover) chegam às
// linhas do pool pelo roteamento normal; linhas não devem ser focáveis
// (seriam recicladas com o foco). Depois de mudar os DADOS, chame Refresh;
// depois de mudar o total, SetCount.
type List[W Widget] struct {
	BaseWidget

	create func() W
	bind   func(row W, index int)
	count  int

	rowH     int
	rowW     int
	rowSized bool
	// rowFace é a face vigente quando a linha foi medida: muda em troca de
	// tema, de escala ou de tamanho de fonte — e a altura re-mede.
	rowFace   font.Face
	pool      []W
	children  []Widget
	poolStart int
	poolLen   int
	poolTop   int
	poolValid bool
	// viewport é informada pelo Scroll (SetViewport) para o pool cobrir
	// tudo que está visível — não apenas a região suja de um frame parcial.
	viewport image.Rectangle
	// selected habilita a seleção de linha (ver BindSelected).
	selected *state.State[int]
	// onChange é chamado quando o usuário seleciona uma linha (ver OnChange).
	onChange func(int)
	// hoverRow é a linha sob o ponteiro (-1 = nenhuma), realçada com a
	// pílula de hover quando a lista é selecionável.
	hoverRow int
	// reorderLabel/onReorder habilitam a reordenação por arrasto (ver
	// OnReorder); arm é o estado do gesto entre o MouseDown e o limiar.
	reorderLabel func(index int) string
	onReorder    func(from, to int)
	arm          reorderArm
}

// NewList cria uma lista virtualizada com count linhas. O tema é herdado no
// mount.
func NewList[W Widget](count int, create func() W, bind func(row W, index int)) *List[W] {
	return &List[W]{create: create, bind: bind, count: count, hoverRow: -1}
}

// Count devolve o total de linhas lógicas.
func (l *List[W]) Count() int {
	return l.count
}

// BindSelected habilita a seleção de linha e a vincula ao State (índice
// LÓGICO; -1 = nenhuma): clicar numa linha faz Set, um Set externo move o
// realce, e a linha selecionada ganha o fundo Theme.Selection. Linhas que
// consomem o próprio clique (botões) não selecionam — o clique só chega à
// List quando a linha o deixa passar (Text puro, por exemplo). Encadeável.
func (l *List[W]) BindSelected(s *state.State[int]) *List[W] {
	l.selected = s
	state.WatchWeak(s, l, func(l *List[W], _ int) { l.Invalidate() })
	return l
}

// Selectable habilita a seleção de linha SEM State externo: a List guarda
// o índice selecionado por conta própria (leia com Selected, reaja com
// OnChange). BindSelected, Select e OnChange já a habilitam. Encadeável.
func (l *List[W]) Selectable() *List[W] {
	if l.selected == nil {
		l.BindSelected(state.New(-1))
	}
	return l
}

// Selected devolve o índice da linha selecionada (-1 = nenhuma, ou seleção
// não habilitada).
func (l *List[W]) Selected() int {
	if l.selected == nil {
		return -1
	}
	return l.selected.Get()
}

// Select seleciona a linha i (-1 limpa; fora do intervalo é ignorado),
// habilitando a seleção se preciso. Como todo setter programático, não
// dispara OnChange.
func (l *List[W]) Select(i int) {
	l.Selectable()
	if i < -1 || i >= l.count || l.selected.Get() == i {
		return
	}
	l.selected.Set(i)
}

// OnChange define o callback chamado, com o índice, quando o USUÁRIO
// seleciona uma linha (depois do Set do State vinculado) e habilita a
// seleção. Encadeável.
func (l *List[W]) OnChange(fn func(index int)) *List[W] {
	l.Selectable()
	l.onChange = fn
	return l
}

// HandleEvent seleciona a linha sob o clique (BindSelected) e arma/dispara
// a reordenação por arrasto (OnReorder).
func (l *List[W]) HandleEvent(ev event.Event) bool {
	if l.selected == nil && l.onReorder == nil {
		return false
	}
	e, ok := ev.(event.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case event.MouseDown:
		if e.Button != event.MouseButtonLeft || !l.rowSized || l.rowH <= 0 {
			return false
		}
		i := (e.Pos.Y - l.Bounds().Min.Y) / l.rowH
		if i < 0 || i >= l.count {
			return false
		}
		if l.onReorder != nil {
			l.arm.arm(i, e.Pos)
		}
		if l.selected != nil {
			if l.selected.Get() != i {
				l.selected.Set(i)
				if l.onChange != nil {
					l.onChange(i)
				}
			}
			l.Invalidate()
		}
		return true
	case event.MouseMove, event.MouseEnter:
		if e.Kind == event.MouseMove {
			if i, fired := l.arm.fire(e.Pos, dragThreshold(l.theme)); fired {
				label := ""
				if l.reorderLabel != nil {
					label = l.reorderLabel(i)
				}
				StartDrag(reorderPayload{owner: l, from: i}, label)
			}
		}
		if l.selected != nil && l.rowSized && l.rowH > 0 {
			i := (e.Pos.Y - l.Bounds().Min.Y) / l.rowH
			if i < 0 || i >= l.count {
				i = -1
			}
			l.setHoverRow(i)
		}
		return false
	case event.MouseUp:
		l.arm.disarm()
		return false
	case event.MouseLeave:
		l.arm.disarm()
		l.setHoverRow(-1)
		return false
	}
	return false
}

// setHoverRow move o realce de hover, danificando só as linhas afetadas.
func (l *List[W]) setHoverRow(i int) {
	if i == l.hoverRow {
		return
	}
	if l.hoverRow >= 0 {
		l.damage(l.rowRect(l.hoverRow))
	}
	l.hoverRow = i
	if i >= 0 {
		l.damage(l.rowRect(i))
	}
}

// SetCount muda o total de linhas e agenda revinculação e redesenho.
// Encadeável.
func (l *List[W]) SetCount(n int) *List[W] {
	if n < 0 {
		n = 0
	}
	if l.count == n {
		return l
	}
	l.count = n
	// A seleção não sobrevive fora do novo intervalo.
	if l.selected != nil && l.selected.Get() >= n {
		l.selected.Set(-1)
	}
	l.Refresh()
	return l
}

// Refresh revincula as linhas visíveis (chame após mudar os dados) e agenda
// um redesenho.
func (l *List[W]) Refresh() {
	l.poolValid = false
	l.Invalidate()
}

// SetViewport informa a região visível (o Scroll a chama no layout); o pool
// de linhas é vinculado imediatamente para cobri-la — antes de eventos e
// desenho.
func (l *List[W]) SetViewport(r image.Rectangle) {
	l.viewport = r
	l.ensure(r)
}

// Children devolve as linhas do pool atualmente vinculadas (roteamento,
// mount e hover funcionam nelas normalmente).
func (l *List[W]) Children() []Widget {
	return l.children
}

// ensureRowSize mede uma linha de amostra para descobrir a altura uniforme.
func (l *List[W]) ensureRowSize() bool {
	if l.theme == nil {
		return false
	}
	if l.rowSized && l.rowFace == l.theme.Face {
		return true
	}
	// Primeira medição OU o tema mudou de métrica (escala, tamanho de
	// fonte, troca de tema): re-mede com uma amostra e invalida o pool.
	fresh := len(l.pool) == 0
	var sample W
	if fresh {
		sample = l.create()
	} else {
		sample = l.pool[0]
	}
	Mount(sample, l.theme)
	if l.count > 0 {
		l.bind(sample, 0)
	}
	p := sample.PreferredSize()
	if p.Y <= 0 {
		return false
	}
	// As linhas respiram: conteúdo mais o respiro do tema (Theme.RowPad)
	// em cima e embaixo; o widget da linha centraliza no espaço.
	l.rowW, l.rowH = p.X, p.Y+2*l.theme.Px(l.theme.RowPad)
	l.rowSized = true
	l.rowFace = l.theme.Face
	l.poolValid = false
	if fresh {
		// A amostra vira a primeira linha do pool (nada é jogado fora).
		l.pool = append(l.pool, sample)
	}
	return true
}

// PreferredSize devolve a largura da amostra e count × altura de linha — o
// "tamanho lógico" que o Scroll usa para rolar.
func (l *List[W]) PreferredSize() image.Point {
	if !l.ensureRowSize() {
		return image.Point{}
	}
	w := l.rowW
	if l.selected != nil && l.theme != nil {
		// O conteúdo nasce recuado dentro da pílula (ver contentRect).
		w += 2 * (l.theme.Px(l.theme.RowPad) + l.theme.PaddingPx())
	}
	return image.Point{X: w, Y: l.count * l.rowH}
}

// rowRect devolve o retângulo absoluto da linha lógica i.
func (l *List[W]) rowRect(i int) image.Rectangle {
	b := l.Bounds()
	top := b.Min.Y + i*l.rowH
	return image.Rect(b.Min.X, top, b.Max.X, top+l.rowH)
}

// pillRect devolve o retângulo da pílula de realce da linha i: a linha com
// a margem horizontal do design system (Theme.RowPad).
func (l *List[W]) pillRect(i int) image.Rectangle {
	r := l.rowRect(i)
	pad := l.theme.Px(l.theme.RowPad)
	r.Min.X += pad
	r.Max.X -= pad
	return r
}

// contentRect devolve onde o conteúdo da linha i é posicionado. Em listas
// selecionáveis o conteúdo nasce DENTRO da pílula de realce (margem RowPad
// mais o Padding do tema), nunca colado na borda dela; nas demais ocupa a
// linha inteira.
func (l *List[W]) contentRect(i int) image.Rectangle {
	r := l.rowRect(i)
	if l.selected == nil || l.theme == nil {
		return r
	}
	in := l.theme.Px(l.theme.RowPad) + l.theme.PaddingPx()
	r.Min.X += in
	r.Max.X = max(r.Max.X-in, r.Min.X)
	return r
}

// ensure vincula e posiciona o pool para cobrir a região visível dada.
func (l *List[W]) ensure(visible image.Rectangle) {
	if !l.ensureRowSize() || l.count == 0 {
		l.children = l.children[:0]
		l.poolLen = 0
		return
	}
	b := l.Bounds()
	visible = visible.Intersect(b)
	if visible.Empty() {
		return
	}
	first := (visible.Min.Y - b.Min.Y) / l.rowH
	last := (visible.Max.Y - 1 - b.Min.Y) / l.rowH
	if first < 0 {
		first = 0
	}
	if last >= l.count {
		last = l.count - 1
	}
	n := last - first + 1
	if n <= 0 {
		return
	}
	if l.poolValid && first == l.poolStart && n == l.poolLen && b.Min.Y == l.poolTop {
		return // pool já cobre a região, nas mesmas posições
	}
	for len(l.pool) < n {
		row := l.create()
		Mount(row, l.theme)
		l.pool = append(l.pool, row)
	}
	l.children = l.children[:0]
	for k := range n {
		row := l.pool[k]
		l.bind(row, first+k)
		row.Layout(l.contentRect(first + k))
		l.children = append(l.children, row)
	}
	l.poolStart, l.poolLen, l.poolTop, l.poolValid = first, n, b.Min.Y, true
}

// Draw garante o pool sobre a viewport (informada pelo Scroll; sem Scroll,
// os bounds do destino) e desenha as linhas que intersectam o destino.
func (l *List[W]) Draw(dst *image.RGBA) {
	region := l.viewport
	if region.Empty() {
		region = dst.Bounds()
	}
	l.ensure(region)
	// Realces como PÍLULAS com margem (design system), por baixo do
	// conteúdo: hover primeiro, seleção por cima.
	if l.selected != nil && l.theme != nil && l.rowH > 0 {
		radius := l.theme.RadiusPx()
		sel := l.selected.Get()
		if l.hoverRow >= 0 && l.hoverRow < l.count && l.hoverRow != sel {
			if r := l.pillRect(l.hoverRow); r.Overlaps(dst.Bounds()) {
				render.FillRoundRect(dst, r, radius, l.theme.HoverBackground)
			}
		}
		if sel >= 0 && sel < l.count {
			if r := l.pillRect(sel); r.Overlaps(dst.Bounds()) {
				render.FillRoundRect(dst, r, radius, l.theme.Selection)
			}
		}
	}
	for _, row := range l.children {
		if row.Bounds().Overlaps(dst.Bounds()) {
			row.Draw(dst)
		}
	}
}
