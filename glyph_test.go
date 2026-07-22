package juigo

import (
	"bytes"
	"image"
	"testing"

	"juigo/render"
)

// TestGlyphCacheParidadeComDrawer garante que o texto desenhado pelo cache de
// glyphs é pixel a pixel idêntico ao desenhado pelo font.Drawer (o caminho
// sem cache de render.DrawText).
func TestGlyphCacheParidadeComDrawer(t *testing.T) {
	th := newTestTheme(t)
	const s = "Ação métrica: çãé ÀÊÕ üï 123"
	dot := image.Pt(8, 24)

	mkBuf := func() *image.RGBA {
		buf := image.NewRGBA(image.Rect(0, 0, 320, 40))
		render.FillRect(buf, buf.Bounds(), th.Background)
		return buf
	}

	comCache := mkBuf()
	th.DrawText(comCache, s, dot, th.Text)

	semCache := mkBuf()
	render.DrawText(semCache, th.Face, s, dot, th.Text)

	if !bytes.Equal(comCache.Pix, semCache.Pix) {
		diff := 0
		for i := range comCache.Pix {
			if comCache.Pix[i] != semCache.Pix[i] {
				diff++
			}
		}
		t.Fatalf("cache de glyphs divergiu do font.Drawer em %d bytes", diff)
	}
}

// TestGlyphCacheSemAlocacao garante que, com o cache aquecido, desenhar texto
// não aloca — requisito do caminho quente de desenho.
func TestGlyphCacheSemAlocacao(t *testing.T) {
	th := newTestTheme(t)
	const s = "Ação métrica: çãé 123"
	buf := image.NewRGBA(image.Rect(0, 0, 320, 40))
	dot := image.Pt(8, 24)

	// Aquece o cache (primeira rasterização de cada glyph aloca).
	th.DrawText(buf, s, dot, th.Text)

	allocs := testing.AllocsPerRun(100, func() {
		th.DrawText(buf, s, dot, th.Text)
	})
	if allocs != 0 {
		t.Fatalf("DrawText alocou %.1f vezes por chamada com cache aquecido, esperado 0", allocs)
	}
}

// TestFillRectSemAlocacao cobre a outra primitiva do caminho quente.
func TestFillRectSemAlocacao(t *testing.T) {
	th := newTestTheme(t)
	buf := image.NewRGBA(image.Rect(0, 0, 320, 200))
	allocs := testing.AllocsPerRun(100, func() {
		render.FillRect(buf, buf.Bounds(), th.Background)
		render.StrokeRect(buf, buf.Bounds(), 2, th.Text)
	})
	if allocs != 0 {
		t.Fatalf("FillRect/StrokeRect alocaram %.1f vezes por chamada, esperado 0", allocs)
	}
}
