package juigo

import "image"

// Container agrupa widgets filhos com posicionamento ABSOLUTO: cada filho
// mantém os bounds que recebeu no próprio Layout; o Container não os move.
// Para layout automático, use VBox ou HBox.
type Container struct {
	BaseWidget
	// Padding é o espaço interno, em pixels, usado pelos containers de
	// layout derivados (VBox/HBox). O Container absoluto não o aplica.
	Padding int
	// Spacing é o espaço entre filhos consecutivos, usado pelos containers
	// de layout derivados (VBox/HBox).
	Spacing int

	children []Widget
}

// NewContainer cria um container vazio de posicionamento absoluto.
func NewContainer() *Container {
	return &Container{}
}

// Add anexa widgets ao final da lista de filhos (desenhados por último,
// portanto por cima e com prioridade no hit-test).
func (c *Container) Add(ws ...Widget) {
	c.children = append(c.children, ws...)
}

// Children devolve os filhos na ordem de desenho.
func (c *Container) Children() []Widget {
	return c.children
}

// Draw desenha os filhos na ordem em que foram adicionados.
func (c *Container) Draw(dst *image.RGBA) {
	for _, ch := range c.children {
		ch.Draw(dst)
	}
}

// PreferredSize devolve a união dos bounds dos filhos, já que no container
// absoluto cada filho define a própria posição.
func (c *Container) PreferredSize() image.Point {
	var u image.Rectangle
	for _, ch := range c.children {
		u = u.Union(ch.Bounds())
	}
	return u.Size()
}

// VBox distribui os filhos verticalmente: cada um recebe a própria altura
// preferida e a largura disponível, com Padding nas bordas e Spacing entre
// filhos consecutivos.
type VBox struct {
	Container
}

// NewVBox cria um VBox com o espaçamento e o padding dados, em pixels.
func NewVBox(spacing, padding int) *VBox {
	v := &VBox{}
	v.Spacing = spacing
	v.Padding = padding
	return v
}

// Layout posiciona os filhos de cima para baixo dentro de bounds.
func (v *VBox) Layout(bounds image.Rectangle) {
	v.BaseWidget.Layout(bounds)
	x0 := bounds.Min.X + v.Padding
	x1 := bounds.Max.X - v.Padding
	y := bounds.Min.Y + v.Padding
	for _, ch := range v.Children() {
		h := ch.PreferredSize().Y
		ch.Layout(image.Rect(x0, y, x1, y+h))
		y += h + v.Spacing
	}
}

// PreferredSize devolve a maior largura preferida entre os filhos e a soma
// das alturas, acrescidas de Spacing e Padding.
func (v *VBox) PreferredSize() image.Point {
	var w, h int
	for i, ch := range v.Children() {
		p := ch.PreferredSize()
		if p.X > w {
			w = p.X
		}
		h += p.Y
		if i > 0 {
			h += v.Spacing
		}
	}
	return image.Point{X: w + 2*v.Padding, Y: h + 2*v.Padding}
}

// HBox distribui os filhos horizontalmente: cada um recebe a própria largura
// preferida e a altura disponível, com Padding nas bordas e Spacing entre
// filhos consecutivos.
type HBox struct {
	Container
}

// NewHBox cria um HBox com o espaçamento e o padding dados, em pixels.
func NewHBox(spacing, padding int) *HBox {
	h := &HBox{}
	h.Spacing = spacing
	h.Padding = padding
	return h
}

// Layout posiciona os filhos da esquerda para a direita dentro de bounds.
func (h *HBox) Layout(bounds image.Rectangle) {
	h.BaseWidget.Layout(bounds)
	y0 := bounds.Min.Y + h.Padding
	y1 := bounds.Max.Y - h.Padding
	x := bounds.Min.X + h.Padding
	for _, ch := range h.Children() {
		w := ch.PreferredSize().X
		ch.Layout(image.Rect(x, y0, x+w, y1))
		x += w + h.Spacing
	}
}

// PreferredSize devolve a soma das larguras preferidas dos filhos e a maior
// altura, acrescidas de Spacing e Padding.
func (h *HBox) PreferredSize() image.Point {
	var w, ht int
	for i, ch := range h.Children() {
		p := ch.PreferredSize()
		if p.Y > ht {
			ht = p.Y
		}
		w += p.X
		if i > 0 {
			w += h.Spacing
		}
	}
	return image.Point{X: w + 2*h.Padding, Y: ht + 2*h.Padding}
}
