package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
)

// Tree é uma árvore VIRTUALIZADA de nós expansíveis. O modelo fica com a
// aplicação, consultado por callbacks: roots devolve os nós de primeiro
// nível, children os filhos de um nó (vazio = folha) — ambos por um ID
// COMPARÁVEL seu (caminho, chave, ponteiro). Como na List, só as linhas
// visíveis existem como widgets: um pool criado por criar() é reciclado
// conforme a rolagem, com vincular(linha, id) preenchendo cada uma.
//
//	tree := juigo.NewTree(
//	    func() []string { return []string{"/"} },
//	    func(dir string) []string { return subdirs(dir) },
//	    func() *juigo.Text { return juigo.NewText("") },
//	    func(t *juigo.Text, dir string) { t.SetText(filepath.Base(dir)) },
//	)
//	juigo.Grow(juigo.NewScroll(tree), 1)
//
// A Tree desenha o recuo por nível (Theme.TreeIndent) e o chevron de
// expandir/recolher; clique no chevron (ou duplo clique na linha) alterna o
// nó. É focável: Cima/Baixo movem a seleção pelas linhas visíveis, Direita
// expande (ou desce ao primeiro filho), Esquerda recolhe (ou sobe ao pai),
// Enter ativa a folha (OnActivate). BindSelected vincula a seleção a um
// State[ID] — o valor ZERO de ID significa "nenhum". Após mudar o modelo,
// chame Refresh.
type Tree[ID comparable, W Widget] struct {
	BaseWidget

	roots      func() []ID
	childrenOf func(ID) []ID
	criar      func() W
	vincular   func(row W, id ID)

	expanded map[ID]bool
	// rows é o achatamento dos nós VISÍVEIS (expandidos), reconstruído sob
	// demanda; maxDepth alimenta o PreferredSize.
	rows      []treeRow[ID]
	rowsValid bool
	maxDepth  int

	rowH, rowW int
	rowSized   bool
	pool       []W
	kids       []Widget
	poolStart  int
	poolLen    int
	poolTop    int
	poolValid  bool
	viewport   image.Rectangle

	selected   *state.State[ID]
	onActivate func(ID)
	focused    bool
	dbl        doubleClick
}

// treeRow é uma linha visível do achatamento: o nó, a profundidade e o que
// o chevron deve mostrar.
type treeRow[ID comparable] struct {
	id    ID
	depth int
	leaf  bool
	open  bool
}

// NewTree cria uma árvore virtualizada sobre o modelo dado (ver Tree). O
// tema é herdado no mount.
func NewTree[ID comparable, W Widget](
	roots func() []ID,
	children func(ID) []ID,
	criar func() W,
	vincular func(row W, id ID),
) *Tree[ID, W] {
	return &Tree[ID, W]{
		roots:      roots,
		childrenOf: children,
		criar:      criar,
		vincular:   vincular,
		expanded:   map[ID]bool{},
	}
}

// BindSelected vincula o nó selecionado ao State em duas vias: clicar numa
// linha faz Set, um Set externo move o realce (o valor zero de ID limpa a
// seleção). Linhas que consomem o próprio clique não selecionam.
// Encadeável.
func (t *Tree[ID, W]) BindSelected(s *state.State[ID]) *Tree[ID, W] {
	t.selected = s
	s.Watch(func(ID) { t.Invalidate() })
	return t
}

// OnActivate define o callback de ativação de uma FOLHA — Enter na seleção
// ou duplo clique na linha. Encadeável.
func (t *Tree[ID, W]) OnActivate(fn func(ID)) *Tree[ID, W] {
	t.onActivate = fn
	return t
}

// Expand expande o nó dado (os ancestrais não são expandidos — um nó só
// aparece quando o caminho até ele está aberto).
func (t *Tree[ID, W]) Expand(id ID) {
	if !t.expanded[id] {
		t.expanded[id] = true
		t.modelChanged()
	}
}

// Collapse recolhe o nó dado.
func (t *Tree[ID, W]) Collapse(id ID) {
	if t.expanded[id] {
		delete(t.expanded, id)
		t.modelChanged()
	}
}

