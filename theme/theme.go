package theme

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"

	"github.com/JonathanSantos/JUIGo/render"
)

// Fonte Go Regular (https://go.dev/blog/go-fonts), licença BSD — a mesma do
// projeto Go. Embutida para que binários JUIGo não dependam de fontes do
// sistema.
//
//go:embed assets/GoRegular.ttf
var embeddedFontTTF []byte

// Fira Code (https://github.com/tonsky/FiraCode), licença SIL OFL 1.1
// (theme/assets/FiraCode-OFL.txt) — alternativa embutida de fonte mono
// para o CodeEditor (ver FiraCode e UseMonoFont). Sem shaping, as
// ligaduras não se formam: os glifos saem um a um.
//
//go:embed assets/FiraCode-Regular.ttf
var firaCodeTTF []byte

// firaCode é o parse único da Fira Code (single-threaded por contrato).
var firaCode *opentype.Font

// FiraCode devolve a fonte Fira Code embutida, interpretada uma única vez —
// troque a mono do tema com UseMonoFont(theme.FiraCode()).
func FiraCode() (*opentype.Font, error) {
	if firaCode != nil {
		return firaCode, nil
	}
	f, err := opentype.Parse(firaCodeTTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a Fira Code embutida: %w", err)
	}
	firaCode = f
	return f, nil
}

// MonoFont é uma fonte monoespaçada pronta para desenhar: face na escala
// corrente, métricas e cache de glyphs próprio. O tema carrega a padrão
// (Mono) e fabrica variações de tamanho (MonoSized) — cada CodeEditor pode
// ter a sua.
type MonoFont struct {
	// Face é a face rasterizada; não guarde referências entre mudanças de
	// escala.
	Face font.Face

	size       float64
	ascent     int
	lineHeight int
	advance    int
	cache      *render.GlyphCache
}

// newMonoFont rasteriza fnt no tamanho lógico dado × escala.
func newMonoFont(fnt *opentype.Font, size, scale float64) (*MonoFont, error) {
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao criar face mono (%.1fpt, escala %v): %w", size, scale, err)
	}
	m := face.Metrics()
	return &MonoFont{
		Face:       face,
		size:       size,
		ascent:     m.Ascent.Ceil(),
		lineHeight: m.Height.Ceil(),
		advance:    render.MeasureText(face, "0"),
		cache:      render.NewGlyphCache(face),
	}, nil
}

// Draw desenha s com esta fonte, baseline em dot — cache de glyphs, sem
// alocação no caminho quente.
func (m *MonoFont) Draw(dst *image.RGBA, s string, dot image.Point, c color.RGBA) {
	m.cache.DrawString(dst, s, dot, c)
}

// Measure devolve a largura de s em pixels.
func (m *MonoFont) Measure(s string) int {
	return render.MeasureText(m.Face, s)
}

// LineHeight devolve a altura de linha em pixels.
func (m *MonoFont) LineHeight() int { return m.lineHeight }

// Ascent devolve o ascent em pixels.
func (m *MonoFont) Ascent() int { return m.ascent }

// Advance devolve a largura da célula em pixels — todo glifo coberto
// avança isto (a base da conta coluna↔pixel).
func (m *MonoFont) Advance() int { return m.advance }

