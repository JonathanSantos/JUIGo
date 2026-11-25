package theme

import (
	"fmt"
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/opentype"

	"github.com/JonathanSantos/JUIGo/render"
)

// goBoldFnt e goMonoBoldFnt são os parses únicos das fontes de display
// (single-threaded por contrato).
var (
	goBoldFnt     *opentype.Font
	goMonoBoldFnt *opentype.Font
)

// GoMonoBold devolve a Go Mono Bold (x/image, já dependência), interpretada
// uma única vez — opção de display com cara de ferramenta (UseDisplayFont).
func GoMonoBold() (*opentype.Font, error) {
	if goMonoBoldFnt != nil {
		return goMonoBoldFnt, nil
	}
	f, err := opentype.Parse(gomonobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a Go Mono Bold embutida: %w", err)
	}
	goMonoBoldFnt = f
	return f, nil
}

// GoBold devolve a Go Bold (x/image, já dependência), interpretada uma única
// vez — a fonte de display dos temas Classic/ClassicDark e o recurso de
// quem não definir uma: hierarquia por peso, sem asset novo.
func GoBold() (*opentype.Font, error) {
	if goBoldFnt != nil {
		return goBoldFnt, nil
	}
	f, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a Go Bold embutida: %w", err)
	}
	goBoldFnt = f
	return f, nil
}

// TextFont é uma fonte proporcional pronta para um PAPEL TIPOGRÁFICO do
// tema (Título, Subtítulo, Legenda): face na escala corrente, métricas e
// cache de glyphs próprios. O tema constrói os papéis no SetScale; widgets
// desenham por ela sem medir nem alocar no caminho quente.
type TextFont struct {
	// Face é a face rasterizada; não guarde referências entre mudanças de
	// escala.
	Face font.Face

	size       float64
	ascent     int
	lineHeight int
	cache      *render.GlyphCache
	measurer   render.Measurer
}

// newTextFont rasteriza fnt no tamanho lógico dado × escala.
func newTextFont(fnt *opentype.Font, size, scale float64) (*TextFont, error) {
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao criar face de texto (%.1fpt, escala %v): %w", size, scale, err)
	}
	m := face.Metrics()
	return &TextFont{
		Face:       face,
		size:       size,
		ascent:     m.Ascent.Ceil(),
		lineHeight: m.Height.Ceil(),
		cache:      render.NewGlyphCache(face),
	}, nil
}

// Draw desenha s com a origem da baseline em dot, pelo cache de glyphs.
func (f *TextFont) Draw(dst *image.RGBA, s string, dot image.Point, c color.RGBA) {
	f.cache.DrawString(dst, s, dot, c)
}

// Measure devolve a largura de s em pixels.
func (f *TextFont) Measure(s string) int {
	return f.measurer.Measure(f.Face, s)
}

// LineHeight devolve a altura de linha em pixels.
func (f *TextFont) LineHeight() int { return f.lineHeight }

// Ascent devolve o ascent em pixels.
func (f *TextFont) Ascent() int { return f.ascent }

// Size devolve o tamanho lógico (pontos) do papel.
func (f *TextFont) Size() float64 { return f.size }

// Title devolve a fonte do papel de TÍTULO (TitleSize, fonte de display).
func (t *Theme) Title() *TextFont { return t.title }

// Subtitle devolve a fonte do papel de SUBTÍTULO (SubtitleSize, fonte de
// display).
func (t *Theme) Subtitle() *TextFont { return t.subtitle }

// Caption devolve a fonte do papel de LEGENDA (CaptionSize, a MESMA fonte
// do corpo, menor).
func (t *Theme) Caption() *TextFont { return t.caption }

// UseDisplayFont troca a fonte de DISPLAY do tema em runtime — títulos e
// subtítulos são reconstruídos na escala corrente; corpo e legenda seguem
// na fonte regular:
//
//	bold, _ := theme.GoBold()
//	th.UseDisplayFont(bold)
func (t *Theme) UseDisplayFont(fnt *opentype.Font) error {
	if fnt == nil || fnt == t.displayFnt {
		return nil
	}
	title, err := newTextFont(fnt, t.roleSize(t.TitleSize), t.scale)
	if err != nil {
		return err
	}
	subtitle, err := newTextFont(fnt, t.roleSize(t.SubtitleSize), t.scale)
	if err != nil {
		return err
	}
	t.displayFnt = fnt
	t.title = title
	t.subtitle = subtitle
	return nil
}

// roleSize protege papéis não configurados: zero ou negativo cai no
// tamanho do corpo.
func (t *Theme) roleSize(v float64) float64 {
	if v <= 0 {
		return t.FontSize
	}
	return v
}

// buildRoles constrói os papéis tipográficos na escala dada (chamado pelo
// SetScale). Sem fonte de display definida, usa a Go Bold.
func (t *Theme) buildRoles(scale float64) error {
	if t.displayFnt == nil {
		display, err := GoBold()
		if err != nil {
			return err
		}
		t.displayFnt = display
	}
	title, err := newTextFont(t.displayFnt, t.roleSize(t.TitleSize), scale)
	if err != nil {
		return err
	}
	subtitle, err := newTextFont(t.displayFnt, t.roleSize(t.SubtitleSize), scale)
	if err != nil {
		return err
	}
	caption, err := newTextFont(t.fnt, t.roleSize(t.CaptionSize), scale)
	if err != nil {
		return err
	}
	t.title, t.subtitle, t.caption = title, subtitle, caption
	return nil
}
