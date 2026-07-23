package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// textColor e textSrc são reutilizados por DrawText para evitar alocações no
// caminho quente de desenho. O Uniform guarda um ponteiro estável para a cor
// (atribuir color.RGBA direto à interface color.Color faria boxing a cada
// chamada). Como o JUIGo é single-threaded (main thread), não há corrida.
var (
	textColor color.RGBA
	textSrc   = &image.Uniform{C: &textColor}
)

// FillRect preenche o retângulo r de dst com a cor sólida c, sem blending.
// A área é recortada para os limites de dst. Não aloca.
func FillRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	rowLen := r.Dx() * 4
	i0 := dst.PixOffset(r.Min.X, r.Min.Y)
	firstRow := dst.Pix[i0 : i0+rowLen]
	for x := 0; x < rowLen; x += 4 {
		firstRow[x+0] = c.R
		firstRow[x+1] = c.G
		firstRow[x+2] = c.B
		firstRow[x+3] = c.A
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		i := dst.PixOffset(r.Min.X, y)
		copy(dst.Pix[i:i+rowLen], firstRow)
	}
}

// StrokeRect desenha o contorno do retângulo r com espessura w (em pixels),
// por dentro dos limites de r. Não aloca.
func StrokeRect(dst *image.RGBA, r image.Rectangle, w int, c color.RGBA) {
	if w <= 0 || r.Empty() {
		return
	}
	// Topo, base, e laterais entre as duas faixas horizontais.
	FillRect(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+w), c)
	FillRect(dst, image.Rect(r.Min.X, r.Max.Y-w, r.Max.X, r.Max.Y), c)
	FillRect(dst, image.Rect(r.Min.X, r.Min.Y+w, r.Min.X+w, r.Max.Y-w), c)
	FillRect(dst, image.Rect(r.Max.X-w, r.Min.Y+w, r.Max.X, r.Max.Y-w), c)
}

// DrawText desenha s em dst com a face dada, com a origem da baseline em dot
// e cor c. O texto não é recortado além dos limites de dst.
func DrawText(dst *image.RGBA, face font.Face, s string, dot image.Point, c color.RGBA) {
	textColor = c
	d := font.Drawer{
		Dst:  dst,
		Src:  textSrc,
		Face: face,
		Dot:  fixed.P(dot.X, dot.Y),
	}
	d.DrawString(s)
}

// MeasureText devolve a largura de s em pixels para a face dada. É a base de
// Theme.MeasureString, a única fonte de verdade para largura de texto.
func MeasureText(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

// fillColor e fillSrc são reutilizados por FillRectOver (mesmo padrão de
// textSrc: ponteiro estável para evitar boxing por chamada). Single-threaded.
var (
	fillColor color.RGBA
	fillSrc   = &image.Uniform{C: &fillColor}
)

// FillRectOver preenche r com a cor c APLICANDO blending alfa (src-over) —
// ao contrário de FillRect, que sobrescreve. A cor é interpretada com alfa
// NÃO pré-multiplicado (straight): informe R/G/B plenos e A com a opacidade;
// a pré-multiplicação exigida pelo draw.Over da stdlib acontece aqui dentro.
// Usa o caminho rápido da stdlib; não aloca.
func FillRectOver(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	a := uint32(c.A)
	fillColor = color.RGBA{
		R: uint8(uint32(c.R) * a / 255),
		G: uint8(uint32(c.G) * a / 255),
		B: uint8(uint32(c.B) * a / 255),
		A: c.A,
	}
	draw.Draw(dst, r, fillSrc, image.Point{}, draw.Over)
}

// FillCircle preenche o círculo de centro c e raio r (em pixels), sem
// blending, por varredura de linhas. Não aloca.
func FillCircle(dst *image.RGBA, c image.Point, r int, col color.RGBA) {
	if r <= 0 {
		return
	}
	for dy := -r; dy <= r; dy++ {
		dx := int(math.Sqrt(float64(r*r - dy*dy)))
		FillRect(dst, image.Rect(c.X-dx, c.Y+dy, c.X+dx, c.Y+dy+1), col)
	}
}

// StrokeCircle desenha um anel (contorno de círculo) com a espessura dada,
// por varredura de linhas — a versão vazada do FillCircle.
func StrokeCircle(dst *image.RGBA, c image.Point, r, w int, col color.RGBA) {
	if r <= 0 || w <= 0 {
		return
	}
	if w >= r {
		FillCircle(dst, c, r, col)
		return
	}
	inner := r - w
	for dy := -r; dy <= r; dy++ {
		dx := int(math.Sqrt(float64(r*r - dy*dy)))
		if dy < -inner || dy > inner {
			// Fora do miolo: a linha inteira pertence ao anel.
			FillRect(dst, image.Rect(c.X-dx, c.Y+dy, c.X+dx, c.Y+dy+1), col)
			continue
		}
		di := int(math.Sqrt(float64(inner*inner - dy*dy)))
		FillRect(dst, image.Rect(c.X-dx, c.Y+dy, c.X-di, c.Y+dy+1), col)
		FillRect(dst, image.Rect(c.X+di, c.Y+dy, c.X+dx, c.Y+dy+1), col)
	}
}

// Clip preenche out com uma VISÃO de dst recortada a r — mesmos pixels, sem
// cópia e sem alocação (out é um scratch reutilizável, normalmente um campo
// do widget) — e a devolve. Desenhar na visão não alcança nada fora de r,
// pois todas as primitivas recortam contra os bounds do destino. Semântica
// idêntica à de image.RGBA.SubImage.
func Clip(dst *image.RGBA, r image.Rectangle, out *image.RGBA) *image.RGBA {
	r = r.Intersect(dst.Bounds())
	if r.Empty() {
		*out = image.RGBA{Rect: r}
		return out
	}
	out.Pix = dst.Pix[dst.PixOffset(r.Min.X, r.Min.Y):]
	out.Stride = dst.Stride
	out.Rect = r
	return out
}
