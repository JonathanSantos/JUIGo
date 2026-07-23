package theme

import "image/color"

// Classic constrói o tema claro com o visual clássico do JUIGo (anterior ao
// Theme.Radius): os mesmos valores de Default, sem cantos arredondados
// (Radius zero) e sem borda de botão. Com Radius zero as primitivas
// arredondadas degradam pixel a pixel para retângulos retos, então o
// resultado é idêntico ao antigo. Para um tema escuro clássico, zere os
// mesmos dois campos em um tema vindo de Dark:
//
//	t, _ := theme.Dark()
//	t.Radius = 0
//	t.ButtonBorder = color.RGBA{}
func Classic() (*Theme, error) {
	t, err := Default()
	if err != nil {
		return nil, err
	}
	t.Radius = 0
	t.ButtonBorder = color.RGBA{}
	return t, nil
}
