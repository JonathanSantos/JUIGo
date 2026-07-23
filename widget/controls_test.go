package widget

import (
	"image"
	"testing"

	"juigo/event"
	"juigo/state"
)

func TestCheckboxToggleEBinding(t *testing.T) {
	th := newTestTheme(t)
	ligado := state.New(false)
	var changes []bool

	c := NewCheckbox("Notificações").BindChecked(ligado)
	c.SetTheme(th)
	c.OnChange(func(v bool) { changes = append(changes, v) })
	c.Layout(image.Rect(0, 0, 150, 24))
	inside := image.Pt(10, 12)

	// Clique completo alterna e propaga ao State.
	c.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: inside, Button: event.MouseButtonLeft})
	c.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: inside, Button: event.MouseButtonLeft})
	if !c.Checked() || !ligado.Get() {
		t.Fatalf("após clique: Checked=%v, State=%v; esperado true/true", c.Checked(), ligado.Get())
	}

	// event.MouseLeave pressionado cancela sem alternar.
	c.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: inside, Button: event.MouseButtonLeft})
	c.HandleEvent(event.MouseEvent{Kind: event.MouseLeave, Pos: image.Pt(300, 12)})
	c.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: image.Pt(300, 12), Button: event.MouseButtonLeft})
	if !c.Checked() {
		t.Fatal("cancelamento não deveria ter alternado o valor")
	}

	// Espaço alterna quando focado; Set externo atualiza a caixa.
	c.HandleEvent(event.KeyEvent{Key: event.KeySpace})
	if c.Checked() || ligado.Get() {
		t.Fatal("Espaço deveria ter desmarcado e propagado ao State")
	}
	ligado.Set(true)
	if !c.Checked() {
		t.Fatal("Set externo no State deveria marcar a caixa")
	}

	// OnChange só nas ações do usuário (clique e Espaço), não no Set externo.
	if len(changes) != 2 || changes[0] != true || changes[1] != false {
		t.Fatalf("OnChange = %v, esperado [true false]", changes)
	}
}

func TestSliderMouseTecladoEBinding(t *testing.T) {
	th := newTestTheme(t)
	vol := state.New(0.0)

	s := NewSlider(0, 100).BindValue(vol)
	s.SetTheme(th)
	// Alça de 16px (escala 1): curso útil de x=8 a x=108 → 1px por unidade.
	s.Layout(image.Rect(0, 0, 116, 24))

	// Clique no meio do curso posiciona o valor e inicia o arraste.
	s.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: image.Pt(58, 12), Button: event.MouseButtonLeft})
	if s.Value() != 50 || vol.Get() != 50 {
		t.Fatalf("após clique no meio: Value=%v, State=%v; esperado 50", s.Value(), vol.Get())
	}

	// Arraste com captura: o movimento vale mesmo fora dos bounds (clamp).
	s.HandleEvent(event.MouseEvent{Kind: event.MouseMove, Pos: image.Pt(83, 12), Button: event.MouseButtonLeft})
	if s.Value() != 75 {
		t.Fatalf("após arrastar até 83px: Value=%v, esperado 75", s.Value())
	}
	s.HandleEvent(event.MouseEvent{Kind: event.MouseMove, Pos: image.Pt(500, -40), Button: event.MouseButtonLeft})
	if s.Value() != 100 {
		t.Fatalf("arraste além do fim deveria limitar ao Max; Value=%v", s.Value())
	}
	s.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: image.Pt(500, -40), Button: event.MouseButtonLeft})

	// Solto: mover sem arrastar não muda nada.
	if s.HandleEvent(event.MouseEvent{Kind: event.MouseMove, Pos: image.Pt(58, 12), Button: event.MouseButtonLeft}) {
		t.Fatal("event.MouseMove sem arraste não deveria ser consumido")
	}

	// Teclado: setas usam Step (5% = 5), Home/End vão aos extremos.
	s.HandleEvent(event.KeyEvent{Key: event.KeyLeft})
	if s.Value() != 95 || vol.Get() != 95 {
		t.Fatalf("após seta esquerda: Value=%v, State=%v; esperado 95", s.Value(), vol.Get())
	}
	s.HandleEvent(event.KeyEvent{Key: event.KeyHome})
	if s.Value() != 0 {
		t.Fatalf("Home deveria ir ao Min; Value=%v", s.Value())
	}

	// Set externo move o slider; valores fora do intervalo são limitados.
	vol.Set(60)
	if s.Value() != 60 {
		t.Fatalf("Set externo: Value=%v, esperado 60", s.Value())
	}
	s.SetValue(1000)
	if s.Value() != 100 {
		t.Fatalf("SetValue além do Max deveria limitar; Value=%v", s.Value())
	}
}

func TestInputOnSubmit(t *testing.T) {
	th := newTestTheme(t)
	enviado := 0
	in := NewInput("x").OnSubmit(func() { enviado++ })
	in.SetTheme(th)
	in.HandleEvent(event.FocusEvent{Gained: true})
	if !in.HandleEvent(event.KeyEvent{Key: event.KeyEnter}) {
		t.Fatal("Enter deveria ser consumido quando há OnSubmit")
	}
	if enviado != 1 {
		t.Fatalf("OnSubmit deveria disparar 1 vez; disparou %d", enviado)
	}

	// Sem OnSubmit, Enter não é consumido (sobe para quem interessar).
	outro := NewInput("y")
	outro.SetTheme(th)
	if outro.HandleEvent(event.KeyEvent{Key: event.KeyEnter}) {
		t.Fatal("sem OnSubmit, Enter não deveria ser consumido")
	}
}
