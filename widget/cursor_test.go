package widget

import "testing"

func TestCursorShapePorWidget(t *testing.T) {
	cases := []struct {
		nome string
		w    Widget
		want CursorShape
	}{
		{"Input", NewInput(""), CursorText},
		{"Button", NewButton("x", nil), CursorHand},
		{"Checkbox", NewCheckbox("x"), CursorHand},
		{"Slider", NewSlider(0, 1), CursorHand},
		{"Text", NewText("x"), CursorDefault},
		{"VBox", NewVBox(), CursorDefault},
		{"Spacer", NewSpacer(), CursorDefault},
	}
	for _, c := range cases {
		if got := CursorShapeOf(c.w); got != c.want {
			t.Fatalf("%s: CursorShapeOf = %v, esperado %v", c.nome, got, c.want)
		}
	}
}
