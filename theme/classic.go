package theme

import (
	"image/color"
)

// Classic constrói o tema claro com o visual clássico do JUIGo, anterior à
// identidade Nanquim: azul de sistema, neutros azulados, cantos retos
// (Radius zero), botão sem borda, títulos em Go Bold e as métricas antigas
// (Padding 8) e corpo em Go Regular. Fica para quem já tinha telas
// desenhadas sobre ele; para algo novo, prefira Default ou um tema próprio
// com New. O par escuro é ClassicDark.
func Classic() (*Theme, error) {
	t, err := base()
	if err != nil {
		return nil, err
	}
	classicLight(t)
	return t, finishClassic(t)
}

// ClassicDark é o par escuro do Classic: o antigo tema Dark (superfícies de
// grafite azulado, o mesmo azul de destaque) com os cantos retos.
func ClassicDark() (*Theme, error) {
	t, err := base()
	if err != nil {
		return nil, err
	}
	classicLight(t)
	classicDark(t)
	return t, finishClassic(t)
}

// finishClassic aplica as métricas e a tipografia clássicas.
func finishClassic(t *Theme) error {
	t.Radius = 0
	t.Padding = 8
	t.Spacing = 8
	t.ButtonBorder = color.RGBA{}
	body, err := GoRegular()
	if err != nil {
		return err
	}
	if err := t.UseBodyFont(body); err != nil {
		return err
	}
	bold, err := GoBold()
	if err != nil {
		return err
	}
	return t.UseDisplayFont(bold)
}

// classicLight pinta a paleta clássica clara.
func classicLight(t *Theme) {
	t.Background = rgb(0xF2F3F5)
	t.Text = rgb(0x1F2328)
	t.Placeholder = rgb(0x9AA0A8)
	t.ButtonNormal = rgb(0x3B82F6)
	t.ButtonHover = rgb(0x60A5FA)
	t.ButtonPressed = rgb(0x1D4ED8)
	t.ButtonText = rgb(0xFFFFFF)
	t.InputBackground = rgb(0xFFFFFF)
	t.InputBorder = rgb(0xB4BAC2)
	t.InputBorderFocused = rgb(0x3B82F6)
	t.Cursor = rgb(0x1F2328)
	t.Selection = rgb(0xBFDBFE)
	t.FocusOutline = rgb(0x1D4ED8)
	t.Accent = rgb(0x3B82F6)
	t.AccentText = rgb(0xFFFFFF)
	t.HoverBackground = rgb(0xE4ECF9)
	t.HeaderBackground = t.HoverBackground
	t.Track = t.InputBorder
	t.TooltipBackground = rgb(0x2E333B)
	t.TooltipText = rgb(0xF5F7FA)
	t.Danger = rgb(0xDC2626)
	t.Success = rgb(0x15803D)
	t.Warning = rgb(0xB45309)
	t.Syntax = SyntaxPalette{
		Keyword: rgb(0xCF222E),
		String:  rgb(0x0A3069),
		Number:  rgb(0x0550AE),
		Comment: rgb(0x6E7781),
		Builtin: rgb(0x953800),
	}
	t.CurrentLine = rgb(0xF2F5FA)
	t.Surface = rgb(0xFFFFFF)
	t.SurfaceBorder = rgb(0xDDE1E6)
	t.Backdrop = color.RGBA{A: 0x66}
	t.DisabledWash = withAlpha(rgb(0xF2F3F5), 0x99)
}

// classicDark sobrescreve a paleta clássica com a variante escura.
func classicDark(t *Theme) {
	t.Background = rgb(0x1E2126)
	t.Text = rgb(0xE8EAED)
	t.Placeholder = rgb(0x6E7681)
	t.InputBackground = rgb(0x262A31)
	t.InputBorder = rgb(0x3D434D)
	t.Track = t.InputBorder
	t.Cursor = rgb(0xE8EAED)
	t.Selection = rgb(0x1E3A5F)
	t.FocusOutline = rgb(0x60A5FA)
	t.HoverBackground = rgb(0x2E343D)
	t.HeaderBackground = t.HoverBackground
	t.TooltipBackground = rgb(0xE8EAED)
	t.TooltipText = rgb(0x1F2328)
	t.Danger = rgb(0xF87171)
	t.Success = rgb(0x4ADE80)
	t.Warning = rgb(0xFBBF24)
	t.Syntax = SyntaxPalette{
		Keyword: rgb(0xFF7B72),
		String:  rgb(0xA5D6FF),
		Number:  rgb(0x79C0FF),
		Comment: rgb(0x8B949E),
		Builtin: rgb(0xFFA657),
	}
	t.CurrentLine = rgb(0x2C323C)
	t.Backdrop = color.RGBA{A: 0x99}
	t.DisabledWash = withAlpha(rgb(0x1E2126), 0x99)
	t.Surface = rgb(0x262A31)
	t.SurfaceBorder = rgb(0x3D434D)
}
