package theme

import "image/color"

// Dark constrói o tema escuro do JUIGo na escala 1: mesma fonte, métricas e
// espaçamentos do tema padrão, com a paleta invertida — superfícies escuras,
// texto claro, o mesmo azul de destaque e tooltip/seleção reequilibrados
// para contraste. Troque em runtime com App.SetTheme.
func Dark() (*Theme, error) {
	t, err := Default()
	if err != nil {
		return nil, err
	}

	t.Background = color.RGBA{R: 0x1E, G: 0x21, B: 0x26, A: 0xFF}
	t.Text = color.RGBA{R: 0xE8, G: 0xEA, B: 0xED, A: 0xFF}
	t.Placeholder = color.RGBA{R: 0x6E, G: 0x76, B: 0x81, A: 0xFF}

	t.InputBackground = color.RGBA{R: 0x26, G: 0x2A, B: 0x31, A: 0xFF}
	t.InputBorder = color.RGBA{R: 0x3D, G: 0x43, B: 0x4D, A: 0xFF}
	t.Cursor = color.RGBA{R: 0xE8, G: 0xEA, B: 0xED, A: 0xFF}
	t.Selection = color.RGBA{R: 0x1E, G: 0x3A, B: 0x5F, A: 0xFF}

	// Foco em azul mais claro: lê melhor sobre superfícies escuras.
	t.FocusOutline = color.RGBA{R: 0x60, G: 0xA5, B: 0xFA, A: 0xFF}
	t.HoverBackground = color.RGBA{R: 0x2E, G: 0x34, B: 0x3D, A: 0xFF}

	// Tooltip invertido (caixa clara sobre interface escura).
	t.TooltipBackground = color.RGBA{R: 0xE8, G: 0xEA, B: 0xED, A: 0xFF}
	t.TooltipText = color.RGBA{R: 0x1F, G: 0x23, B: 0x28, A: 0xFF}

	// Vermelho mais claro para contraste no escuro.
	t.Danger = color.RGBA{R: 0xF8, G: 0x71, B: 0x71, A: 0xFF}

	// Paleta de sintaxe reequilibrada para superfícies escuras.
	t.Syntax = SyntaxPalette{
		Keyword: color.RGBA{R: 0xFF, G: 0x7B, B: 0x72, A: 0xFF},
		String:  color.RGBA{R: 0xA5, G: 0xD6, B: 0xFF, A: 0xFF},
		Number:  color.RGBA{R: 0x79, G: 0xC0, B: 0xFF, A: 0xFF},
		Comment: color.RGBA{R: 0x8B, G: 0x94, B: 0x9E, A: 0xFF},
		Builtin: color.RGBA{R: 0xFF, G: 0xA6, B: 0x57, A: 0xFF},
	}

	t.Backdrop = color.RGBA{A: 0x99}
	t.DisabledWash = color.RGBA{R: 0x1E, G: 0x21, B: 0x26, A: 0x99}

	return t, nil
}