// Size devolve o tamanho lógico (pontos) desta variação.
func (m *MonoFont) Size() float64 { return m.size }

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
	// ButtonBorder é a cor da borda do Button; alfa zero desliga a borda
	// (o visual clássico não tem borda de botão).
	ButtonBorder color.RGBA

	// InputBackground é o fundo do campo de texto.
	InputBackground color.RGBA
	// InputBorder é a borda do campo sem foco; InputBorderFocused, com foco.
	InputBorder        color.RGBA
	InputBorderFocused color.RGBA
	// Cursor é a cor da linha vertical do cursor de texto.
	Cursor color.RGBA
	// Selection é o fundo da seleção de texto no Input.
	Selection color.RGBA
	// FocusOutline é o contorno de indicação de foco em widgets focáveis
	// não-textuais (ex.: Button).
	FocusOutline color.RGBA
	// Accent é a cor de destaque dos controles: a marca do Checkbox e o
	// trilho ativo e a alça do Slider.
	Accent color.RGBA
	// HoverBackground é o fundo do item sob o ponteiro em listas (ex.: o
	// popup do Dropdown).
	HoverBackground color.RGBA
	// TooltipBackground e TooltipText são as cores da caixa de dica.
	TooltipBackground color.RGBA
	TooltipText       color.RGBA
	// Danger é a cor de erros e ações destrutivas (mensagens de validação).
	Danger color.RGBA
	// Syntax é a paleta de highlight do CodeEditor (classes léxicas).
	Syntax SyntaxPalette
	// CurrentLine é a faixa sutil atrás da linha do cursor no CodeEditor.
	CurrentLine color.RGBA

	// Face é a fonte já rasterizada na escala atual (ver SetScale). É
	// reconstruída a cada mudança de escala; não guarde referências a ela.
	Face font.Face
	// MonoFace é a fonte MONOESPAÇADA do tema (Go Mono, do mesmo projeto da
	// fonte regular), na escala corrente — editores de código medem e
	// desenham por ela (DrawMono/MeasureMono). Reconstruída no SetScale.
	MonoFace font.Face
	// FontSize é o tamanho LÓGICO da fonte (independente de escala). A face
	// é criada com FontSize × escala.
	FontSize float64

	// Padding é o espaço interno padrão dos widgets, em unidades LÓGICAS.
	// No desenho e no layout, use PaddingPx (ou Px) para converter.
	Padding int
	// Spacing é o espaço padrão entre widgets em containers de layout, em
	// unidades lógicas (use SpacingPx).
	Spacing int
	// BorderWidth é a espessura padrão de bordas, em unidades lógicas (use
	// BorderPx).
	BorderWidth int
	// Radius é o raio dos cantos arredondados dos controles, em unidades
	// lógicas (use RadiusPx). Zero desliga o arredondamento e reproduz o
	// visual clássico (retângulos retos) pixel a pixel — ver Classic.
	Radius int
	// InputMinWidth é a largura preferida mínima do Input, em unidades
	// lógicas (use InputMinWidthPx).
	InputMinWidth int
	// SliderMinWidth é a largura preferida mínima do Slider, em unidades
	// lógicas.
	SliderMinWidth int
	// SliderTrackThickness é a espessura do trilho do Slider, em unidades
	// lógicas.
	SliderTrackThickness int
	// SliderHandleSize é o lado da alça do Slider, em unidades lógicas.
	SliderHandleSize int
	// ScrollStep é quantas unidades lógicas o conteúdo rola por passo da
	// roda do mouse.
	ScrollStep int
	// ScrollbarWidth é a largura do indicador de rolagem, em unidades
	// lógicas.
	ScrollbarWidth int

	// CaretBlink é o intervalo de piscada do cursor de texto; zero desliga
	// a piscada (cursor sempre visível).
	CaretBlink time.Duration
	// TooltipDelay é quanto tempo o ponteiro precisa pausar sobre um widget
	// com Tooltip até a dica aparecer.
	TooltipDelay time.Duration
	// ToastDuration é quanto tempo um toast (aviso transitório) fica na
	// tela antes de sumir sozinho.
	ToastDuration time.Duration
	// TextAreaMinLines é a altura preferida da TextArea, em linhas.
	TextAreaMinLines int
	// SpinnerStep é o intervalo entre quadros do indicador de loading.
	SpinnerStep time.Duration
	// Backdrop é a cor translúcida do pano de fundo do Modal.
	Backdrop color.RGBA
	// DisabledWash é a lavagem translúcida aplicada sobre widgets
	// desabilitados, esmaecendo-os em direção ao fundo.
	DisabledWash color.RGBA

	fnt        *opentype.Font
	scale      float64
	ascent     int
	lineHeight int
	cache      *render.GlyphCache

	monoFnt *opentype.Font
	mono    *MonoFont
}

// SyntaxPalette são as cores das classes léxicas do highlight de código
// (ver widget.SyntaxStyle); texto comum usa Theme.Text.
type SyntaxPalette struct {
	// Keyword colore palavras-chave da linguagem.
	Keyword color.RGBA
	// String colore literais de texto (incluindo runas e raw strings).
	String color.RGBA
	// Number colore literais numéricos.
	Number color.RGBA
	// Comment colore comentários.
	Comment color.RGBA
	// Builtin colore tipos e identificadores embutidos da linguagem.
	Builtin color.RGBA
}

