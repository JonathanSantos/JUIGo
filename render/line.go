package render

import (
	"image"
	"image/color"
	"math"
)

// StrokeLine desenha o segmento de a a b com espessura w (pixels),
// antialiasing nas bordas e pontas redondas. As coordenadas são CENTROS de
// pixel: uma linha horizontal de espessura 1 acende exatamente a linha dos
// pontos, sem meio-tom — eixos de gráfico saem nítidos e diagonais saem
// suaves. A varredura anda pelo eixo maior do segmento com uma janela
// perpendicular estreita (custo ~comprimento × espessura). Não aloca.
func StrokeLine(dst *image.RGBA, a, b image.Point, w int, col color.RGBA) {
	if w <= 0 {
		return
	}
	clip := dst.Bounds()
	if clip.Empty() {
		return
	}
	half := float64(w) / 2
	pad := int(math.Ceil(half)) + 1

	ax, ay := float64(a.X), float64(a.Y)
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	len2 := dx*dx + dy*dy

	// span é a meia-janela no eixo menor que garante cobrir a espessura
	// mesmo inclinada (folga conservadora; pixels além da rampa saem com
	// cobertura zero e são pulados).
	major := math.Max(math.Abs(dx), math.Abs(dy))
	span := half + 1.5
	if major > 0 {
		span = (half + 1.5) * math.Hypot(dx, dy) / major
	}

	blend := func(x, y int) {
		px, py := float64(x)-ax, float64(y)-ay
		t := 0.0
		if len2 > 0 {
			t = (px*dx + py*dy) / len2
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		ddx, ddy := px-t*dx, py-t*dy
		cov := half + 0.5 - math.Hypot(ddx, ddy)
		if cov <= 0 {
			return
		}
		if cov > 1 {
			cov = 1
		}
		blendPixel(dst, x, y, col, cov)
	}

	if math.Abs(dx) >= math.Abs(dy) {
		x0 := max(min(a.X, b.X)-pad, clip.Min.X)
		x1 := min(max(a.X, b.X)+pad, clip.Max.X-1)
		for x := x0; x <= x1; x++ {
			t := 0.0
			if dx != 0 {
				t = (float64(x) - ax) / dx
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
			}
			yc := ay + t*dy
			y0 := max(int(math.Floor(yc-span)), clip.Min.Y)
			y1 := min(int(math.Ceil(yc+span)), clip.Max.Y-1)
			for y := y0; y <= y1; y++ {
				blend(x, y)
			}
		}
		return
	}
	y0 := max(min(a.Y, b.Y)-pad, clip.Min.Y)
	y1 := min(max(a.Y, b.Y)+pad, clip.Max.Y-1)
	for y := y0; y <= y1; y++ {
		t := 0.0
		if dy != 0 {
			t = (float64(y) - ay) / dy
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		xc := ax + t*dx
		x0 := max(int(math.Floor(xc-span)), clip.Min.X)
		x1 := min(int(math.Ceil(xc+span)), clip.Max.X-1)
		for x := x0; x <= x1; x++ {
			blend(x, y)
		}
	}
}

// StrokePolyline liga os pontos na sequência com StrokeLine — a base das
// séries de gráficos. Nas juntas os caps redondos se sobrepõem (com cor
// opaca, invisível; com alfa parcial, a junta escurece de leve — limitação
// documentada). Não aloca.
func StrokePolyline(dst *image.RGBA, pts []image.Point, w int, col color.RGBA) {
	for i := 1; i < len(pts); i++ {
		StrokeLine(dst, pts[i-1], pts[i], w, col)
	}
}

// blendPixel mistura col no pixel (x,y) com a cobertura dada (0..1),
// assumindo (x,y) dentro dos bounds de dst.
func blendPixel(dst *image.RGBA, x, y int, col color.RGBA, cov float64) {
	f := uint32(cov*255 + 0.5)
	if f == 0 {
		return
	}
	i := dst.PixOffset(x, y)
	p := dst.Pix[i : i+4 : i+4]
	if f >= 255 {
		p[0], p[1], p[2], p[3] = col.R, col.G, col.B, col.A
		return
	}
	inv := 255 - f
	p[0] = uint8((uint32(col.R)*f + uint32(p[0])*inv + 127) / 255)
	p[1] = uint8((uint32(col.G)*f + uint32(p[1])*inv + 127) / 255)
	p[2] = uint8((uint32(col.B)*f + uint32(p[2])*inv + 127) / 255)
	p[3] = uint8((uint32(col.A)*f + uint32(p[3])*inv + 127) / 255)
}