// Toggle alterna a expansão do nó dado.
func (t *Tree[ID, W]) Toggle(id ID) {
	if t.expanded[id] {
		t.Collapse(id)
	} else {
		t.Expand(id)
	}
}

// Expanded informa se o nó está expandido.
func (t *Tree[ID, W]) Expanded(id ID) bool {
	return t.expanded[id]
}

// Refresh relê o modelo (roots/children) e revincula as linhas visíveis —
// chame após mudar os dados.
func (t *Tree[ID, W]) Refresh() {
	t.modelChanged()
}

// modelChanged invalida o achatamento e o pool e agenda um redesenho.
func (t *Tree[ID, W]) modelChanged() {
	t.rowsValid = false
	t.poolValid = false
	t.Invalidate()
}

// ensureRows reconstrói o achatamento dos nós visíveis, se preciso.
func (t *Tree[ID, W]) ensureRows() {
	if t.rowsValid {
		return
	}
	t.rows = t.rows[:0]
	t.maxDepth = 0
	if t.roots != nil {
		for _, id := range t.roots() {
			t.flatten(id, 0)
		}
	}
	t.rowsValid = true
}

// flatten acrescenta o nó e, se expandido, os descendentes.
func (t *Tree[ID, W]) flatten(id ID, depth int) {
	kids := t.childrenOf(id)
	open := t.expanded[id]
	t.rows = append(t.rows, treeRow[ID]{id: id, depth: depth, leaf: len(kids) == 0, open: open})
	if depth > t.maxDepth {
		t.maxDepth = depth
	}
	if open {
		for _, k := range kids {
			t.flatten(k, depth+1)
		}
	}
}

// indentPx devolve o recuo por nível em pixels.
func (t *Tree[ID, W]) indentPx() int {
	return max(t.theme.Px(t.theme.TreeIndent), 1)
}

// ensureRowSize mede uma linha de amostra para a altura uniforme.
func (t *Tree[ID, W]) ensureRowSize() bool {
	if t.theme == nil {
		return false
	}
	if t.rowSized {
		return true
	}
	t.ensureRows()
	sample := t.criar()
	Mount(sample, t.theme)
	if len(t.rows) > 0 {
		t.vincular(sample, t.rows[0].id)
	}
	p := sample.PreferredSize()
	if p.Y <= 0 {
		return false
	}
	t.rowW, t.rowH = p.X, p.Y
	t.rowSized = true
	t.pool = append(t.pool, sample)
	return true
}

// PreferredSize devolve o total das linhas visíveis: a largura acomoda o
// recuo mais fundo e o chevron; a altura é linhas × altura de linha.
func (t *Tree[ID, W]) PreferredSize() image.Point {
	if !t.ensureRowSize() {
		return image.Point{}
	}
	t.ensureRows()
	indent := t.indentPx()
	return image.Point{
		X: (t.maxDepth+1)*indent + t.rowW,
		Y: len(t.rows) * t.rowH,
	}
}

// rowRect devolve o retângulo absoluto da linha visível i.
func (t *Tree[ID, W]) rowRect(i int) image.Rectangle {
	b := t.Bounds()
	top := b.Min.Y + i*t.rowH
	return image.Rect(b.Min.X, top, b.Max.X, top+t.rowH)
}

// rowIndexAt devolve o índice da linha visível sob o ponto, ou -1.
func (t *Tree[ID, W]) rowIndexAt(p image.Point) int {
	if !t.rowSized || t.rowH <= 0 {
		return -1
	}
	i := (p.Y - t.Bounds().Min.Y) / t.rowH
	t.ensureRows()
	if i < 0 || i >= len(t.rows) {
		return -1
	}
	return i
}

// contentX devolve onde começa o conteúdo (widget) da linha: depois do
// recuo e da coluna do chevron.
func (t *Tree[ID, W]) contentX(depth int) int {
	return t.Bounds().Min.X + (depth+1)*t.indentPx()
}

// Layout guarda os bounds e faz deles a viewport de fallback: uma Tree fora
// de um Scroll vincula o pool para os próprios bounds — sem isso, um render
// PARCIAL (dirty region menor que a árvore) encolheria o pool para a região
// suja. Dentro de um Scroll, SetViewport substitui pela região visível real.
func (t *Tree[ID, W]) Layout(bounds image.Rectangle) {
	t.BaseWidget.Layout(bounds)
	t.viewport = bounds
}

