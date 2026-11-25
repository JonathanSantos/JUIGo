package theme

import (
	"fmt"
	"image/color"
	"math"
)

// Palette são as SEMENTES de um tema: meia dúzia de cores das quais
// New/ApplyPalette derivam todos os tokens de cor do Theme — fundos,
// bordas, estados de botão, seleção, foco, mensagens e a paleta de sintaxe
// — já com o contraste WCAG garantido. Criar um tema é escolher sementes:
//
//	th, err := theme.New(theme.Palette{
//		Primary: color.RGBA{R: 0x2B, G: 0x59, B: 0xD9, A: 0xFF}, // cobalto
//	})
//
// Campos com alfa zero usam o padrão indicado em cada um. Os mesmos
// valores com Dark ligado produzem o par escuro, com as mesmas métricas —
// alternar claro/escuro em runtime não move um pixel de layout.
type Palette struct {
	// Dark gera o modo escuro: fundos de carvão e texto claro.
	Dark bool
	// Neutral dá o MATIZ dos neutros (fundos, bordas, texto). Só a
	// tonalidade importa — a luminosidade de cada token vem da escala do
	// tema e o croma é limitado a um toque. Zero = cinza puro.
	Neutral color.RGBA
	// Primary é a cor de AÇÃO: botão primário, marca do checkbox e do
	// radio, trilho ativo do slider, aba ativa, gráficos. Uma Primary sem
	// croma (preto, grafite) vira tinta: no modo escuro ela se inverte para
	// a cor do texto. Zero = tinta.
	Primary color.RGBA
	// Signal é a cor de SINAL: anel de foco e, sem Highlight, o matiz da
	// seleção. Zero = Primary.
	Signal color.RGBA
	// Highlight dá o MATIZ da seleção (texto, linhas, dias) — o marca-texto;
	// a luminosidade vem do modo. Útil quando o tom que funciona como anel
	// não funciona como fundo claro (o âmbar vira pêssego; o amarelo, não).
	// Zero = Signal.
	Highlight color.RGBA
	// Danger, Success e Warning são as cores de estado (erro, sucesso,
	// aviso); servem como texto, então são ajustadas a 4.5:1 sobre o fundo.
	// Zero = vermelho, verde e âmbar padrão.
	Danger  color.RGBA
	Success color.RGBA
	Warning color.RGBA
}

// Nanquim devolve as sementes da identidade do JUIGo: tinta preta como
// ação, neutros de cinza puro e um único sinal âmbar no foco e na seleção.
// Ligue Dark para o par escuro (ver Default e Dark).
func Nanquim() Palette {
	return Palette{
		Primary:   rgb(0x111111),
		Signal:    rgb(0xC98A00),
		Highlight: rgb(0xFFE7A0),
	}
}

// rgb converte 0xRRGGBB numa cor opaca.
func rgb(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}

// Padrões das sementes de estado.
var (
	defaultDanger  = rgb(0xD92D20)
	defaultSuccess = rgb(0x1F8A4C)
	defaultWarning = rgb(0xC77700)
)

// ramp é a escala de luminosidades (OKLCh L) dos neutros num modo.
type ramp struct {
	background, surface, hover, header, track, border, inputBorder float64
	placeholder, text, currentLine                                 float64
}

var (
	lightRamp = ramp{
		background: 0.985, surface: 1, hover: 0.955, header: 0.968,
		track: 0.905, border: 0.917, inputBorder: 0.868,
		placeholder: 0.555, text: 0.18, currentLine: 0.97,
	}
	darkRamp = ramp{
		background: 0.17, surface: 0.205, hover: 0.245, header: 0.225,
		track: 0.31, border: 0.275, inputBorder: 0.315,
		placeholder: 0.665, text: 0.945, currentLine: 0.235,
	}
)

// achromatic informa se a cor é praticamente neutra (sem croma visível).
func achromatic(c color.RGBA) bool { return toOKLCh(c).C < 0.03 }

// orDefault devolve c, ou def quando c tem alfa zero (semente omitida).
func orDefault(c, def color.RGBA) color.RGBA {
	if c.A == 0 {
		return def
	}
	return c
}

// shift devolve c com a luminosidade deslocada de d (OKLCh).
func shift(c color.RGBA, d float64) color.RGBA {
	o := toOKLCh(c)
	return o.withL(o.L + d).rgb()
}

