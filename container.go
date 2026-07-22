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
