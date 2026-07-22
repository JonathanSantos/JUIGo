package render

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// glyphColor e glyphSrc são reutilizados pelo blit de glyphs, pelo mesmo
// motivo de textSrc: evitar alocação por frame. O Uniform guarda um ponteiro
// estável para a cor — atribuir color.RGBA direto à interface color.Color
// faria boxing (uma alocação) a cada chamada. Single-threaded.
var (
	glyphColor color.RGBA
	glyphSrc   = &image.Uniform{C: &glyphColor}
)

// cachedGlyph guarda a máscara alpha rasterizada de um glyph e as métricas
// necessárias para posicioná-lo a partir da origem (dot) na baseline.
type cachedGlyph struct {
	// mask é a máscara alpha própria do cache (copiada, pois a face
	// reutiliza o buffer interno entre chamadas de Glyph).
	mask *image.Alpha
	// bounds é o retângulo do glyph relativo ao dot (Y negativo = acima da
	// baseline).
	bounds image.Rectangle
	// advance é o avanço horizontal do glyph.
	advance fixed.Int26_6
	// ok indica se a face produziu um glyph para a rune.
	ok bool
}

// GlyphCache rasteriza cada glyph de uma face UMA única vez e o reutiliza em
// todos os desenhos seguintes, eliminando as alocações de rasterização do
// caminho quente. Os glyphs são rasterizados em posições inteiras de pixel —
// com font.HintingFull os avanços também são inteiros, então o resultado é
// idêntico ao de um font.Drawer.
//
// O cache cresce sob demanda, uma entrada por rune, e vale para uma única
// face: ao trocar de fonte ou tamanho (ex.: mudança de escala HiDPI), crie um
// cache novo. Como todo o JUIGo, não é seguro para uso concorrente.
type GlyphCache struct {
	face   font.Face
	glyphs map[rune]*cachedGlyph
}

// NewGlyphCache cria um cache vazio para a face dada.
func NewGlyphCache(face font.Face) *GlyphCache {
	return &GlyphCache{face: face, glyphs: make(map[rune]*cachedGlyph)}
}

// Face devolve a face para a qual este cache rasteriza.
func (c *GlyphCache) Face() font.Face {
	return c.face
}

// lookup devolve o glyph de r, rasterizando-o na primeira consulta.
func (c *GlyphCache) lookup(r rune) *cachedGlyph {
	if g, ok := c.glyphs[r]; ok {
		return g
	}
	g := &cachedGlyph{}
	dr, mask, maskp, advance, ok := c.face.Glyph(fixed.Point26_6{}, r)
	if ok {
		g.ok = true
		g.bounds = dr
		g.advance = advance
		// Copia a máscara: a face reutiliza o buffer devolvido por Glyph.
		g.mask = image.NewAlpha(image.Rect(0, 0, dr.Dx(), dr.Dy()))
		draw.Draw(g.mask, g.mask.Bounds(), mask, maskp, draw.Src)
	}
	c.glyphs[r] = g
	return g
}

// DrawString desenha s em dst com a baseline começando em dot e cor col.
// Aplica kerning e, com o cache aquecido, não aloca.
func (c *GlyphCache) DrawString(dst *image.RGBA, s string, dot image.Point, col color.RGBA) {
	glyphColor = col
	x := fixed.I(dot.X)
	prev := rune(-1)
	for _, r := range s {
		g := c.lookup(r)
		if !g.ok {
			// Como font.Drawer: rune sem glyph é pulada.
			continue
		}
		if prev >= 0 {
			x += c.face.Kern(prev, r)
		}
		target := g.bounds.Add(image.Pt(x.Round(), dot.Y))
		draw.DrawMask(dst, target, glyphSrc, image.Point{}, g.mask, image.Point{}, draw.Over)
		x += g.advance
		prev = r
	}
}