// ApplyPalette deriva TODOS os tokens de cor do tema a partir das sementes
// (ver Palette), sem tocar em fontes, métricas ou durações — dá para
// re-colorir um tema já ajustado. É função pura das sementes: as mesmas
// sementes produzem sempre as mesmas cores.
func (t *Theme) ApplyPalette(p Palette) {
	r := lightRamp
	if p.Dark {
		r = darkRamp
	}
	// Neutros: o matiz da semente com no máximo um toque de croma.
	n := toOKLCh(orDefault(p.Neutral, rgb(0x808080)))
	n.C = math.Min(n.C, 0.012)
	neutral := func(l float64) color.RGBA { return n.withL(l).rgb() }

	t.Background = neutral(r.background)
	t.Surface = neutral(r.surface)
	t.SurfaceBorder = neutral(r.border)
	t.Text = neutral(r.text)
	t.Cursor = t.Text
	t.InputBackground = t.Surface
	t.InputBorder = neutral(r.inputBorder)
	t.HoverBackground = neutral(r.hover)
	t.HeaderBackground = neutral(r.header)
	t.Track = neutral(r.track)
	t.CurrentLine = neutral(r.currentLine)
	t.Placeholder = ensureContrast(neutral(r.placeholder), t.InputBackground, 4.5)

	// Ação. Tinta (sem croma) se inverte no escuro; cor com croma é
	// ajustada para ser percebida (3:1) sobre o fundo.
	primary := orDefault(p.Primary, rgb(0x111111))
	switch {
	case achromatic(primary) && p.Dark == (toOKLCh(primary).L < 0.5):
		primary = t.Text
	default:
		primary = ensureContrast(primary, t.Background, 3)
	}
	// Rótulo: o neutro claro ou escuro que mais contrasta com a ação; se
	// nem o melhor alcança 4.5:1 (ações de luminância média), a ação se
	// afasta do rótulo até alcançar.
	light, dark := neutral(lightRamp.surface), neutral(lightRamp.text)
	label := dark
	if Contrast(light, primary) >= Contrast(dark, primary) {
		label = light
	}
	primary = ensureContrast(primary, label, 4.5)
	t.ButtonText = label
	t.AccentText = label
	t.Accent = primary
	t.ButtonNormal = primary
	// Hover se afasta do fundo do botão em direção ao meio; pressionado vai
	// para o extremo. Botão escuro clareia no hover e escurece ao apertar.
	if toOKLCh(primary).L < 0.6 {
		t.ButtonHover, t.ButtonPressed = shift(primary, 0.08), shift(primary, -0.06)
	} else {
		t.ButtonHover, t.ButtonPressed = shift(primary, 0.04), shift(primary, -0.08)
	}
	t.ButtonBorder = t.ButtonPressed
	t.InputBorderFocused = primary

	// Sinal: foco percebido (3:1) e seleção como marca-texto legível.
	signal := orDefault(p.Signal, orDefault(p.Primary, primary))
	// O anel fica numa faixa de luminosidade por modo — fundo no claro, aceso
	// no escuro — para ler como sinal e não como mancha; depois, 3:1.
	f := toOKLCh(signal)
	if p.Dark {
		f = f.withL(math.Max(f.L, 0.8))
	} else {
		f = f.withL(math.Min(f.L, 0.62))
	}
	t.FocusOutline = ensureContrast(f.rgb(), t.Background, 3)
	s := toOKLCh(orDefault(p.Highlight, signal))
	s.C = math.Min(s.C, 0.11)
	if p.Dark {
		t.Selection = ensureContrast(s.withL(0.4).rgb(), t.Text, 4.5)
	} else {
		t.Selection = ensureContrast(s.withL(0.93).rgb(), t.Text, 4.5)
	}

	// Estados: usados como texto, então 4.5:1 sobre o fundo e a superfície.
	state := func(c color.RGBA) color.RGBA {
		return ensureContrast(ensureContrast(c, t.Background, 4.5), t.Surface, 4.5)
	}
	t.Danger = state(orDefault(p.Danger, defaultDanger))
	t.Success = state(orDefault(p.Success, defaultSuccess))
	t.Warning = state(orDefault(p.Warning, defaultWarning))

	// Dica: o modo invertido (tinta sobre papel no escuro e vice-versa).
	t.TooltipBackground, t.TooltipText = t.Text, t.Background

	// Sintaxe: a palavra-chave é a ação; strings no matiz do sinal; números
	// e embutidos em matizes fixos — tudo a 4.5:1 sobre o fundo do editor.
	code := func(c color.RGBA) color.RGBA { return ensureContrast(c, t.InputBackground, 4.5) }
	hue := func(deg, c float64) color.RGBA {
		l := 0.55
		if p.Dark {
			l = 0.78
		}
		return oklch{L: l, C: c, H: deg * math.Pi / 180}.rgb()
	}
	str := s.withC(math.Max(s.C, 0.1))
	if p.Dark {
		str = str.withL(0.8)
	} else {
		str = str.withL(0.5)
	}
	// Ação sem croma (tinta) não serve de palavra-chave — ficaria igual aos
	// identificadores —, então ela ganha um matiz próprio.
	keyword := primary
	if achromatic(primary) {
		keyword = hue(262, 0.15)
	}
	builtin := hue(262, 0.13)
	if achromatic(primary) {
		builtin = hue(190, 0.1)
	}
	t.Syntax = SyntaxPalette{
		Keyword: code(keyword),
		String:  code(str.rgb()),
		Number:  code(hue(45, 0.14)),
		Comment: t.Placeholder,
		Builtin: code(builtin),
	}

	// Translúcidos: lavagem do fundo e pano do modal.
	t.DisabledWash = withAlpha(t.Background, 0x99)
	t.Backdrop = color.RGBA{A: 0x66}
	if p.Dark {
		t.Backdrop = color.RGBA{A: 0x99}
	}
}

