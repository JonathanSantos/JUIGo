package uitest_test

import (
	"image"
	"testing"

	"juigo"
	"juigo/uitest"
)

// TestDisabledBloqueiaInteracao cobre a aplicação central do disabled:
// clique não dispara, Tab pula, hover não realça, cursor fica padrão.
func TestDisabledBloqueiaInteracao(t *testing.T) {
	cliques := 0
	btn := juigo.NewButton("Enviar", func() { cliques++ })
	campo := juigo.NewInput("campo")
	h := uitest.New(t, juigo.NewVBox(campo, btn).Pad(8), 400, 200)

	btn.SetDisabled(true)

	// Clique não dispara nem foca.
	h.Click(uitest.Text("Enviar"))
	if cliques != 0 {
		t.Fatalf("botão desabilitado disparou %d vez(es)", cliques)
	}
	if h.Focused() == juigo.Widget(btn) {
		t.Fatal("botão desabilitado não deveria receber foco por clique")
	}

	// Hover não realça e o cursor fica padrão.
	h.Hover(uitest.Text("Enviar"))
	if btn.State() != juigo.ButtonStateNormal {
		t.Fatal("botão desabilitado não deveria mostrar hover")
	}
	if h.Session().CursorShape() != juigo.CursorDefault {
		t.Fatal("cursor sobre desabilitado deveria ser a seta padrão")
	}

	// Tab pula o desabilitado (campo → campo, com wraparound).
	h.Key(juigo.KeyTab)
	if h.Focused() != juigo.Widget(campo) {
		t.Fatalf("1º Tab deveria focar o campo; focado: %T", h.Focused())
	}
	h.Key(juigo.KeyTab)
	if h.Focused() != juigo.Widget(campo) {
		t.Fatalf("Tab deveria pular o botão desabilitado; focado: %T", h.Focused())
	}

	// Reabilitado, tudo volta.
	btn.SetDisabled(false)
	h.Click(uitest.Text("Enviar"))
	if cliques != 1 {
		t.Fatalf("botão reabilitado deveria disparar; cliques=%d", cliques)
	}
}

// TestDisabledEnquantoFocado cobre a fresta: o widget desabilita COM o foco
// nele (formulário que invalida) e o teclado para de chegar.
func TestDisabledEnquantoFocado(t *testing.T) {
	campo := juigo.NewInput("campo")
	h := uitest.New(t, juigo.NewVBox(campo).Pad(8), 400, 100)

	h.Click(uitest.Placeholder("campo"))
	h.Type("a")
	campo.SetDisabled(true)
	h.Type("b")
	h.Key(juigo.KeyBackspace)
	if campo.Text() != "a" {
		t.Fatalf("teclado não deveria alcançar campo desabilitado; Text=%q", campo.Text())
	}
}

// TestDisabledReativoELavagem cobre BindDisabled (o caminho do formulário) e
// a lavagem visual.
func TestDisabledReativoELavagem(t *testing.T) {
	valor := juigo.NewState("")
	vazio := juigo.Map(valor, func(s string) bool { return s == "" })

	cliques := 0
	btn := juigo.BindDisabled(juigo.NewButton("Enviar", func() { cliques++ }), vazio)
	campo := juigo.NewInput("Digite…").BindValue(valor)
	h := uitest.New(t, juigo.NewVBox(campo, btn).Pad(8), 400, 200)

	// Vazio: desabilitado de verdade.
	h.Click(uitest.Text("Enviar"))
	if cliques != 0 || !btn.Disabled() {
		t.Fatalf("com o campo vazio o botão deveria estar desabilitado (cliques=%d)", cliques)
	}

	// A lavagem aparece: pixel central do botão difere do azul normal.
	th := h.Session().Theme()
	img := h.Screenshot()
	centro := image.Pt((btn.Bounds().Min.X+btn.Bounds().Max.X)/2, btn.Bounds().Min.Y+2)
	if img.RGBAAt(centro.X, centro.Y) == th.ButtonNormal {
		t.Fatal("botão desabilitado deveria estar esmaecido pela lavagem")
	}

	// Digitou: reabilita via reatividade e dispara.
	h.Click(uitest.Placeholder("Digite…"))
	h.Type("olá")
	if btn.Disabled() {
		t.Fatal("digitar deveria reabilitar o botão via BindDisabled")
	}
	h.Click(uitest.Text("Enviar"))
	if cliques != 1 {
		t.Fatalf("botão reabilitado deveria disparar; cliques=%d", cliques)
	}
}

// TestLoadingSpinnerEBloqueio cobre o estado de carregamento: bloqueia como
// desabilitado e o spinner anima com o relógio virtual.
func TestLoadingSpinnerEBloqueio(t *testing.T) {
	cliques := 0
	btn := juigo.NewButton("Enviar", func() { cliques++ })
	h := uitest.New(t, juigo.NewVBox(btn).Pad(8), 300, 100)
	th := h.Session().Theme()

	btn.SetLoading(true)

	// Bloqueado para clique e Tab.
	h.Click(uitest.Text("Enviar"))
	if cliques != 0 {
		t.Fatalf("botão em loading disparou %d vez(es)", cliques)
	}
	h.Key(juigo.KeyTab)
	if h.Focused() != nil {
		t.Fatal("Tab não deveria alcançar botão em loading")
	}

	// Spinner visível e ANIMADO: o quadro muda após SpinnerStep.
	antes := h.Screenshot()
	h.Advance(th.SpinnerStep)
	depois := h.Screenshot()
	iguais := true
	for i := range antes.Pix {
		if antes.Pix[i] != depois.Pix[i] {
			iguais = false
			break
		}
	}
	if iguais {
		t.Fatal("o spinner deveria mudar de quadro após SpinnerStep")
	}

	// Sai do loading: rótulo volta e o clique funciona.
	btn.SetLoading(false)
	h.Click(uitest.Text("Enviar"))
	if cliques != 1 {
		t.Fatalf("após o loading o botão deveria disparar; cliques=%d", cliques)
	}
}