// Default constrói o tema padrão do JUIGo na escala 1, interpretando a
// fonte embutida. Falhas na fonte são devolvidas como erro. O App ajusta a
// escala para o monitor via SetScale.
func Default() (*Theme, error) {
	parsed, err := opentype.Parse(embeddedFontTTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a fonte embutida: %w", err)
	}

	mono, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("juigo: falha ao interpretar a fonte mono embutida: %w", err)
	}

	t := &Theme{
		Background:  color.RGBA{R: 0xF2, G: 0xF3, B: 0xF5, A: 0xFF},
		Text:        color.RGBA{R: 0x1F, G: 0x23, B: 0x28, A: 0xFF},
		Placeholder: color.RGBA{R: 0x9A, G: 0xA0, B: 0xA8, A: 0xFF},

		ButtonNormal:  color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		ButtonHover:   color.RGBA{R: 0x60, G: 0xA5, B: 0xFA, A: 0xFF},
		ButtonPressed: color.RGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF},
		ButtonText:    color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		ButtonBorder:  color.RGBA{R: 0x25, G: 0x63, B: 0xEB, A: 0xFF},

		InputBackground:    color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		InputBorder:        color.RGBA{R: 0xB4, G: 0xBA, B: 0xC2, A: 0xFF},
		InputBorderFocused: color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		Cursor:             color.RGBA{R: 0x1F, G: 0x23, B: 0x28, A: 0xFF},
		Selection:          color.RGBA{R: 0xBF, G: 0xDB, B: 0xFE, A: 0xFF},
		FocusOutline:       color.RGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF},
		Accent:             color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF},
		HoverBackground:    color.RGBA{R: 0xE4, G: 0xEC, B: 0xF9, A: 0xFF},
		TooltipBackground:  color.RGBA{R: 0x2E, G: 0x33, B: 0x3B, A: 0xFF},
		TooltipText:        color.RGBA{R: 0xF5, G: 0xF7, B: 0xFA, A: 0xFF},
		Danger:             color.RGBA{R: 0xDC, G: 0x26, B: 0x26, A: 0xFF},
		Syntax: SyntaxPalette{
			Keyword: color.RGBA{R: 0xCF, G: 0x22, B: 0x2E, A: 0xFF},
			String:  color.RGBA{R: 0x0A, G: 0x30, B: 0x69, A: 0xFF},
			Number:  color.RGBA{R: 0x05, G: 0x50, B: 0xAE, A: 0xFF},
			Comment: color.RGBA{R: 0x6E, G: 0x77, B: 0x81, A: 0xFF},
			Builtin: color.RGBA{R: 0x95, G: 0x38, B: 0x00, A: 0xFF},
		},
		CurrentLine: color.RGBA{R: 0xF2, G: 0xF5, B: 0xFA, A: 0xFF},

		FontSize: 16,

		Padding:              8,
		Spacing:              8,
		BorderWidth:          1,
		Radius:               4,
		InputMinWidth:        220,
		SliderMinWidth:       160,
		SliderTrackThickness: 4,
		SliderHandleSize:     16,
		ScrollStep:           40,
		ScrollbarWidth:       4,

		CaretBlink:       530 * time.Millisecond,
		TooltipDelay:     600 * time.Millisecond,
		ToastDuration:    2500 * time.Millisecond,
		TextAreaMinLines: 4,
		SpinnerStep:      250 * time.Millisecond,
		Backdrop:         color.RGBA{A: 0x66},
		DisabledWash:     color.RGBA{R: 0xF2, G: 0xF3, B: 0xF5, A: 0x99},

		fnt:     parsed,
		monoFnt: mono,
	}
	if err := t.SetScale(1); err != nil {
		return nil, err
	}
	return t, nil
}

