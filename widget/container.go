package widget

import (
	"image"

	"juigo/theme"
)

// Container agrupa widgets filhos com posicionamento ABSOLUTO: cada filho
// mantém os bounds que recebeu no próprio Layout; o Container não os move.
// Para layout automático, use VBox ou HBox.
type Container struct {
	BaseWidget
	// Padding é o espaço interno das bordas em unidades LÓGICAS (convertido
	// pela escala do tema no layout). Negativo usa o padrão: sem padding.
	// Aplicado apenas pelos containers de layout (VBox/HBox).
	Padding int
	// Spacing é o espaço entre filhos consecutivos em unidades lógicas.
	// Negativo usa o padrão do tema (Theme.Spacing). Aplicado apenas pelos
	// containers de layout (VBox/HBox).
	Spacing int

	children []Widget
}

// NewContainer cria um container de posicionamento absoluto com os filhos
// dados. O tema é herdado no mount.
func NewContainer(children ...Widget) *Container {
	c := &Container{Padding: -1, Spacing: -1}
	c.Add(children...)
	return c
}

// Add anexa widgets ao final da lista de filhos (desenhados por último,
// portanto por cima e com prioridade no hit-test).
func (c *Container) Add(ws ...Widget) {
	c.children = append(c.children, ws...)
}

// SetTheme define um tema explícito para o container e o propaga
// imediatamente à subárvore atual (filhos adicionados depois o herdam no
// próximo mount). Também permite montar árvores sem App — testes e
// renderização offscreen — aplicando o tema pela raiz.
func (c *Container) SetTheme(t *theme.Theme) {
	c.BaseWidget.SetTheme(t)
	Mount(c, t)
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

// metrics resolve Spacing e Padding para pixels na escala do tema, aplicando
// os padrões (Spacing do tema; padding zero) quando os campos são negativos.
func (c *Container) metrics() (spacingPx, paddingPx int) {
	spacing, padding := c.Spacing, c.Padding
	if spacing < 0 {
		if c.theme != nil {
			spacing = c.theme.Spacing
		} else {
			spacing = 0
		}
	}
	if padding < 0 {
		padding = 0
	}
	if c.theme != nil {
		return c.theme.Px(spacing), c.theme.Px(padding)
	}
	return spacing, padding
}

// VBox distribui os filhos verticalmente: cada um recebe a própria altura
// preferida e a largura disponível, com Padding nas bordas e Spacing entre
// filhos consecutivos.
type VBox struct {
	Container
}

// NewVBox cria um VBox com os filhos dados, espaçamento padrão do tema e sem
// padding. Ajuste com Gap e Pad (unidades lógicas, escaladas pelo tema).
func NewVBox(children ...Widget) *VBox {
	v := &VBox{}
	v.Padding = -1
	v.Spacing = -1
	v.Add(children...)
	return v
}

// Gap define o espaço entre filhos, em unidades lógicas. Encadeável.
func (v *VBox) Gap(spacing int) *VBox {
	v.Spacing = spacing
	return v
}

// Pad define o espaço interno das bordas, em unidades lógicas. Encadeável.
func (v *VBox) Pad(padding int) *VBox {
	v.Padding = padding
	return v
}

// Layout posiciona os filhos de cima para baixo dentro de bounds.
func (v *VBox) Layout(bounds image.Rectangle) {
	v.BaseWidget.Layout(bounds)
	spacing, padding := v.metrics()
	x0 := bounds.Min.X + padding
	x1 := bounds.Max.X - padding
	y := bounds.Min.Y + padding
	for _, ch := range v.Children() {
		h := ch.PreferredSize().Y
		ch.Layout(image.Rect(x0, y, x1, y+h))
		y += h + spacing
	}
}

// PreferredSize devolve a maior largura preferida entre os filhos e a soma
// das alturas, acrescidas de Spacing e Padding.
func (v *VBox) PreferredSize() image.Point {
	spacing, padding := v.metrics()
	var w, h int
	for i, ch := range v.Children() {
		p := ch.PreferredSize()
		if p.X > w {
			w = p.X
		}
		h += p.Y
		if i > 0 {
			h += spacing
		}
	}
	return image.Point{X: w + 2*padding, Y: h + 2*padding}
}

// HBox distribui os filhos horizontalmente: cada um recebe a própria largura
// preferida e a altura disponível, com Padding nas bordas e Spacing entre
// filhos consecutivos.
type HBox struct {
	Container
}

// NewHBox cria um HBox com os filhos dados, espaçamento padrão do tema e sem
// padding. Ajuste com Gap e Pad (unidades lógicas, escaladas pelo tema).
func NewHBox(children ...Widget) *HBox {
	h := &HBox{}
	h.Padding = -1
	h.Spacing = -1
	h.Add(children...)
	return h
}

// Gap define o espaço entre filhos, em unidades lógicas. Encadeável.
func (h *HBox) Gap(spacing int) *HBox {
	h.Spacing = spacing
	return h
}

// Pad define o espaço interno das bordas, em unidades lógicas. Encadeável.
func (h *HBox) Pad(padding int) *HBox {
	h.Padding = padding
	return h
}

// Layout posiciona os filhos da esquerda para a direita dentro de bounds.
func (h *HBox) Layout(bounds image.Rectangle) {
	h.BaseWidget.Layout(bounds)
	spacing, padding := h.metrics()
	y0 := bounds.Min.Y + padding
	y1 := bounds.Max.Y - padding
	x := bounds.Min.X + padding
	for _, ch := range h.Children() {
		w := ch.PreferredSize().X
		ch.Layout(image.Rect(x, y0, x+w, y1))
		x += w + spacing
	}
}

// PreferredSize devolve a soma das larguras preferidas dos filhos e a maior
// altura, acrescidas de Spacing e Padding.
func (h *HBox) PreferredSize() image.Point {
	spacing, padding := h.metrics()
	var w, ht int
	for i, ch := range h.Children() {
		p := ch.PreferredSize()
		if p.Y > ht {
			ht = p.Y
		}
		w += p.X
		if i > 0 {
			w += spacing
		}
	}
	return image.Point{X: w + 2*padding, Y: ht + 2*padding}
}