// New constrói um tema a partir das sementes (ver Palette), com as
// métricas, fontes e durações do tema padrão, na escala 1. É o caminho
// curto para um tema próprio:
//
//	marca := theme.Palette{Primary: color.RGBA{R: 0x2D, G: 0x6A, B: 0x4F, A: 0xFF}}
//	claro, _ := theme.New(marca)
//	marca.Dark = true
//	escuro, _ := theme.New(marca)
func New(p Palette) (*Theme, error) {
	t, err := base()
	if err != nil {
		return nil, err
	}
	t.ApplyPalette(p)
	return t, nil
}

// ContrastIssue é um par de tokens com contraste abaixo do mínimo (ver
// Theme.Audit).
type ContrastIssue struct {
	// Pair nomeia os tokens, "frente/fundo".
	Pair string
	// Ratio é o contraste medido e Min o exigido.
	Ratio, Min float64
}

// String descreve o problema em uma linha.
func (c ContrastIssue) String() string {
	return fmt.Sprintf("%s: %.2f:1 (mínimo %.1f:1)", c.Pair, c.Ratio, c.Min)
}

// Audit confere os pares de cor que a interface desenha um sobre o outro
// contra a régua WCAG (4.5:1 para texto, 3:1 para foco e marcas) e
// devolve os que ficaram abaixo — vazio quando o tema passa. Útil em teste
// de tema próprio:
//
//	if issues := th.Audit(); len(issues) > 0 { t.Fatal(issues) }
func (t *Theme) Audit() []ContrastIssue {
	pairs := []struct {
		name string
		a, b color.RGBA
		min  float64
	}{
		{"Text/Background", t.Text, t.Background, 4.5},
		{"Text/Surface", t.Text, t.Surface, 4.5},
		{"Text/InputBackground", t.Text, t.InputBackground, 4.5},
		{"Text/Selection", t.Text, t.Selection, 4.5},
		{"ButtonText/ButtonNormal", t.ButtonText, t.ButtonNormal, 4.5},
		{"Placeholder/InputBackground", t.Placeholder, t.InputBackground, 4.5},
		{"Danger/Background", t.Danger, t.Background, 4.5},
		{"Success/Background", t.Success, t.Background, 4.5},
		{"Warning/Background", t.Warning, t.Background, 4.5},
		{"TooltipText/TooltipBackground", t.TooltipText, t.TooltipBackground, 4.5},
		{"FocusOutline/Background", t.FocusOutline, t.Background, 3},
		{"Accent/Background", t.Accent, t.Background, 3},
		{"AccentText/Accent", t.AccentText, t.Accent, 3},
	}
	var out []ContrastIssue
	for _, p := range pairs {
		if r := Contrast(p.a, p.b); r < p.min {
			out = append(out, ContrastIssue{Pair: p.name, Ratio: r, Min: p.min})
		}
	}
	return out
}
