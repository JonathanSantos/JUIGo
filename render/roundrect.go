package render

import (
	"image"
	"image/color"
	"math"
)

// clampRadius limita o raio de canto ao que cabe no retângulo: no máximo a
// metade do menor lado. Raios negativos viram zero.
func clampRadius(r image.Rectangle, radius int) int {
	if radius < 0 {
		return 0
	}
	if half := r.Dx() / 2; radius > half {
		radius = half
	}
	if half := r.Dy() / 2; radius > half {
		radius = half
	}
	return radius
}

// FillRoundRect preenche r com a cor c e cantos arredondados de raio radius
// (em pixels), sobrescrevendo como FillRect; nos arcos dos cantos o traço é
// suavizado (antialiasing) misturando c com o conteúdo já desenhado em dst —
// desenhe o fundo antes. radius <= 0 equivale exatamente a FillRect (é o que
// preserva o visual clássico com Theme.Radius zero); raios maiores do que
// cabe no retângulo são reduzidos à metade do menor lado. Não aloca.
func FillRoundRect(dst *image.RGBA, r image.Rectangle, radius int, c color.RGBA) {
	radius = clampRadius(r, radius)
	if radius == 0 {
		FillRect(dst, r, c)
		return
	}
	// Miolo em três faixas cheias; os quatro blocos de canto (radius×radius)
	// são resolvidos por cobertura, pixel a pixel.
	FillRect(dst, image.Rect(r.Min.X, r.Min.Y+radius, r.Max.X, r.Max.Y-radius), c)
	FillRect(dst, image.Rect(r.Min.X+radius, r.Min.Y, r.Max.X-radius, r.Min.Y+radius), c)
	FillRect(dst, image.Rect(r.Min.X+radius, r.Max.Y-radius, r.Max.X-radius, r.Max.Y), c)
	roundCorners(dst, r, radius, 0, c, false)
}

// FillRoundRectOver é FillRoundRect com blending alfa (src-over) em toda a
// área, como FillRectOver: a cor é interpretada com alfa straight e a
// pré-multiplicação acontece aqui dentro; nos arcos a cobertura do canto
// multiplica o alfa. radius <= 0 equivale exatamente a FillRectOver. Não
// aloca.
func FillRoundRectOver(dst *image.RGBA, r image.Rectangle, radius int, c color.RGBA) {
	radius = clampRadius(r, radius)
	if radius == 0 {
		FillRectOver(dst, r, c)
		return
	}
	FillRectOver(dst, image.Rect(r.Min.X, r.Min.Y+radius, r.Max.X, r.Max.Y-radius), c)
	FillRectOver(dst, image.Rect(r.Min.X+radius, r.Min.Y, r.Max.X-radius, r.Min.Y+radius), c)
	FillRectOver(dst, image.Rect(r.Min.X+radius, r.Max.Y-radius, r.Max.X-radius, r.Max.Y), c)
	roundCorners(dst, r, radius, 0, c, true)
}

// StrokeRoundRect desenha o contorno de r com espessura w (em pixels) e
// cantos arredondados de raio radius, por dentro dos limites de r, com
// antialiasing nos arcos. radius <= 0 equivale exatamente a StrokeRect. Não
// aloca.
func StrokeRoundRect(dst *image.RGBA, r image.Rectangle, radius, w int, c color.RGBA) {
	if w <= 0 || r.Empty() {
		return
	}
	radius = clampRadius(r, radius)
	if radius == 0 {
		StrokeRect(dst, r, w, c)
		return
	}
	// Barras retas entre os cantos. Quando w > radius elas se sobrepõem
	// dentro dos blocos de canto; como FillRect sobrescreve com a mesma cor,
	// a sobreposição é inofensiva.
	FillRect(dst, image.Rect(r.Min.X+radius, r.Min.Y, r.Max.X-radius, r.Min.Y+w), c)
	FillRect(dst, image.Rect(r.Min.X+radius, r.Max.Y-w, r.Max.X-radius, r.Max.Y), c)
	FillRect(dst, image.Rect(r.Min.X, r.Min.Y+radius, r.Min.X+w, r.Max.Y-radius), c)
	FillRect(dst, image.Rect(r.Max.X-w, r.Min.Y+radius, r.Max.X, r.Max.Y-radius), c)
	inner := radius - w
	if inner < 0 {
		inner = 0
	}
	roundCorners(dst, r, radius, inner, c, false)
}