// SetViewport informa a região visível (o Scroll a chama no layout); o pool
// é vinculado imediatamente para cobri-la.
func (t *Tree[ID, W]) SetViewport(r image.Rectangle) {
	t.viewport = r
	t.ensure(r)
}

// Children devolve as linhas do pool atualmente vinculadas.
func (t *Tree[ID, W]) Children() []Widget {
	return t.kids
}

// Focusable devolve true: a seleção anda pelo teclado.
func (t *Tree[ID, W]) Focusable() bool {
	return true
}

// ensure vincula e posiciona o pool para cobrir a região visível dada.
func (t *Tree[ID, W]) ensure(visible image.Rectangle) {
	if !t.ensureRowSize() {
		t.kids = t.kids[:0]
		t.poolLen = 0
		return
	}
	t.ensureRows()
	if len(t.rows) == 0 {
		t.kids = t.kids[:0]
		t.poolLen = 0
		return
	}
	b := t.Bounds()
	visible = visible.Intersect(b)
	if visible.Empty() {
		return
	}
	first := (visible.Min.Y - b.Min.Y) / t.rowH
	last := (visible.Max.Y - 1 - b.Min.Y) / t.rowH
	if first < 0 {
		first = 0
	}
	if last >= len(t.rows) {
		last = len(t.rows) - 1
	}
	n := last - first + 1
	if n <= 0 {
		return
	}
	if t.poolValid && first == t.poolStart && n == t.poolLen && b.Min.Y == t.poolTop {
		return
	}
	for len(t.pool) < n {
		row := t.criar()
		Mount(row, t.theme)
		t.pool = append(t.pool, row)
	}
	t.kids = t.kids[:0]
	for k := 0; k < n; k++ {
		row := t.pool[k]
		r := t.rows[first+k]
		t.vincular(row, r.id)
		rect := t.rowRect(first + k)
		rect.Min.X = t.contentX(r.depth)
		row.Layout(rect)
		t.kids = append(t.kids, row)
	}
	t.poolStart, t.poolLen, t.poolTop, t.poolValid = first, n, b.Min.Y, true
}

// selectedIndex devolve o índice da linha visível selecionada, ou -1.
func (t *Tree[ID, W]) selectedIndex() int {
	if t.selected == nil {
		return -1
	}
	var zero ID
	id := t.selected.Get()
	if id == zero {
		return -1
	}
	t.ensureRows()
	for i, r := range t.rows {
		if r.id == id {
			return i
		}
	}
	return -1
}

// selectRow move a seleção para a linha visível i (com State vinculado).
func (t *Tree[ID, W]) selectRow(i int) {
	if t.selected == nil {
		return
	}
	t.ensureRows()
	if i < 0 || i >= len(t.rows) {
		return
	}
	if t.selected.Get() != t.rows[i].id {
		t.selected.Set(t.rows[i].id)
	}
	t.Invalidate()
}

// HandleEvent seleciona no clique, alterna no chevron (ou duplo clique) e
// navega pelo teclado quando focada.
func (t *Tree[ID, W]) HandleEvent(ev event.Event) bool {
	switch e := ev.(type) {
	case event.FocusEvent:
		t.focused = e.Gained
		t.Invalidate()
		return true
	case event.KeyEvent:
		return t.handleKey(e.Key)
	case event.MouseEvent:
		if e.Kind != event.MouseDown || e.Button != event.MouseButtonLeft {
			return false
		}
		i := t.rowIndexAt(e.Pos)
		if i < 0 {
			return false
		}
		r := t.rows[i]
		// Duplo clique na linha: alterna nós internos, ativa folhas.
		if t.theme != nil && t.dbl.hit(e.Pos, t.theme.DoubleClick, t.theme.Px(4)) {
			if r.leaf {
				if t.onActivate != nil {
					t.onActivate(r.id)
				}
			} else {
				t.Toggle(r.id)
			}
			return true
		}
		// Clique na coluna do chevron alterna o nó.
		if !r.leaf && e.Pos.X < t.contentX(r.depth) {
			t.Toggle(r.id)
			return true
		}
		t.selectRow(i)
		return true
	}
	return false
}

