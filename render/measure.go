package render

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// measureCacheLimit é o teto de entradas do cache de medidas; ao estourar,
// o cache é esvaziado de uma vez (textos estáveis voltam no frame seguinte).
// measureCacheMaxLen deixa de fora textos longos demais para valer a chave.
const (
	measureCacheLimit  = 4096
	measureCacheMaxLen = 512
)

// Measurer mede texto para UMA face guardando o resultado por string: a
// interface mede os mesmos rótulos a cada layout e a cada frame, e medir de
// verdade (busca de glifo + kerning por rune) é a parte cara. Com o cache
// aquecido, Measure custa uma consulta de mapa e não aloca.
//
// O cache é limitado (measureCacheLimit entradas) e se descarta sozinho
// quando a face muda. Como todo o JUIGo, não é seguro para uso concorrente.
type Measurer struct {
	face   font.Face
	widths map[string]int
}

// Measure devolve a largura de s em pixels na face dada — o mesmo valor de
// MeasureText, memorizado.
func (m *Measurer) Measure(face font.Face, s string) int {
	if s == "" {
		return 0
	}
	if len(s) > measureCacheMaxLen {
		return MeasureText(face, s)
	}
	if m.face != face || m.widths == nil {
		m.face = face
		m.widths = make(map[string]int)
	}
	if w, ok := m.widths[s]; ok {
		return w
	}
	w := MeasureText(face, s)
	if len(m.widths) >= measureCacheLimit {
		clear(m.widths)
	}
	m.widths[s] = w
	return w
}

// PrefixWidths calcula, numa única passada, a largura em pixels de cada
// PREFIXO de runes: o resultado tem len(runes)+1 posições e out[i] é
// exatamente MeasureText(face, string(runes[:i])) — avanço e kerning
// acumulados com a mesma aritmética. É a tabela que dá posição de cursor,
// largura de seleção e clique→índice (busca binária) sem medir prefixo por
// prefixo. Reutiliza a capacidade de out; não aloca quando ela basta.
func PrefixWidths(face font.Face, runes []rune, out []int) []int {
	out = append(out[:0], 0)
	var x fixed.Int26_6
	prev := rune(-1)
	for _, r := range runes {
		// Mesma aritmética de font.MeasureString, rune a rune.
		if prev >= 0 {
			x += face.Kern(prev, r)
		}
		adv, _ := face.GlyphAdvance(r)
		x += adv
		prev = r
		out = append(out, x.Ceil())
	}
	return out
}

// IndexAtX devolve o índice da fronteira de widths (tabela de PrefixWidths)
// mais próxima de x: 0 antes do primeiro glifo, len(widths)-1 depois do
// último. Busca binária — widths é não decrescente.
func IndexAtX(widths []int, x int) int {
	if len(widths) == 0 || x <= widths[0] {
		return 0
	}
	lo, hi := 0, len(widths)-1
	if x >= widths[hi] {
		return hi
	}
	// Invariante: widths[lo] <= x < widths[hi].
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if widths[mid] <= x {
			lo = mid
		} else {
			hi = mid
		}
	}
	if x-widths[lo] <= widths[hi]-x {
		return lo
	}
	return hi
}

// MeasureRunes devolve a largura de runes em pixels — o mesmo valor de
// MeasureText(face, string(runes)), sem converter para string (não aloca).
func MeasureRunes(face font.Face, runes []rune) int {
	var x fixed.Int26_6
	prev := rune(-1)
	for _, r := range runes {
		if prev >= 0 {
			x += face.Kern(prev, r)
		}
		adv, _ := face.GlyphAdvance(r)
		x += adv
		prev = r
	}
	return x.Ceil()
}

// PrefixWidthsString é PrefixWidths para uma string: out[i] é a largura dos
// i primeiros caracteres (runes) de s.
func PrefixWidthsString(face font.Face, s string, out []int) []int {
	out = append(out[:0], 0)
	var x fixed.Int26_6
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			x += face.Kern(prev, r)
		}
		adv, _ := face.GlyphAdvance(r)
		x += adv
		prev = r
		out = append(out, x.Ceil())
	}
	return out
}
