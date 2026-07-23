package uitest_test

import (
	"image"
	"testing"

	"juigo"
	"juigo/uitest"
)

// TestCliqueFocaEDigita cobre o fluxo básico: clique foca o campo, digitação
// flui para o State vinculado.
func TestCliqueFocaEDigita(t *testing.T) {
	valor := juigo.NewState("")
	campo := juigo.NewInput("Digite…").BindValue(valor)
	ui := juigo.NewVBox(
		campo,
		juigo.NewButton("Enviar", nil),
	).Pad(8)

	h := uitest.New(t, ui, 400, 200)
	h.Click(uitest.Placeholder("Digite…"))
	if h.Focused() != juigo.Widget(campo) {
		t.Fatalf("clique deveria focar o campo; focado: %T", h.Focused())
	}
	h.Type("olá, ação")
	if valor.Get() != "olá, ação" {
		t.Fatalf("State = %q", valor.Get())
	}

	// Clique em área vazia limpa o foco.
	h.ClickAt(image.Pt(390, 190))
	if h.Focused() != nil {
		t.Fatalf("clique no vazio deveria limpar o foco; focado: %T", h.Focused())
	}
}

// TestTabCircula cobre a navegação de foco do App (antes intestável).
func TestTabCircula(t *testing.T) {
	campo := juigo.NewInput("a")
	btn := juigo.NewButton("OK", nil)
	check := juigo.NewCheckbox("C")
	h := uitest.New(t, juigo.NewVBox(campo, btn, check), 400, 300)

	h.Key(juigo.KeyTab)
	if h.Focused() != juigo.Widget(campo) {
		t.Fatalf("1º Tab deveria focar o campo; focado: %T", h.Focused())
	}
	h.Key(juigo.KeyTab)
	if h.Focused() != juigo.Widget(btn) {
		t.Fatalf("2º Tab deveria focar o botão; focado: %T", h.Focused())
	}
	h.Key(juigo.KeyTab)
	h.Key(juigo.KeyTab) // wraparound
	if h.Focused() != juigo.Widget(campo) {
		t.Fatalf("Tab deveria dar a volta; focado: %T", h.Focused())
	}
	h.Key(juigo.KeyTab, juigo.ModShift) // Shift+Tab recua
	if h.Focused() != juigo.Widget(check) {
		t.Fatalf("Shift+Tab deveria recuar ao checkbox; focado: %T", h.Focused())
	}
}

// TestHoverECursor cobre Enter/Leave e o formato do cursor.
func TestHoverECursor(t *testing.T) {
	btn := juigo.NewButton("OK", nil)
	campo := juigo.NewInput("x")
	h := uitest.New(t, juigo.NewVBox(btn, campo), 400, 200)

	h.Hover(uitest.Text("OK"))
	if btn.State() != juigo.ButtonStateHover {
		t.Fatalf("hover deveria realçar o botão; estado %v", btn.State())
	}
	if h.Session().CursorShape() != juigo.CursorHand {
		t.Fatal("cursor sobre o botão deveria ser mãozinha")
	}
	h.Hover(uitest.Placeholder("x"))
	if btn.State() != juigo.ButtonStateNormal {
		t.Fatal("sair do botão deveria remover o realce")
	}
	if h.Session().CursorShape() != juigo.CursorText {
		t.Fatal("cursor sobre o campo deveria ser I-beam")
	}
}

// TestArrasteComCaptura cobre a captura de mouse de ponta a ponta: o arraste
// do slider continua valendo fora dos bounds.
func TestArrasteComCaptura(t *testing.T) {
	s := juigo.NewSlider(0, 100)
	h := uitest.New(t, juigo.NewVBox(s).Pad(0), 116, 40)

	from := image.Pt(58, s.Bounds().Min.Y+s.Bounds().Dy()/2)
	h.Drag(from, image.Pt(999, -50)) // solta bem fora da janela
	if s.Value() != 100 {
		t.Fatalf("arraste além do fim deveria saturar no Max; Value=%v", s.Value())
	}
}

// TestDropdownOverlayEngoleClique cobre o ciclo do popup: abrir, selecionar
// por clique, e o clique fora FECHANDO e sendo engolido (não ativa o que
// está por baixo).
func TestDropdownOverlayEngoleClique(t *testing.T) {
	cliques := 0
	btn := juigo.NewButton("Fora", func() { cliques++ })
	drop := juigo.NewDropdown("Baixa", "Média", "Alta")
	h := uitest.New(t, juigo.NewVBox(btn, drop).Pad(8), 400, 300)
	th := h.Session().Theme()

	// Abre o popup e seleciona "Alta" clicando no 3º item.
	h.Click(uitest.Text("Baixa")) // valor atual do dropdown
	ov := h.Session().Overlay()
	if ov == nil {
		t.Fatal("clique no dropdown deveria abrir o popup")
	}
	itemH := th.LineHeight() + th.PaddingPx()
	pb := ov.Bounds()
	h.ClickAt(image.Pt(pb.Min.X+10, pb.Min.Y+th.BorderPx()+2*itemH+itemH/2))
	if drop.Value() != "Alta" || h.Session().Overlay() != nil {
		t.Fatalf("seleção por clique: Value=%q overlay=%v", drop.Value(), h.Session().Overlay())
	}

	// Reabre e clica no botão FORA do popup: fecha e o clique é engolido.
	h.Click(uitest.Text("Alta"))
	if h.Session().Overlay() == nil {
		t.Fatal("popup deveria reabrir")
	}
	h.Click(uitest.Text("Fora"))
	if h.Session().Overlay() != nil {
		t.Fatal("clique fora deveria fechar o popup")
	}
	if cliques != 0 {
		t.Fatalf("clique fora deveria ser ENGOLIDO; o botão disparou %d vez(es)", cliques)
	}
	// Sem popup, o mesmo clique dispara normalmente.
	h.Click(uitest.Text("Fora"))
	if cliques != 1 {
		t.Fatalf("sem popup o botão deveria disparar; cliques=%d", cliques)
	}
}