// handleKey navega a seleção pelas linhas visíveis: Cima/Baixo movem,
// Direita expande (ou desce ao primeiro filho), Esquerda recolhe (ou sobe
// ao pai), Home/End extremos, Enter ativa a folha.
func (t *Tree[ID, W]) handleKey(k event.Key) bool {
	if t.selected == nil {
		return false
	}
	t.ensureRows()
	if len(t.rows) == 0 {
		return false
	}
	i := t.selectedIndex()
	switch k {
	case event.KeyDown:
		t.selectRow(min(i+1, len(t.rows)-1))
	case event.KeyUp:
		if i < 0 {
			t.selectRow(0)
		} else {
			t.selectRow(max(i-1, 0))
		}
	case event.KeyHome:
		t.selectRow(0)
	case event.KeyEnd:
		t.selectRow(len(t.rows) - 1)
	case event.KeyRight:
		if i < 0 {
			return false
		}
		r := t.rows[i]
		switch {
		case r.leaf:
			return true
		case !r.open:
			t.Expand(r.id)
		default:
			t.selectRow(i + 1) // já aberto: desce ao primeiro filho
		}
	case event.KeyLeft:
		if i < 0 {
			return false
		}
		r := t.rows[i]
		if r.open {
			t.Collapse(r.id)
			return true
		}
		// Sobe ao pai: a linha anterior com profundidade menor.
		for j := i - 1; j >= 0; j-- {
			if t.rows[j].depth < r.depth {
				t.selectRow(j)
				break
			}
		}
	case event.KeyEnter:
		if i < 0 {
			return false
		}
		if r := t.rows[i]; r.leaf {
			if t.onActivate != nil {
				t.onActivate(r.id)
			}
		} else {
			t.Toggle(r.id)
		}
	default:
		return false
	}
	return true
}

// Draw garante o pool sobre a viewport, pinta o realce da seleção (com anel
// de foco quando a árvore está focada), os chevrons e as linhas.
func (t *Tree[ID, W]) Draw(dst *image.RGBA) {
	if t.theme == nil {
		return
	}
	region := t.viewport
	if region.Empty() {
		region = dst.Bounds()
	}
	t.ensure(region)
	th := t.theme

	if sel := t.selectedIndex(); sel >= 0 {
		if r := t.rowRect(sel); r.Overlaps(dst.Bounds()) {
			render.FillRect(dst, r, th.Selection)
			if t.focused {
				render.StrokeRoundRect(dst, r, th.RadiusPx(), th.BorderPx(), th.FocusOutline)
			}
		}
	}

	// Chevrons das linhas cobertas pelo pool (as visíveis).
	for k := 0; k < t.poolLen; k++ {
		r := t.rows[t.poolStart+k]
		if r.leaf {
			continue
		}
		rect := t.rowRect(t.poolStart + k)
		if rect.Overlaps(dst.Bounds()) {
			t.drawChevron(dst, rect, r)
		}
	}

	for _, row := range t.kids {
		if row.Bounds().Overlaps(dst.Bounds()) {
			row.Draw(dst)
		}
	}
	t.drawDisabledOverlay(dst)
}

// drawChevron desenha o triângulo de expansão na coluna de recuo da linha:
// apontando para a direita (fechado) ou para baixo (aberto), com faixas de
// FillRect como o Dropdown.
func (t *Tree[ID, W]) drawChevron(dst *image.RGBA, row image.Rectangle, r treeRow[ID]) {
	th := t.theme
	size := th.Px(8)
	indent := t.indentPx()
	x := row.Min.X + r.depth*indent + (indent-size)/2
	y := row.Min.Y + (row.Dy()-size/2)/2
	if r.open {
		// ▼: faixas horizontais de largura decrescente.
		for i := 0; i < size/2; i++ {
			render.FillRect(dst, image.Rect(x+i, y+i, x+size-i, y+i+1), th.Placeholder)
		}
		return
	}
	// ▶: faixas verticais de altura decrescente (girado 90°).
	cy := row.Min.Y + (row.Dy()-size)/2
	for i := 0; i < size/2; i++ {
		render.FillRect(dst, image.Rect(x+i, cy+i, x+i+1, cy+size-i), th.Placeholder)
	}
}
