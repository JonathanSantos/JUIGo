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

// FillCircle preenche o círculo de centro c e raio r (em pixels),
// sobrescrevendo como FillRect, com antialiasing na borda: os pixels do
// traço do arco misturam a cor com o conteúdo já desenhado em dst (rampa
// de 1px de cobertura) — desenhe o fundo antes. O miolo de cada linha é
// preenchido em faixa cheia. Não aloca.
func FillCircle(dst *image.RGBA, c image.Point, r int, col color.RGBA) {
	if r <= 0 {
		return
	}
	clip := dst.Bounds()
	for y := c.Y - r - 1; y < c.Y+r+1; y++ {
		if y < clip.Min.Y || y >= clip.Max.Y {
			continue
		}
		py := float64(y) + 0.5 - float64(c.Y)
		outer := (float64(r)+0.5)*(float64(r)+0.5) - py*py
		if outer <= 0 {
			continue
		}
		xo := math.Sqrt(outer) // além disso, cobertura zero
		// Faixa sólida: pixels cujo centro dista até r-0.5 do centro.
		xi := -1.0
		if inner := (float64(r)-0.5)*(float64(r)-0.5) - py*py; inner > 0 {
			xi = math.Sqrt(inner)
		}
		solid0, solid1 := c.X, c.X
		if xi >= 0 {
			solid0 = int(math.Ceil(float64(c.X) - xi - 0.5))
			solid1 = int(math.Floor(float64(c.X)+xi-0.5)) + 1
			FillRect(dst, image.Rect(solid0, y, solid1, y+1), col)
		}
		// Beiradas com antialiasing: poucos pixels por lado.
		edge0 := int(math.Floor(float64(c.X) - xo + 0.5))
		edge1 := int(math.Ceil(float64(c.X)+xo-0.5)) + 1
		for x := edge0; x < solid0; x++ {
			blendRingPixel(dst, clip, x, y, c, r, 0, col)
		}
		for x := solid1; x < edge1; x++ {
			blendRingPixel(dst, clip, x, y, c, r, 0, col)
		}
	}
}

// StrokeCircle desenha um anel (contorno de círculo) com a espessura dada e
// antialiasing nos dois arcos — a versão vazada do FillCircle. Não aloca.
func StrokeCircle(dst *image.RGBA, c image.Point, r, w int, col color.RGBA) {
	if r <= 0 || w <= 0 {
		return
	}
	if w >= r {
		FillCircle(dst, c, r, col)
		return
	}
	inner := r - w
	clip := dst.Bounds()
	for y := c.Y - r - 1; y < c.Y+r+1; y++ {
		if y < clip.Min.Y || y >= clip.Max.Y {
			continue
		}
		py := float64(y) + 0.5 - float64(c.Y)
		outer := (float64(r)+0.5)*(float64(r)+0.5) - py*py
		if outer <= 0 {
			continue
		}
		xo := math.Sqrt(outer)
		edge0 := int(math.Floor(float64(c.X) - xo + 0.5))
		edge1 := int(math.Ceil(float64(c.X)+xo-0.5)) + 1
		// Miolo poupado: pixels cujo centro dista menos que inner-0.5 do
		// centro têm cobertura zero no anel.
		hole := -1.0
		if h := (float64(inner)-0.5)*(float64(inner)-0.5) - py*py; h > 0 {
			hole = math.Sqrt(h)
		}
		if hole < 0 {
			// Linha sem miolo: o anel atravessa inteiro.
			for x := edge0; x < edge1; x++ {
				blendRingPixel(dst, clip, x, y, c, r, inner, col)
			}
			continue
		}
		hole0 := int(math.Ceil(float64(c.X) - hole - 0.5))
		hole1 := int(math.Floor(float64(c.X)+hole-0.5)) + 1
		for x := edge0; x < hole0; x++ {
			blendRingPixel(dst, clip, x, y, c, r, inner, col)
		}
		for x := hole1; x < edge1; x++ {
			blendRingPixel(dst, clip, x, y, c, r, inner, col)
		}
	}
}

// blendRingPixel mistura col no pixel (x,y) com a cobertura do anel entre
// os raios inner e r do círculo de centro c (inner <= 0 preenche até o arco
// externo — o caso do FillCircle). É a rampa de 1px do antialiasing.
func blendRingPixel(dst *image.RGBA, clip image.Rectangle, x, y int, c image.Point, r, inner int, col color.RGBA) {
	if x < clip.Min.X || x >= clip.Max.X {
		return
	}
	d := math.Hypot(float64(x)+0.5-float64(c.X), float64(y)+0.5-float64(c.Y))
	cov := float64(r) - d + 0.5
	if cov <= 0 {
		return
	}
	if cov > 1 {
		cov = 1
	}
	if inner > 0 {
		if ci := float64(inner) - d + 0.5; ci > 0 {
			if ci > 1 {
				ci = 1
			}
			cov -= ci
		}
		if cov <= 0 {
			return
		}
	}
	f := uint32(cov*255 + 0.5)
	i := dst.PixOffset(x, y)
	p := dst.Pix[i : i+4 : i+4]
	if f == 255 {
		p[0], p[1], p[2], p[3] = col.R, col.G, col.B, col.A
		return
	}
	inv := 255 - f
	p[0] = uint8((uint32(col.R)*f + uint32(p[0])*inv + 127) / 255)
	p[1] = uint8((uint32(col.G)*f + uint32(p[1])*inv + 127) / 255)
	p[2] = uint8((uint32(col.B)*f + uint32(p[2])*inv + 127) / 255)
	p[3] = uint8((uint32(col.A)*f + uint32(p[3])*inv + 127) / 255)
}

// CrossFade preenche a região r de dst com a mistura linear entre a e b:
// t=0 copia a, t=1 copia b, valores intermediários fundem os dois canal a
// canal. a e b devem estar ALINHADOS a dst (mesmo sistema de coordenadas —
// como os snapshots do Navigator, criados com os bounds do widget); a região
// é recortada à interseção dos três. É a base da transição de fade entre
// telas. Não aloca.
func CrossFade(dst *image.RGBA, a, b *image.RGBA, r image.Rectangle, t float64) {
	r = r.Intersect(dst.Bounds()).Intersect(a.Bounds()).Intersect(b.Bounds())
	if r.Empty() {
		return
	}
	f := int32(t*256 + 0.5)
	if f <= 0 {
		draw.Draw(dst, r, a, r.Min, draw.Src)
		return
	}
	if f >= 256 {
		draw.Draw(dst, r, b, r.Min, draw.Src)
		return
	}
	rowLen := r.Dx() * 4
	for y := r.Min.Y; y < r.Max.Y; y++ {
		di := dst.PixOffset(r.Min.X, y)
		ai := a.PixOffset(r.Min.X, y)
		bi := b.PixOffset(r.Min.X, y)
		drow := dst.Pix[di : di+rowLen : di+rowLen]
		arow := a.Pix[ai : ai+rowLen : ai+rowLen]
		brow := b.Pix[bi : bi+rowLen : bi+rowLen]
		for x := 0; x < rowLen; x++ {
			va := int32(arow[x])
			drow[x] = uint8(va + (int32(brow[x])-va)*f/256)
		}
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
