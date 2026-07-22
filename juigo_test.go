package juigo

import (
	"image"
	"testing"
)

// newTestTheme falha o teste se o tema padrão (fonte embutida) não carregar.
func newTestTheme(t *testing.T) *Theme {
	t.Helper()
	th, err := DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}
	return th
}

func typeString(in *Input, s string) {
	for _, r := range s {
		in.HandleEvent(CharEvent{Rune: r})
	}
}

func TestInputRunesComAcentuacao(t *testing.T) {
	in := NewInput("")
	in.SetTheme(newTestTheme(t))
	typeString(in, "ação")

	if got := in.Text(); got != "ação" {
		t.Fatalf("Text() = %q, esperado %q", got, "ação")
	}
	if got := in.Cursor(); got != 4 {
		t.Fatalf("Cursor() = %d, esperado 4 (runes, não bytes)", got)
	}

	// Backspace apaga UMA rune (o "o"), não um byte.
	in.HandleEvent(KeyEvent{Key: KeyBackspace})
	if got := in.Text(); got != "açã" {
		t.Fatalf("após Backspace, Text() = %q, esperado %q", got, "açã")
	}

	// Home + Delete apagam a primeira rune.
	in.HandleEvent(KeyEvent{Key: KeyHome})
	in.HandleEvent(KeyEvent{Key: KeyDelete})
	if got := in.Text(); got != "çã" {
		t.Fatalf("após Home+Delete, Text() = %q, esperado %q", got, "çã")
	}

	// Inserção no meio: seta direita e digita.
	in.HandleEvent(KeyEvent{Key: KeyRight})
	typeString(in, "é")
	if got := in.Text(); got != "çéã" {
		t.Fatalf("após inserção no meio, Text() = %q, esperado %q", got, "çéã")
	}
	if got := in.Cursor(); got != 2 {
		t.Fatalf("Cursor() = %d, esperado 2", got)
	}

	// End move ao fim; setas não passam dos limites.
	in.HandleEvent(KeyEvent{Key: KeyEnd})
	if got := in.Cursor(); got != 3 {
		t.Fatalf("após End, Cursor() = %d, esperado 3", got)
	}
	if in.HandleEvent(KeyEvent{Key: KeyRight}) {
		t.Fatal("KeyRight no fim não deveria ser consumido")
	}
}

func TestInputOnChange(t *testing.T) {
	in := NewInput("")
	in.SetTheme(newTestTheme(t))
	var got []string
	in.OnChange = func(s string) { got = append(got, s) }

	typeString(in, "oi")
	in.HandleEvent(KeyEvent{Key: KeyBackspace})
	in.HandleEvent(KeyEvent{Key: KeyLeft}) // mover cursor não dispara OnChange

	want := []string{"o", "oi", "o"}
	if len(got) != len(want) {
		t.Fatalf("OnChange chamado %d vezes, esperado %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OnChange[%d] = %q, esperado %q", i, got[i], want[i])
		}
	}
}

func TestButtonClique(t *testing.T) {
	th := newTestTheme(t)
	fired := 0
	b := NewButton("OK", func() { fired++ })
	b.SetTheme(th)
	b.Layout(image.Rect(0, 0, 100, 40))
	inside := image.Pt(50, 20)

	// Down dentro + Up dentro dispara.
	b.HandleEvent(MouseEvent{Kind: MouseEnter, Pos: inside})
	b.HandleEvent(MouseEvent{Kind: MouseDown, Pos: inside, Button: MouseButtonLeft})
	if b.State() != ButtonStatePressed {
		t.Fatalf("após MouseDown, State() = %v, esperado pressed", b.State())
	}
	b.HandleEvent(MouseEvent{Kind: MouseUp, Pos: inside, Button: MouseButtonLeft})
	if fired != 1 {
		t.Fatalf("OnClick disparado %d vezes, esperado 1", fired)
	}
	if b.State() != ButtonStateHover {
		t.Fatalf("após clique, State() = %v, esperado hover", b.State())
	}

	// Down dentro + Leave cancela sem disparar; Up posterior não dispara.
	b.HandleEvent(MouseEvent{Kind: MouseDown, Pos: inside, Button: MouseButtonLeft})
	b.HandleEvent(MouseEvent{Kind: MouseLeave, Pos: image.Pt(200, 20)})
	if b.State() != ButtonStateNormal {
		t.Fatalf("após Leave pressionado, State() = %v, esperado normal", b.State())
	}
	b.HandleEvent(MouseEvent{Kind: MouseUp, Pos: image.Pt(200, 20), Button: MouseButtonLeft})
	if fired != 1 {
		t.Fatalf("OnClick disparado %d vezes após cancelamento, esperado 1", fired)
	}

	// Enter e Espaço disparam via teclado.
	b.HandleEvent(KeyEvent{Key: KeyEnter})
	b.HandleEvent(KeyEvent{Key: KeySpace})
	if fired != 3 {
		t.Fatalf("OnClick disparado %d vezes, esperado 3", fired)
	}
}

