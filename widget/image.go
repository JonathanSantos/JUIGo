package widget

import (
	"image"
	"image/draw"

	xdraw "golang.org/x/image/draw"

	"github.com/JonathanSantos/JUIGo/render"
)

// Image exibe uma image.Image, redimensionada para caber nos bounds
// PRESERVANDO a proporção, centralizada. O tamanho preferido é o da imagem
// em unidades lógicas (escalado pelo tema em HiDPI). Não é focável.
//
// O redimensionamento (bilinear) acontece apenas quando a imagem ou o
// tamanho de destino mudam — o resultado fica em cache e o desenho por
// frame é uma cópia sem alocação.
type Image struct {
	BaseWidget

	src        image.Image
	scaled     *image.RGBA
	srcChanged bool
	clip       image.RGBA
}

// NewImage cria o widget exibindo src (pode ser nil e definido depois).
// O tema é herdado no mount.
func NewImage(src image.Image) *Image {
	return &Image{src: src, srcChanged: true}
}

// SetImage troca a imagem exibida e agenda um redesenho.
func (im *Image) SetImage(src image.Image) {
	im.src = src
	im.srcChanged = true
	im.Invalidate()
}

// PreferredSize devolve o tamanho da imagem em unidades lógicas convertidas
// pela escala do tema. Antes do mount ou sem imagem, devolve zero.
func (im *Image) PreferredSize() image.Point {
	if im.theme == nil || im.src == nil {
		return image.Point{}
	}
	b := im.src.Bounds()
	return image.Point{X: im.theme.Px(b.Dx()), Y: im.theme.Px(b.Dy())}
}

// target devolve o retângulo de destino: a maior área dentro dos bounds que
// preserva a proporção da imagem, centralizada.
func (im *Image) target() image.Rectangle {
	b := im.Bounds()
	sb := im.src.Bounds()
	if sb.Dx() == 0 || sb.Dy() == 0 || b.Dx() == 0 || b.Dy() == 0 {
		return image.Rectangle{}
	}
	w, h := b.Dx(), b.Dx()*sb.Dy()/sb.Dx()
	if h > b.Dy() {
		h = b.Dy()
		w = b.Dy() * sb.Dx() / sb.Dy()
	}
	x := b.Min.X + (b.Dx()-w)/2
	y := b.Min.Y + (b.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

// Draw desenha a imagem (rescalonando apenas se necessário).
func (im *Image) Draw(dst *image.RGBA) {
	if im.src == nil {
		return
	}
	target := im.target()
	if target.Empty() {
		return
	}
	if im.scaled == nil || im.scaled.Bounds().Size() != target.Size() || im.srcChanged {
		im.scaled = image.NewRGBA(image.Rectangle{Max: target.Size()})
		xdraw.ApproxBiLinear.Scale(im.scaled, im.scaled.Bounds(), im.src, im.src.Bounds(), xdraw.Src, nil)
		im.srcChanged = false
	}
	view := render.Clip(dst, im.Bounds(), &im.clip)
	draw.Draw(view, target, im.scaled, image.Point{}, draw.Over)
}
