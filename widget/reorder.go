package widget

import (
	"image"

	"github.com/JonathanSantos/JUIGo/theme"
)

// dragThreshold é o limiar de movimento (em pixels) que separa o clique do
// arrasto de reordenação.
func dragThreshold(th *theme.Theme) int {
	if th != nil {
		return th.Px(4)
	}
	return 4
}

// reorderPayload identifica um arrasto de reordenação: só o próprio widget
// de origem o aceita (arrastar ENTRE widgets pede StartDrag/DropTarget
// próprios da aplicação).
type reorderPayload struct {
	owner Widget
	from  int
}

// reorderArm é o estado de "braço armado" compartilhado por List e Table: o
// MouseDown numa linha arma; o MouseMove além do limiar dispara o arrasto.
type reorderArm struct {
	armed bool
	row   int
	pos   image.Point
}

// arm registra a linha e a posição do MouseDown.
func (a *reorderArm) arm(row int, pos image.Point) {
	a.armed, a.row, a.pos = true, row, pos
}

// disarm desarma sem disparar (MouseUp, MouseLeave).
func (a *reorderArm) disarm() {
	a.armed = false
}

// fire informa se o movimento até pos passou do limiar de arrasto e, nesse
// caso, desarma e devolve a linha armada.
func (a *reorderArm) fire(pos image.Point, threshold int) (int, bool) {
	if !a.armed {
		return 0, false
	}
	d := pos.Sub(a.pos)
	if d.X*d.X+d.Y*d.Y < threshold*threshold {
		return 0, false
	}
	a.armed = false
	return a.row, true
}

// insertionIndex converte a posição vertical do cursor na fronteira de
// inserção mais próxima: 0 antes da primeira linha até count depois da
// última.
func insertionIndex(posY, topY, rowH, count int) int {
	if rowH <= 0 {
		return 0
	}
	b := (posY - topY + rowH/2) / rowH
	if b < 0 {
		b = 0
	}
	if b > count {
		b = count
	}
	return b
}

// finalIndex converte a fronteira de inserção no índice FINAL do item após
// o movimento (remover de from e inserir na fronteira b).
func finalIndex(from, b int) int {
	if b > from {
		return b - 1
	}
	return b
}

// indicatorRect devolve a linha fina do indicador de inserção na fronteira
// b, contida nos bounds do widget.
func indicatorRect(bounds image.Rectangle, topY, rowH, b, thickness int) image.Rectangle {
	y := topY + b*rowH
	r := image.Rect(bounds.Min.X, y-thickness/2, bounds.Max.X, y-thickness/2+thickness)
	if r.Min.Y < bounds.Min.Y {
		r = r.Add(image.Pt(0, bounds.Min.Y-r.Min.Y))
	}
	if r.Max.Y > bounds.Max.Y {
		r = r.Add(image.Pt(0, bounds.Max.Y-r.Max.Y))
	}
	return r
}

// OnReorder habilita arrastar linhas para reordenar: segurar uma linha e
// mover além de um pequeno limiar inicia o arrasto (label(i) é o rótulo do
// fantasma; nil deixa o fantasma sem texto), a fronteira de inserção mais
// próxima do cursor ganha um indicador, e soltar chama fn(de, para) com o
// índice original e o índice FINAL do item — soltar no mesmo lugar não
// chama. A aplicação move o item nos dados e chama Refresh. Encadeável.
func (l *List[W]) OnReorder(label func(index int) string, fn func(de, para int)) *List[W] {
	l.reorderLabel = label
	l.onReorder = fn
	return l
}

// CanDrop aceita apenas reordenações vindas desta própria lista.
func (l *List[W]) CanDrop(payload any) bool {
	p, ok := payload.(reorderPayload)
	return ok && p.owner == Widget(l) && l.onReorder != nil
}

// Drop conclui a reordenação: converte a posição na fronteira de inserção e
// chama o fn do OnReorder quando o item realmente muda de lugar.
func (l *List[W]) Drop(payload any, pos image.Point) {
	p, ok := payload.(reorderPayload)
	if !ok || l.onReorder == nil || l.rowH <= 0 {
		return
	}
	b := insertionIndex(pos.Y, l.Bounds().Min.Y, l.rowH, l.count)
	if para := finalIndex(p.from, b); para != p.from {
		l.onReorder(p.from, para)
	}
}

// DropIndicatorRect desenha o indicador na fronteira de inserção sob o
// cursor (ver DropIndicator).
func (l *List[W]) DropIndicatorRect(payload any, pos image.Point) image.Rectangle {
	if l.theme == nil || l.rowH <= 0 {
		return image.Rectangle{}
	}
	b := insertionIndex(pos.Y, l.Bounds().Min.Y, l.rowH, l.count)
	return indicatorRect(l.Bounds(), l.Bounds().Min.Y, l.rowH, b, 2*l.theme.BorderPx())
}

// OnReorder habilita arrastar linhas para reordenar, como na List; o rótulo
// do fantasma é a primeira célula da linha. Soltar chama fn(de, para) com o
// índice original e o índice FINAL da linha — soltar no mesmo lugar não
// chama. A aplicação move a linha nos dados e chama Refresh. Encadeável.
func (t *Table) OnReorder(fn func(de, para int)) *Table {
	t.onReorder = fn
	return t
}

// CanDrop aceita apenas reordenações vindas desta própria tabela.
func (t *Table) CanDrop(payload any) bool {
	p, ok := payload.(reorderPayload)
	return ok && p.owner == Widget(t) && t.onReorder != nil
}

// Drop conclui a reordenação (ver List.Drop); a primeira linha fica logo
// abaixo do cabeçalho.
func (t *Table) Drop(payload any, pos image.Point) {
	p, ok := payload.(reorderPayload)
	if !ok || t.onReorder == nil || t.theme == nil {
		return
	}
	h := t.rowH()
	b := insertionIndex(pos.Y, t.Bounds().Min.Y+h, h, t.count)
	if para := finalIndex(p.from, b); para != p.from {
		t.onReorder(p.from, para)
	}
}

// DropIndicatorRect desenha o indicador na fronteira de inserção sob o
// cursor (ver DropIndicator).
func (t *Table) DropIndicatorRect(payload any, pos image.Point) image.Rectangle {
	if t.theme == nil {
		return image.Rectangle{}
	}
	h := t.rowH()
	top := t.Bounds().Min.Y + h
	b := insertionIndex(pos.Y, top, h, t.count)
	return indicatorRect(t.Bounds(), top, h, b, 2*t.theme.BorderPx())
}