func TestRoteamentoGeometriaEProfundidade(t *testing.T) {
	th := newTestTheme(t)

	inner := NewButton("fundo", nil)
	inner.Layout(image.Rect(10, 10, 60, 40))
	box := NewContainer(inner)
	box.Layout(image.Rect(0, 0, 100, 100))
	root := NewContainer(box)
	root.Layout(image.Rect(0, 0, 200, 200))
	propagateTheme(root, th)

	// Dentro do botão: o mais profundo consome.
	if !dispatchMouse(root, MouseEvent{Kind: MouseDown, Pos: image.Pt(20, 20), Button: MouseButtonLeft}) {
		t.Fatal("evento dentro do botão deveria ser consumido")
	}
	if inner.State() != ButtonStatePressed {
		t.Fatal("botão interno deveria estar pressed")
	}

	// Fora do botão, dentro dos containers: ninguém consome (propaga e morre).
	if dispatchMouse(root, MouseEvent{Kind: MouseDown, Pos: image.Pt(90, 90), Button: MouseButtonLeft}) {
		t.Fatal("evento fora do botão não deveria ser consumido")
	}

	// focusableAt encontra o botão pelo ponto; fora dele, nil.
	if got := focusableAt(root, image.Pt(20, 20)); got != Widget(inner) {
		t.Fatalf("focusableAt = %T, esperado o botão interno", got)
	}
	if got := focusableAt(root, image.Pt(90, 90)); got != nil {
		t.Fatalf("focusableAt fora do botão = %T, esperado nil", got)
	}
}

func TestVBoxHBoxLayout(t *testing.T) {
	th := newTestTheme(t)

	a := NewButton("A", nil)
	b := NewButton("B", nil)

	v := NewVBox(a, b).Gap(10).Pad(4)
	propagateTheme(v, th)
	v.Layout(image.Rect(0, 0, 200, 300))

	ah := a.PreferredSize().Y
	wantA := image.Rect(4, 4, 196, 4+ah)
	if a.Bounds() != wantA {
		t.Fatalf("VBox: bounds de A = %v, esperado %v", a.Bounds(), wantA)
	}
	wantBTop := 4 + ah + 10
	if b.Bounds().Min.Y != wantBTop {
		t.Fatalf("VBox: topo de B = %d, esperado %d", b.Bounds().Min.Y, wantBTop)
	}

	c := NewButton("C", nil)
	d := NewButton("D", nil)
	hb := NewHBox(c, d).Gap(6).Pad(2)
	propagateTheme(hb, th)
	hb.Layout(image.Rect(0, 0, 300, 100))

	cw := c.PreferredSize().X
	wantC := image.Rect(2, 2, 2+cw, 98)
	if c.Bounds() != wantC {
		t.Fatalf("HBox: bounds de C = %v, esperado %v", c.Bounds(), wantC)
	}
	if d.Bounds().Min.X != 2+cw+6 {
		t.Fatalf("HBox: esquerda de D = %d, esperado %d", d.Bounds().Min.X, 2+cw+6)
	}

	// PreferredSize agrega: soma no eixo principal, máximo no transversal.
	pv := v.PreferredSize()
	wantH := a.PreferredSize().Y + b.PreferredSize().Y + 10 + 8
	if pv.Y != wantH {
		t.Fatalf("VBox.PreferredSize().Y = %d, esperado %d", pv.Y, wantH)
	}
}

func TestEventBusSincrono(t *testing.T) {
	bus := NewEventBus()
	order := []string{}
	bus.Subscribe("t", func(any) { order = append(order, "a") })
	bus.Subscribe("t", func(any) { order = append(order, "b") })
	bus.Publish("t", nil)
	// Síncrono: os handlers já executaram quando Publish retorna.
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("Publish síncrono fora de ordem: %v", order)
	}
}
