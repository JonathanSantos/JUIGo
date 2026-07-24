package theme

import "image/color"

// Claude constrói o tema claro do design system "papel e tinta" do JUIGo,
// inspirado na linguagem visual pública do Claude (Anthropic): fundo de
// papel, texto em tinta quase-preta quente, terracota reservada a ações e
// estados ativos, neutros SEMPRE quentes (nada de cinza azulado) e cantos
// generosos (Radius 10). As cores são uma homenagem aproximada à identidade
// pública — o tema não usa marcas nem se propõe a representar a Anthropic.
//
// As decisões de uso dos tokens estão documentadas em docs/DESIGN.md; o par
// escuro é ClaudeDark. Troque em runtime com App.SetTheme.
func Claude() (*Theme, error) {
	t, err := Default()
	if err != nil {
		return nil, err
	}

	// Papel e tinta: o fundo é papel (#FAF9F5), superfícies de repouso são
	// aveia (#F0EEE6) e o texto é tinta quente (#141413).
	t.Background = color.RGBA{R: 0xFA, G: 0xF9, B: 0xF5, A: 0xFF}
	t.Text = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}
	t.Placeholder = color.RGBA{R: 0x92, G: 0x8B, B: 0x7F, A: 0xFF}

	// Terracota (#D97757) é A cor de ação — botões, foco de campo, marca de
	// seleção. Hover clareia, pressionado aprofunda ao "book cloth".
	t.ButtonNormal = color.RGBA{R: 0xD9, G: 0x77, B: 0x57, A: 0xFF}
	t.ButtonHover = color.RGBA{R: 0xE0, G: 0x8A, B: 0x6C, A: 0xFF}
	t.ButtonPressed = color.RGBA{R: 0xBD, G: 0x5F, B: 0x3D, A: 0xFF}
	t.ButtonText = color.RGBA{R: 0xFA, G: 0xF9, B: 0xF5, A: 0xFF}
	t.ButtonBorder = color.RGBA{R: 0xCC, G: 0x78, B: 0x5C, A: 0xFF}

	t.InputBackground = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	t.InputBorder = color.RGBA{R: 0xD6, G: 0xCF, B: 0xC2, A: 0xFF}
	t.InputBorderFocused = color.RGBA{R: 0xD9, G: 0x77, B: 0x57, A: 0xFF}
	t.Cursor = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}
	// Seleção de texto em manilla: papel marcado, não azul de sistema.
	t.Selection = color.RGBA{R: 0xEB, G: 0xDB, B: 0xBC, A: 0xFF}
	t.FocusOutline = color.RGBA{R: 0xC1, G: 0x5F, B: 0x3C, A: 0xFF}
	t.Accent = color.RGBA{R: 0xD9, G: 0x77, B: 0x57, A: 0xFF}
	t.HoverBackground = color.RGBA{R: 0xF0, G: 0xEE, B: 0xE6, A: 0xFF}

	// Tooltip é tinta sobre papel invertidos: caixa escura, texto claro.
	t.TooltipBackground = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}
	t.TooltipText = color.RGBA{R: 0xFA, G: 0xF9, B: 0xF5, A: 0xFF}

	// Perigo em barro queimado — avermelhado, mas da mesma família térrea.
	t.Danger = color.RGBA{R: 0xBF, G: 0x4D, B: 0x43, A: 0xFF}

	// Sintaxe na família da marca: terracota funda, oliva, azul-névoa e
	// kraft — códigos legíveis sem sair do mundo quente.
	t.Syntax = SyntaxPalette{
		Keyword: color.RGBA{R: 0xB0, G: 0x51, B: 0x2F, A: 0xFF},
		String:  color.RGBA{R: 0x66, G: 0x7B, B: 0x4E, A: 0xFF},
		Number:  color.RGBA{R: 0x4E, G: 0x73, B: 0x96, A: 0xFF},
		Comment: color.RGBA{R: 0x8C, G: 0x85, B: 0x7A, A: 0xFF},
		Builtin: color.RGBA{R: 0x9A, G: 0x6B, B: 0x2F, A: 0xFF},
	}
	t.CurrentLine = color.RGBA{R: 0xF2, G: 0xEF, B: 0xE7, A: 0xFF}

	t.DisabledWash = color.RGBA{R: 0xFA, G: 0xF9, B: 0xF5, A: 0x99}

	// Formas generosas e respiro maior que o Default: raio 10 e passo 10.
	t.Radius = 10
	t.Padding = 10
	t.Spacing = 10

	return t, nil
}

// ClaudeDark constrói o par escuro do design system "papel e tinta" (ver
// Claude): superfícies de grafite quente, texto de papel, a MESMA terracota
// de ação e neutros térreos reequilibrados para contraste. Métricas
// idênticas às do Claude claro — os dois alternam em runtime sem a
// interface pular.
func ClaudeDark() (*Theme, error) {
	t, err := Claude()
	if err != nil {
		return nil, err
	}

	t.Background = color.RGBA{R: 0x26, G: 0x26, B: 0x24, A: 0xFF}
	t.Text = color.RGBA{R: 0xED, G: 0xEB, B: 0xE3, A: 0xFF}
	t.Placeholder = color.RGBA{R: 0xA0, G: 0x9A, B: 0x8D, A: 0xFF}

	t.ButtonPressed = color.RGBA{R: 0xB0, G: 0x53, B: 0x2E, A: 0xFF}
	t.ButtonBorder = color.RGBA{R: 0xB5, G: 0x64, B: 0x3F, A: 0xFF}

	t.InputBackground = color.RGBA{R: 0x30, G: 0x30, B: 0x2E, A: 0xFF}
	t.InputBorder = color.RGBA{R: 0x4C, G: 0x4A, B: 0x44, A: 0xFF}
	t.Cursor = color.RGBA{R: 0xED, G: 0xEB, B: 0xE3, A: 0xFF}
	// Seleção em kraft profundo: quente e legível sob texto de papel.
	t.Selection = color.RGBA{R: 0x57, G: 0x44, B: 0x34, A: 0xFF}
	// Foco clareado para ler sobre grafite.
	t.FocusOutline = color.RGBA{R: 0xE3, G: 0x9A, B: 0x7D, A: 0xFF}
	t.HoverBackground = color.RGBA{R: 0x38, G: 0x37, B: 0x33, A: 0xFF}

	// Tooltip invertido de volta: papel sobre a interface escura.
	t.TooltipBackground = color.RGBA{R: 0xF0, G: 0xEE, B: 0xE6, A: 0xFF}
	t.TooltipText = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}

	t.Danger = color.RGBA{R: 0xE5, G: 0x75, B: 0x6A, A: 0xFF}

	t.Syntax = SyntaxPalette{
		Keyword: color.RGBA{R: 0xE5, G: 0x87, B: 0x6A, A: 0xFF},
		String:  color.RGBA{R: 0xA9, G: 0xBC, B: 0x8F, A: 0xFF},
		Number:  color.RGBA{R: 0x8F, G: 0xB4, B: 0xD9, A: 0xFF},
		Comment: color.RGBA{R: 0x8F, G: 0x8A, B: 0x80, A: 0xFF},
		Builtin: color.RGBA{R: 0xD4, G: 0xA2, B: 0x7F, A: 0xFF},
	}
	t.CurrentLine = color.RGBA{R: 0x2E, G: 0x2E, B: 0x2B, A: 0xFF}

	t.Backdrop = color.RGBA{A: 0x99}
	t.DisabledWash = color.RGBA{R: 0x26, G: 0x26, B: 0x24, A: 0x99}

	return t, nil
}
