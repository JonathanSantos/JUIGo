package uitest

import (
	"fmt"

	"github.com/JonathanSantos/JUIGo/widget"
)

// Selector localiza widgets na árvore. Desc aparece nas mensagens de erro do
// harness ("nenhum widget encontrado para …").
type Selector struct {
	Desc  string
	Match func(w widget.Widget) bool
}

// Text seleciona pelo texto visível: conteúdo de um Text, rótulo de Button,
// Checkbox ou Radio, ou o valor atual de um Dropdown.
func Text(s string) Selector {
	return Selector{
		Desc: fmt.Sprintf("Text(%q)", s),
		Match: func(w widget.Widget) bool {
			switch t := w.(type) {
			case *widget.Text:
				return t.Text() == s
			case *widget.Button:
				return t.Label == s
			case *widget.Checkbox:
				return t.Label == s
			case *widget.Radio:
				return t.Label == s
			case *widget.Dropdown:
				return t.Value() == s
			}
			return false
		},
	}
}

// Placeholder seleciona Inputs e TextAreas pelo placeholder.
func Placeholder(s string) Selector {
	return Selector{
		Desc: fmt.Sprintf("Placeholder(%q)", s),
		Match: func(w widget.Widget) bool {
			switch t := w.(type) {
			case *widget.Input:
				return t.Placeholder == s
			case *widget.TextArea:
				return t.Placeholder == s
			}
			return false
		},
	}
}

// OfType seleciona o primeiro widget do tipo concreto T (em pré-ordem).
func OfType[T widget.Widget]() Selector {
	var zero T
	return Selector{
		Desc: fmt.Sprintf("OfType[%T]()", zero),
		Match: func(w widget.Widget) bool {
			_, ok := w.(T)
			return ok
		},
	}
}

// Where seleciona por um predicado arbitrário, com uma descrição para as
// mensagens de erro.
func Where(desc string, fn func(w widget.Widget) bool) Selector {
	return Selector{Desc: desc, Match: fn}
}