// SetScale reconstrói a fonte, as métricas e o cache de glyphs para a escala
// dada, em pixels por unidade lógica (1 = tela comum, 2 = retina). O App a
// chama com a escala de conteúdo da janela e a cada mudança de monitor. Os
// campos lógicos do tema (FontSize, Padding, ...) não são alterados: a
// conversão para pixels acontece na face e nos métodos Px/*Px.
func (t *Theme) SetScale(scale float64) error {
	if scale <= 0 {
		return fmt.Errorf("juigo: escala de tema inválida: %v", scale)
	}
	face, err := opentype.NewFace(t.fnt, &opentype.FaceOptions{
		Size:    t.FontSize * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("juigo: falha ao criar a face da fonte na escala %v: %w", scale, err)
	}
	m := face.Metrics()
	t.Face = face
	t.scale = scale
	t.ascent = m.Ascent.Ceil()
	t.lineHeight = m.Height.Ceil()
	t.cache = render.NewGlyphCache(face)

	mono, err := newMonoFont(t.monoFnt, t.FontSize, scale)
	if err != nil {
		return err
	}
	t.mono = mono
	t.MonoFace = mono.Face
	return nil
}

// Mono devolve a fonte mono padrão do tema (tamanho FontSize, na escala
// corrente).
func (t *Theme) Mono() *MonoFont {
	return t.mono
}

// MonoSized fabrica uma variação da fonte mono no tamanho LÓGICO dado
// (pontos), na escala corrente — cada CodeEditor pode ter o próprio
// tamanho. A variação vale até a próxima mudança de escala ou de fonte
// mono (detectável por Theme.MonoFace).
func (t *Theme) MonoSized(size float64) (*MonoFont, error) {
	if size <= 0 {
		return t.mono, nil
	}
	return newMonoFont(t.monoFnt, size, t.scale)
}

// UseMonoFont troca a FONTE mono do tema (ex.: theme.FiraCode()) e refaz a
// face padrão na escala corrente; editores percebem pelo MonoFace no
// próximo frame. A fonte regular dos demais widgets não muda.
func (t *Theme) UseMonoFont(fnt *opentype.Font) error {
	if fnt == nil || fnt == t.monoFnt {
		return nil
	}
	mono, err := newMonoFont(fnt, t.FontSize, t.scale)
	if err != nil {
		return err
	}
	t.monoFnt = fnt
	t.mono = mono
	t.MonoFace = mono.Face
	return nil
}

// Clone devolve uma cópia independente do tema, na mesma escala: os campos
// de cores e métricas são copiados, a fonte interpretada é compartilhada
// (imutável) e a face e o cache de glyphs são reconstruídos. Cada JANELA
// precisa do próprio *Theme — a escala segue o monitor da janela, e
// SetScale num tema compartilhado desregularia as outras.
func (t *Theme) Clone() (*Theme, error) {
	c := *t
	if err := c.SetScale(t.scale); err != nil {
		return nil, err
	}
	return &c, nil
}

// Scale devolve a escala atual do tema (pixels por unidade lógica).
func (t *Theme) Scale() float64 {
	return t.scale
}

// Px converte um valor em unidades lógicas para pixels na escala atual,
// arredondando para o inteiro mais próximo.
func (t *Theme) Px(v int) int {
	return int(math.Round(float64(v) * t.scale))
}

// PaddingPx devolve Padding convertido para pixels.
func (t *Theme) PaddingPx() int {
	return t.Px(t.Padding)
}

// SpacingPx devolve Spacing convertido para pixels.
func (t *Theme) SpacingPx() int {
	return t.Px(t.Spacing)
}

// BorderPx devolve BorderWidth convertido para pixels, no mínimo 1 para que
// bordas nunca desapareçam em escalas baixas.
func (t *Theme) BorderPx() int {
	if px := t.Px(t.BorderWidth); px > 1 {
		return px
	}
	return 1
}

// RadiusPx devolve Radius convertido para pixels na escala atual. Zero
// permanece zero (cantos retos) em qualquer escala.
func (t *Theme) RadiusPx() int {
	return t.Px(t.Radius)
}

// InputMinWidthPx devolve InputMinWidth convertido para pixels.
func (t *Theme) InputMinWidthPx() int {
	return t.Px(t.InputMinWidth)
}

// DrawText desenha s em dst com a fonte do tema, baseline em dot e cor c,
// usando o cache de glyphs: cada glyph é rasterizado uma única vez, e o
// caminho quente de desenho não aloca.
func (t *Theme) DrawText(dst *image.RGBA, s string, dot image.Point, c color.RGBA) {
	t.cache.DrawString(dst, s, dot, c)
}

// MeasureString devolve a largura de s em pixels com a fonte do tema. É a
// ÚNICA fonte de verdade para largura de texto: layout e posicionamento de
// cursor devem passar por aqui.
func (t *Theme) MeasureString(s string) int {
	return render.MeasureText(t.Face, s)
}

// DrawMono desenha s com a fonte mono PADRÃO do tema (ver Mono/MonoFont).
func (t *Theme) DrawMono(dst *image.RGBA, s string, dot image.Point, c color.RGBA) {
	t.mono.Draw(dst, s, dot, c)
}

// MeasureMono devolve a largura de s em pixels na fonte mono padrão.
func (t *Theme) MeasureMono(s string) int {
	return t.mono.Measure(s)
}

// MonoLineHeight devolve a altura de linha da fonte mono padrão.
func (t *Theme) MonoLineHeight() int {
	return t.mono.LineHeight()
}

// MonoAscent devolve o ascent da fonte mono padrão.
func (t *Theme) MonoAscent() int {
	return t.mono.Ascent()
}

// MonoAdvance devolve a largura da CÉLULA da fonte mono padrão — todo
// glifo coberto avança exatamente isso, o que torna a conta coluna↔pixel
// aritmética pura (a base do CodeEditor).
func (t *Theme) MonoAdvance() int {
	return t.mono.Advance()
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
