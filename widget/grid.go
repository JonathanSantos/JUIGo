package widget

import "image"

// Grid distribui os filhos em uma grade de N colunas, preenchida por linhas
// (row-major): o filho i cai na coluna i%N da linha i/N. Cada coluna tem a
// largura do seu filho mais largo e cada linha, a altura do mais alto — o
// layout clássico de formulários rótulo+campo:
//
//	juigo.NewGrid(2,
//	    juigo.NewText("Nome:"), campoNome,
//	    juigo.NewText("E-mail:"), campoEmail,
//	)
//
// A largura que sobrar é distribuída entre as colunas que contêm filhos
// marcados com Grow (proporcional ao maior peso da coluna). Dentro da
// célula, o filho estica; Centered/AtStart/AtEnd posicionam na largura
// preferida. Gap e Pad funcionam como nos boxes.
type Grid struct {
	Container

	cols int
	// Medidas reutilizadas entre layouts (sem alocação por frame).
	colW    []int
	rowH    []int
	colGrow []int
}

// NewGrid cria uma grade com o número de colunas e os filhos dados. O tema é
// herdado no mount.
func NewGrid(cols int, children ...Widget) *Grid {
	g := &Grid{cols: cols}
	g.Padding = -1
	g.Spacing = -1
	g.Add(children...)
	return g
}

// Gap define o espaço entre células, em unidades lógicas. Encadeável.
func (g *Grid) Gap(spacing int) *Grid {
	g.Spacing = spacing
	return g
}

// Pad define o espaço interno das bordas, em unidades lógicas. Encadeável.
func (g *Grid) Pad(padding int) *Grid {
	g.Padding = padding
	return g
}

// measure recalcula larguras de coluna, alturas de linha e pesos.
func (g *Grid) measure() (cols, rows int) {
	children := g.Children()
	cols = g.cols
	if cols < 1 {
		cols = 1
	}
	rows = (len(children) + cols - 1) / cols

	g.colW = resizeInts(g.colW, cols)
	g.colGrow = resizeInts(g.colGrow, cols)
	g.rowH = resizeInts(g.rowH, rows)
	for i, ch := range children {
		c, r := i%cols, i/cols
		p := ch.PreferredSize()
		if p.X > g.colW[c] {
			g.colW[c] = p.X
		}
		if p.Y > g.rowH[r] {
			g.rowH[r] = p.Y
		}
		if gw := growOf(ch); gw > g.colGrow[c] {
			g.colGrow[c] = gw
		}
	}
	return cols, rows
}

// Layout posiciona os filhos nas células, distribuindo a sobra horizontal
// entre as colunas com Grow.
func (g *Grid) Layout(bounds image.Rectangle) {
	g.BaseWidget.Layout(bounds)
	children := g.Children()
	if len(children) == 0 {
		return
	}
	cols, rows := g.measure()
	spacing, padding := g.metrics()

	avail := bounds.Dx() - 2*padding - spacing*(cols-1)
	total, growSum := 0, 0
	for c := 0; c < cols; c++ {
		total += g.colW[c]
		growSum += g.colGrow[c]
	}
	if leftover := avail - total; leftover > 0 && growSum > 0 {
		d := distributor{leftover: leftover, weightSum: growSum}
		for c := 0; c < cols; c++ {
			if g.colGrow[c] > 0 {
				g.colW[c] += d.next(g.colGrow[c])
			}
		}
	}

	y := bounds.Min.Y + padding
	for r := 0; r < rows; r++ {
		x := bounds.Min.X + padding
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if i >= len(children) {
				break
			}
			g.placeCell(children[i], image.Rect(x, y, x+g.colW[c], y+g.rowH[r]))
			x += g.colW[c] + spacing
		}
		y += g.rowH[r] + spacing
	}
}

// placeCell aplica o alinhamento horizontal do filho dentro da célula.
func (g *Grid) placeCell(ch Widget, cell image.Rectangle) {
	align := crossOf(ch)
	if align == crossStretch {
		ch.Layout(cell)
		return
	}
	w := ch.PreferredSize().X
	if max := cell.Dx(); w > max {
		w = max
	}
	x := cell.Min.X
	switch align {
	case crossCenter:
		x = cell.Min.X + (cell.Dx()-w)/2
	case crossEnd:
		x = cell.Max.X - w
	}
	ch.Layout(image.Rect(x, cell.Min.Y, x+w, cell.Max.Y))
}

// PreferredSize devolve a soma das colunas e linhas medidas, mais os espaços.
func (g *Grid) PreferredSize() image.Point {
	children := g.Children()
	if len(children) == 0 {
		return image.Point{}
	}
	cols, rows := g.measure()
	spacing, padding := g.metrics()
	w, h := 0, 0
	for c := 0; c < cols; c++ {
		w += g.colW[c]
	}
	for r := 0; r < rows; r++ {
		h += g.rowH[r]
	}
	return image.Point{
		X: w + 2*padding + spacing*(cols-1),
		Y: h + 2*padding + spacing*(rows-1),
	}
}

// resizeInts devolve s com exatamente n posições zeradas, reutilizando a
// capacidade existente.
func resizeInts(s []int, n int) []int {
	if cap(s) < n {
		return make([]int, n)
	}
	s = s[:n]
	for i := range s {
		s[i] = 0
	}
	return s
}
