package theme

import (
	"image/color"
	"math"
)

// oklch é uma cor no espaço OKLCh (Björn Ottosson, 2020): L é a
// luminosidade percebida (0 = preto, 1 = branco), C o croma (0 = neutro) e
// H o matiz em radianos. Derivar tons em OKLCh — e não em RGB — mantém
// passos de luminosidade parecidos para qualquer matiz: um "um pouco mais
// claro" do azul e do âmbar parecem igualmente mais claros.
type oklch struct{ L, C, H float64 }

// toLinear converte um canal sRGB (0–255) para a escala linear (0–1).
func toLinear(v uint8) float64 {
	s := float64(v) / 255
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// fromLinear converte um canal linear (0–1) de volta para sRGB (0–255),
// arredondando e saturando nas pontas.
func fromLinear(v float64) uint8 {
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
}

// toOKLCh converte uma cor sRGB opaca para OKLCh.
func toOKLCh(c color.RGBA) oklch {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L := 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	A := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return oklch{L: L, C: math.Hypot(A, B), H: math.Atan2(B, A)}
}

// rgb converte de volta para sRGB opaco. Cores fora do gamut têm o croma
// reduzido ao MAIOR valor que cabe (busca binária) — o matiz e a
// luminosidade, que são o que o olho mais percebe, ficam preservados.
func (o oklch) rgb() color.RGBA {
	if out, ok := o.linear(); ok {
		return out
	}
	lo, hi := 0.0, o.C
	for range 20 {
		mid := (lo + hi) / 2
		if _, ok := o.withC(mid).linear(); ok {
			lo = mid
		} else {
			hi = mid
		}
	}
	out, _ := o.withC(lo).linear()
	return out
}

// withC devolve a cor com outro croma.
func (o oklch) withC(c float64) oklch { return oklch{L: o.L, C: c, H: o.H} }

// withL devolve a cor com outra luminosidade (presa a 0–1).
func (o oklch) withL(l float64) oklch {
	return oklch{L: math.Max(0, math.Min(1, l)), C: o.C, H: o.H}
}

// linear converte para sRGB e informa se a cor cabia no gamut.
func (o oklch) linear() (color.RGBA, bool) {
	A, B := o.C*math.Cos(o.H), o.C*math.Sin(o.H)
	l := o.L + 0.3963377774*A + 0.2158037573*B
	m := o.L - 0.1055613458*A - 0.0638541728*B
	s := o.L - 0.0894841775*A - 1.2914855480*B
	l, m, s = l*l*l, m*m*m, s*s*s
	r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	const eps = 1e-4
	ok := r >= -eps && r <= 1+eps && g >= -eps && g <= 1+eps && b >= -eps && b <= 1+eps
	return color.RGBA{R: fromLinear(r), G: fromLinear(g), B: fromLinear(b), A: 0xFF}, ok
}

// luminance é a luminância relativa WCAG 2 de uma cor opaca.
func luminance(c color.RGBA) float64 {
	return 0.2126*toLinear(c.R) + 0.7152*toLinear(c.G) + 0.0722*toLinear(c.B)
}

// Contrast devolve a razão de contraste WCAG 2 entre duas cores opacas, de
// 1 (iguais) a 21 (preto sobre branco). A régua usual: 4.5 para texto
// comum, 3 para texto grande e para formas que precisam ser percebidas
// (anel de foco, marcas de controles).
func Contrast(a, b color.RGBA) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// ensureContrast move a luminosidade de c (em OKLCh, mantendo matiz e
// croma) até que o contraste com bg alcance min — escurecendo contra
// referências claras e clareando contra escuras (bg pode ser um fundo ou
// um rótulo que vai por cima de c). Se c já alcança, volta intacta.
func ensureContrast(c, bg color.RGBA, min float64) color.RGBA {
	if Contrast(c, bg) >= min {
		return c
	}
	o := toOKLCh(c)
	step := -0.01
	if luminance(bg) < 0.18 {
		step = 0.01
	}
	for range 100 {
		o = o.withL(o.L + step)
		out := o.rgb()
		if Contrast(out, bg) >= min || o.L <= 0 || o.L >= 1 {
			return out
		}
	}
	return o.rgb()
}

// withAlpha devolve c com o alfa dado (cores translúcidas do tema, como a
// lavagem de desabilitado).
func withAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}