// TestModalEscapeComFocoInterno cobre o fallback de Escape do App: fecha o
// modal mesmo com o foco em um campo DENTRO dele, e restaura o foco anterior.
func TestModalEscapeComFocoInterno(t *testing.T) {
	campoFora := juigo.NewInput("fora")
	interno := juigo.NewInput("dentro")
	m := juigo.NewModal(juigo.NewVBox(juigo.NewText("Diálogo"), interno))
	fechado := 0
	m.OnClose(func() { fechado++ })

	h := uitest.New(t, juigo.NewVBox(campoFora, juigo.NewButton("Abrir", m.Show)).Pad(8), 400, 300)

	h.Click(uitest.Placeholder("fora")) // foco inicial no campo de fora
	h.Click(uitest.Text("Abrir"))
	if h.Session().Overlay() == nil || !m.Shown() {
		t.Fatal("modal deveria abrir")
	}
	h.Layout() // painel centralizado

	// Foca o campo interno e digita.
	h.Click(uitest.Placeholder("dentro"))
	if h.Focused() != juigo.Widget(interno) {
		t.Fatalf("campo interno deveria focar; focado: %T", h.Focused())
	}
	h.Type("çã")
	if interno.Text() != "çã" {
		t.Fatalf("digitação no modal: %q", interno.Text())
	}

	// Tab circula SÓ dentro do modal.
	h.Key(juigo.KeyTab)
	if h.Focused() == juigo.Widget(campoFora) {
		t.Fatal("Tab não deveria escapar do modal")
	}

	// Escape com foco interno fecha (fallback do App para a overlay).
	h.Key(juigo.KeyEscape)
	if m.Shown() || h.Session().Overlay() != nil || fechado != 1 {
		t.Fatalf("Escape deveria fechar o modal; shown=%v fechado=%d", m.Shown(), fechado)
	}
	// Foco restaurado para o botão que abriu.
	if h.Focused() == nil || h.Focused() == juigo.Widget(interno) {
		t.Fatalf("foco deveria ser restaurado ao fechar; focado: %T", h.Focused())
	}
}

// TestTooltipComRelogioVirtual cobre o atraso do tooltip sem sleep.
func TestTooltipComRelogioVirtual(t *testing.T) {
	btn := juigo.Tooltip(juigo.NewButton("Enviar", nil), "Envia o formulário")
	h := uitest.New(t, juigo.NewVBox(btn).Pad(8), 400, 200)
	th := h.Session().Theme()

	h.Hover(uitest.Text("Enviar"))
	if h.Session().TooltipVisible() {
		t.Fatal("tooltip não deveria aparecer antes do atraso")
	}
	h.Advance(th.TooltipDelay)
	if !h.Session().TooltipVisible() {
		t.Fatal("tooltip deveria aparecer após o atraso")
	}
	// A dica está na imagem composta.
	img := h.Screenshot()
	achou := false
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] == th.TooltipBackground.R && img.Pix[i+1] == th.TooltipBackground.G && img.Pix[i+2] == th.TooltipBackground.B {
			achou = true
			break
		}
	}
	if !achou {
		t.Fatal("screenshot deveria conter a caixa da dica")
	}
	// Mover esconde.
	h.MoveTo(image.Pt(390, 190))
	if h.Session().TooltipVisible() {
		t.Fatal("mover para fora deveria esconder a dica")
	}
}

// TestCaretPiscaComRelogioVirtual observa a piscada do cursor por pixels.
func TestCaretPiscaComRelogioVirtual(t *testing.T) {
	campo := juigo.NewInput("")
	h := uitest.New(t, juigo.NewVBox(campo).Pad(0), 300, 40)
	th := h.Session().Theme()

	h.Click(uitest.OfType[*juigo.Input]())
	caret := image.Pt(campo.Bounds().Min.X+th.PaddingPx(), campo.Bounds().Min.Y+campo.Bounds().Dy()/2)

	corDoCaret := func() bool {
		img := h.Screenshot()
		return img.RGBAAt(caret.X, caret.Y) == th.Cursor
	}
	if !corDoCaret() {
		t.Fatal("caret deveria estar visível logo após focar")
	}
	h.Advance(th.CaretBlink)
	if corDoCaret() {
		t.Fatal("caret deveria apagar após um intervalo de piscada")
	}
	h.Advance(th.CaretBlink)
	if !corDoCaret() {
		t.Fatal("caret deveria reacender no intervalo seguinte")
	}
	// Digitar reacende na hora.
	h.Advance(th.CaretBlink) // apaga
	h.Type("a")
	img := h.Screenshot()
	x := campo.Bounds().Min.X + th.PaddingPx() + th.MeasureString("a")
	if img.RGBAAt(x, caret.Y) != th.Cursor {
		t.Fatal("digitar deveria reacender o caret imediatamente")
	}
}

// TestRolagemPeloHarness cobre a rolagem roteada por geometria.
func TestRolagemPeloHarness(t *testing.T) {
	lista := juigo.NewVBox().Gap(0).Pad(0)
	for i := 0; i < 30; i++ {
		lista.Add(juigo.NewText("linha"))
	}
	sc := juigo.NewScroll(lista)
	h := uitest.New(t, juigo.Grow(sc, 1), 300, 120)

	h.Scroll(uitest.OfType[*juigo.Scroll](), -2)
	if sc.Offset() == 0 {
		t.Fatal("rolagem deveria mover o conteúdo")
	}
}