// roundCorners resolve os quatro blocos de canto (radius×radius) de r por
// cobertura: cada pixel recebe a fração do arco que o cobre — 1 no miolo, 0
// fora, com uma rampa de 1px na beirada (antialiasing). inner > 0 limita a
// cobertura ao anel entre os raios inner e radius (contornos); inner == 0
// preenche até o arco externo (nenhum pixel dos blocos fica a menos de meio
// pixel do centro, então o arco interno de raio zero nunca desconta). over
// escolhe entre sobrescrever (interpolação dst↔c pela cobertura) e blending
// src-over com a cobertura multiplicando o alfa straight de c. Não aloca.
func roundCorners(dst *image.RGBA, r image.Rectangle, radius, inner int, c color.RGBA, over bool) {
	clip := dst.Bounds()
	corners := [4]struct {
		block  image.Rectangle
		cx, cy float64
	}{
		{image.Rect(r.Min.X, r.Min.Y, r.Min.X+radius, r.Min.Y+radius), float64(r.Min.X + radius), float64(r.Min.Y + radius)},
		{image.Rect(r.Max.X-radius, r.Min.Y, r.Max.X, r.Min.Y+radius), float64(r.Max.X - radius), float64(r.Min.Y + radius)},
		{image.Rect(r.Min.X, r.Max.Y-radius, r.Min.X+radius, r.Max.Y), float64(r.Min.X + radius), float64(r.Max.Y - radius)},
		{image.Rect(r.Max.X-radius, r.Max.Y-radius, r.Max.X, r.Max.Y), float64(r.Max.X - radius), float64(r.Max.Y - radius)},
	}
	for _, corner := range corners {
		block := corner.block.Intersect(clip)
		if block.Empty() {
			continue
		}
		for y := block.Min.Y; y < block.Max.Y; y++ {
			py := float64(y) + 0.5
			i := dst.PixOffset(block.Min.X, y)
			for x := block.Min.X; x < block.Max.X; x, i = x+1, i+4 {
				px := float64(x) + 0.5
				d := math.Hypot(px-corner.cx, py-corner.cy)
				cov := float64(radius) - d + 0.5
				if cov <= 0 {
					continue
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
				}
				if cov <= 0 {
					continue
				}
				f := uint32(cov*255 + 0.5)
				p := dst.Pix[i : i+4 : i+4]
				switch {
				case over:
					// src-over em espaço pré-multiplicado, com a cobertura
					// reduzindo o alfa straight de c (mesma convenção de
					// FillRectOver).
					ea := (uint32(c.A)*f + 127) / 255
					inv := 255 - ea
					p[0] = uint8((uint32(c.R)*ea + uint32(p[0])*inv + 127) / 255)
					p[1] = uint8((uint32(c.G)*ea + uint32(p[1])*inv + 127) / 255)
					p[2] = uint8((uint32(c.B)*ea + uint32(p[2])*inv + 127) / 255)
					p[3] = uint8((ea*255 + uint32(p[3])*inv + 127) / 255)
				case f == 255:
					p[0], p[1], p[2], p[3] = c.R, c.G, c.B, c.A
				default:
					inv := 255 - f
					p[0] = uint8((uint32(c.R)*f + uint32(p[0])*inv + 127) / 255)
					p[1] = uint8((uint32(c.G)*f + uint32(p[1])*inv + 127) / 255)
					p[2] = uint8((uint32(c.B)*f + uint32(p[2])*inv + 127) / 255)
					p[3] = uint8((uint32(c.A)*f + uint32(p[3])*inv + 127) / 255)
				}
			}
		}
	}
}
