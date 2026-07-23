package widget

// CursorShape identifica o formato do cursor do mouse que um widget deseja
// enquanto está sob o ponteiro. O App aplica o formato do widget mais
// profundo sob o cursor (e o mantém durante uma captura de mouse).
type CursorShape int

const (
	// CursorDefault é a seta padrão do sistema.
	CursorDefault CursorShape = iota
	// CursorText é o I-beam de campos de texto.
	CursorText
	// CursorHand é a mãozinha de elementos clicáveis/arrastáveis.
	CursorHand
)

// CursorShape devolve CursorDefault; widgets interativos sobrescrevem.
func (b *BaseWidget) CursorShape() CursorShape {
	return CursorDefault
}

// cursorShaper é como o App consulta o formato desejado pelo widget.
type cursorShaper interface {
	CursorShape() CursorShape
}

// CursorShapeOf devolve o formato de cursor desejado por w (CursorDefault
// para widgets que não declaram um).
func CursorShapeOf(w Widget) CursorShape {
	if p, ok := w.(cursorShaper); ok {
		return p.CursorShape()
	}
	return CursorDefault
}

// CursorShape do Input: I-beam, como todo campo de texto.
func (in *Input) CursorShape() CursorShape {
	return CursorText
}

// CursorShape do Button: mãozinha.
func (b *Button) CursorShape() CursorShape {
	return CursorHand
}

// CursorShape do Checkbox: mãozinha.
func (c *Checkbox) CursorShape() CursorShape {
	return CursorHand
}

// CursorShape do Slider: mãozinha (a alça é arrastável).
func (s *Slider) CursorShape() CursorShape {
	return CursorHand
}
