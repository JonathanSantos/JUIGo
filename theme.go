package juigo

import (
	_ "embed"
	"fmt"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"juigo/render"
)

// Fonte Go Regular (https://go.dev/blog/go-fonts), licença BSD — a mesma do
// projeto Go. Embutida para que binários JUIGo não dependam de fontes do
// sistema.
//
//go:embed assets/GoRegular.ttf
var embeddedFontTTF []byte

// Theme centraliza todas as cores, a fonte e os espaçamentos usados pelos
// widgets. Nenhum widget deve ter cor ou tamanho hardcoded: tudo vem daqui.
type Theme struct {
	// Background é a cor de fundo da janela.
	Background color.RGBA
	// Text é a cor padrão de texto.
	Text color.RGBA
	// Placeholder é a cor do texto de sugestão do Input vazio.
	Placeholder color.RGBA

	// ButtonNormal, ButtonHover e ButtonPressed são as cores de fundo do
	// Button em cada estado.
	ButtonNormal  color.RGBA
	ButtonHover   color.RGBA
	ButtonPressed color.RGBA
	// ButtonText é a cor do rótulo do Button.
	ButtonText color.RGBA

	// InputBackground é o fundo do campo de texto.
	InputBackground color.RGBA
	// InputBorder é a borda do campo sem foco; InputBorderFocused, com foco.
	InputBorder        color.RGBA
	InputBorderFocused color.RGBA
	// Cursor é a cor da linha vertical do cursor de texto.
	Cursor color.RGBA
	// FocusOutline é o contorno de indicação de foco em widgets focáveis
	// não-textuais (ex.: Button).
	FocusOutline color.RGBA

	// Face é a fonte usada por todos os widgets.
	Face font.Face
	// FontSize é o tamanho nominal da fonte, em pixels.
	FontSize float64

	// Padding é o espaço interno padrão dos widgets, em pixels.
	Padding int
	// Spacing é o espaço padrão entre widgets em containers de layout.
	Spacing int
	// BorderWidth é a espessura padrão de bordas, em pixels.
	BorderWidth int
	// InputMinWidth é a largura preferida mínima do Input, em pixels.
	InputMinWidth int

	ascent     int
	lineHeight int
}

// DefaultTheme constrói o tema padrão do JUIGo, interpretando a fonte
// embutida. Falhas na fonte são devolvidas como erro.
func DefaultTheme() (*Theme, error) {
	parsed, err := opentype.Parse(embeddedFontTTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a fonte embutida: %w", err)
	}
	const size = 16
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao criar a face da fonte: %w", err)
	}
	m := face.Metrics()

	return &Theme{
		Background:  color.RGBA{R: 0xF2, G: 0xF3, B: 0xF5, A: 0xFF},
		Text:        color.RGBA{R: 0x1F, G: 0x23, B: 0x28, A: 0xFF},
		Placeholder: color.RGBA{R: 0x9A, G: 0xA0, B: 0xA8, A: 0xFF},

		ButtonNormal:  color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		ButtonHover:   color.RGBA{R: 0x60, G: 0xA5, B: 0xFA, A: 0xFF},
		ButtonPressed: color.RGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF},
		ButtonText:    color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},

		InputBackground:    color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		InputBorder:        color.RGBA{R: 0xB4, G: 0xBA, B: 0xC2, A: 0xFF},
		InputBorderFocused: color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		Cursor:             color.RGBA{R: 0x1F, G: 0x23, B: 0x28, A: 0xFF},
		FocusOutline:       color.RGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF},

		Face:     face,
		FontSize: size,

		Padding:       8,
		Spacing:       8,
		BorderWidth:   1,
		InputMinWidth: 220,

		ascent:     m.Ascent.Ceil(),
		lineHeight: m.Height.Ceil(),
	}, nil
}

// MeasureString devolve a largura de s em pixels com a fonte do tema. É a
// ÚNICA fonte de verdade para largura de texto: layout e posicionamento de
// cursor devem passar por aqui.
func (t *Theme) MeasureString(s string) int {
	return render.MeasureText(t.Face, s)
}

// LineHeight devolve a altura de uma linha de texto, em pixels.
func (t *Theme) LineHeight() int {
	return t.lineHeight
}

// Ascent devolve a distância do topo da linha até a baseline, em pixels.
// Útil para converter um Y de topo em um Y de baseline para DrawText.
func (t *Theme) Ascent() int {
	return t.ascent
}
