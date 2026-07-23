package widget

import "image"

// List é uma lista VIRTUALIZADA de linhas uniformes: das count linhas
// lógicas, só as visíveis existem como widgets — um pool pequeno de linhas é
// criado por criar() e RECICLADO conforme a rolagem, com vincular(linha, i)
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

	criar    func() W
	vincular func(row W, index int)
	count    int

	rowH      int
	rowW      int
	rowSized  bool
	pool      []W
	children  []Widget
	poolStart int
	poolLen   int
	poolTop   int
	poolValid bool
	// viewport é informada pelo Scroll (SetViewport) para o pool cobrir
	// tudo que está visível — não apenas a região suja de um frame parcial.
	viewport image.Rectangle
}

// NewList cria uma lista virtualizada com count linhas. O tema é herdado no
// mount.
func NewList[W Widget](count int, criar func() W, vincular func(row W, index int)) *List[W] {
	return &List[W]{criar: criar, vincular: vincular, count: count}
}

// Count devolve o total de linhas lógicas.
func (l *List[W]) Count() int {
	return l.count
}

// SetCount muda o total de linhas e agenda revinculação e redesenho.
func (l *List[W]) SetCount(n int) {
	if n < 0 {
		n = 0
	}
	if l.count == n {
		return
	}
	l.count = n
	l.Refresh()
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
	if l.rowSized {
		return true
	}
	sample := l.criar()
	Mount(sample, l.theme)
	if l.count > 0 {
		l.vincular(sample, 0)
	}
	p := sample.PreferredSize()
	if p.Y <= 0 {
		return false
	}
	l.rowW, l.rowH = p.X, p.Y
	l.rowSized = true
	// A amostra vira a primeira linha do pool (nada é jogado fora).
	l.pool = append(l.pool, sample)
	return true
}

// PreferredSize devolve a largura da amostra e count × altura de linha — o
// "tamanho lógico" que o Scroll usa para rolar.
func (l *List[W]) PreferredSize() image.Point {
	if !l.ensureRowSize() {
		return image.Point{}
	}
	return image.Point{X: l.rowW, Y: l.count * l.rowH}
}

// rowRect devolve o retângulo absoluto da linha lógica i.
func (l *List[W]) rowRect(i int) image.Rectangle {
	b := l.Bounds()
	top := b.Min.Y + i*l.rowH
	return image.Rect(b.Min.X, top, b.Max.X, top+l.rowH)
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
		row := l.criar()
		Mount(row, l.theme)
		l.pool = append(l.pool, row)
	}
	l.children = l.children[:0]
	for k := 0; k < n; k++ {
		row := l.pool[k]
		l.vincular(row, first+k)
		row.Layout(l.rowRect(first + k))
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
	for _, row := range l.children {
		if row.Bounds().Overlaps(dst.Bounds()) {
			row.Draw(dst)
		}
	}
}
