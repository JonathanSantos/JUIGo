package widget

import (
	"image"
	"testing"
	"time"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/state"
)

func TestContains(t *testing.T) {
	a := NewButton("a", nil)
	box := NewVBox(a)
	root := NewContainer(box)

	if !Contains(root, a) || !Contains(root, box) || !Contains(root, root) {
		t.Fatal("Contains deveria encontrar descendentes e a própria raiz")
	}
	if Contains(root, NewButton("x", nil)) || Contains(a, root) {
		t.Fatal("Contains não deveria encontrar widgets fora da árvore")
	}
}

func TestDropdownAbreNavegaSeleciona(t *testing.T) {
	var opened, closed Widget
	hooks.OpenOverlay = func(w any) { opened = w.(Widget) }
	hooks.CloseOverlay = func(w any) { closed = w.(Widget) }
	defer func() { hooks.OpenOverlay, hooks.CloseOverlay = nil, nil }()

	th := newTestTheme(t)
	sel := state.New("Média")
	var changed string
	d := NewDropdown("Baixa", "Média", "Alta").BindValue(sel)
	d.OnChange(func(v string) { changed = v })
	d.SetTheme(th)
	d.Layout(image.Rect(10, 10, 210, 46))

	// BindValue adota o valor do State.
	if d.Value() != "Média" || d.SelectedIndex() != 1 {
		t.Fatalf("bind inicial: Value=%q idx=%d, esperado Média/1", d.Value(), d.SelectedIndex())
	}

	// Enter abre o popup logo abaixo do gatilho, com a mesma largura.
	d.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	if opened == nil {
		t.Fatal("Enter deveria abrir o popup na overlay")
	}
	pb := opened.Bounds()
	if pb.Min.Y != 46+th.Px(2) || pb.Min.X != 10 || pb.Max.X != 210 {
		t.Fatalf("popup mal posicionado: %v", pb)
	}

	// Setas navegam a partir da seleção atual; Enter seleciona.
	opened.HandleEvent(event.KeyEvent{Key: event.KeyDown}) // Média → Alta
	opened.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	if d.Value() != "Alta" || sel.Get() != "Alta" || changed != "Alta" {
		t.Fatalf("seleção por teclado: Value=%q State=%q OnChange=%q", d.Value(), sel.Get(), changed)
	}
	if closed == nil {
		t.Fatal("selecionar deveria fechar o popup")
	}

	// Escape fecha sem selecionar.
	closed = nil
	d.HandleEvent(event.KeyEvent{Key: event.KeySpace})
	opened.HandleEvent(event.KeyEvent{Key: event.KeyEscape})
	if closed == nil || d.Value() != "Alta" {
		t.Fatalf("Escape deveria fechar sem mudar; Value=%q", d.Value())
	}

	// Clique no primeiro item seleciona por mouse.
	d.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	l := opened.(*dropdownList)
	pos := image.Pt(pb.Min.X+5, pb.Min.Y+th.BorderPx()+l.itemHeight()/2)
	opened.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: event.MouseButtonLeft})
	opened.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: pos, Button: event.MouseButtonLeft})
	if d.Value() != "Baixa" || sel.Get() != "Baixa" {
		t.Fatalf("seleção por clique: Value=%q State=%q", d.Value(), sel.Get())
	}

	// Perder o foco (clique fora/Tab, simulado) fecha o popup.
	closed = nil
	d.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	opened.HandleEvent(event.FocusEvent{Gained: false})
	if closed == nil {
		t.Fatal("perder o foco deveria fechar o popup")
	}

	// Set externo seleciona sem disparar OnChange.
	changed = ""
	sel.Set("Média")
	if d.SelectedIndex() != 1 || changed != "" {
		t.Fatalf("Set externo: idx=%d OnChange=%q", d.SelectedIndex(), changed)
	}

	// Fechado, setas trocam a seleção diretamente.
	d.HandleEvent(event.KeyEvent{Key: event.KeyDown})
	if d.Value() != "Alta" || sel.Get() != "Alta" {
		t.Fatalf("seta com popup fechado: Value=%q", d.Value())
	}
}

func TestInputCaretPisca(t *testing.T) {
	var pending []func()
	hooks.Schedule = func(d time.Duration, fn func()) func() {
		pending = append(pending, fn)
		return func() {}
	}
	defer func() { hooks.Schedule = nil }()

	th := newTestTheme(t)
	in := NewInput("")
	in.SetTheme(th)

	in.HandleEvent(event.FocusEvent{Gained: true})
	if !in.caretOn || len(pending) == 0 {
		t.Fatal("foco deveria deixar o caret visível e agendar a piscada")
	}

	// O tick apaga o caret e agenda o próximo.
	n := len(pending)
	pending[n-1]()
	if in.caretOn {
		t.Fatal("tick deveria apagar o caret")
	}
	if len(pending) == n {
		t.Fatal("tick deveria agendar a próxima piscada")
	}

	// Digitar reacende imediatamente (o caret não some enquanto digita).
	typeString(in, "a")
	if !in.caretOn {
		t.Fatal("edição deveria reacender o caret")
	}

	// Sem foco, caret fica visível e ticks antigos são inertes.
	in.HandleEvent(event.FocusEvent{Gained: false})
	if !in.caretOn {
		t.Fatal("sem foco, o caret fica no estado visível")
	}
}

func TestTooltipSetterEView(t *testing.T) {
	th := newTestTheme(t)

	b := Tooltip(NewButton("OK", nil), "Envia o formulário")
	var _ *Button = b // tipo concreto preservado
	if TooltipTextOf(b) != "Envia o formulário" {
		t.Fatalf("TooltipTextOf = %q", TooltipTextOf(b))
	}
	if TooltipTextOf(NewText("x")) != "" {
		t.Fatal("widget sem dica deveria devolver texto vazio")
	}

	v := NewTooltipView()
	v.SetTheme(th)
	v.SetText("dica")
	pref := v.PreferredSize()
	if pref.X <= 0 || pref.Y <= 0 {
		t.Fatalf("PreferredSize inválido: %v", pref)
	}
	v.Layout(image.Rectangle{Max: pref})
	buf := image.NewRGBA(image.Rect(0, 0, 200, 60))
	v.Draw(buf)
	if got := buf.RGBAAt(2, 2); got != th.TooltipBackground {
		t.Fatalf("fundo da dica = %v, esperado %v", got, th.TooltipBackground)
	}
}
