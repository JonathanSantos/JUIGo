package render

import (
	"image"
	"image/color"

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
