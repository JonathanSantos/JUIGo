package uitest_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestNanquimPorPixels confere a identidade na tela: fundo quase branco e
// ação em tinta preta; no escuro, fundo de carvão e a tinta INVERTIDA
// (ação clara) — a troca em runtime sem reconstruir a árvore.
func TestNanquimPorPixels(t *testing.T) {
	ui := juigo.NewVBox(juigo.NewButton("Ação", nil)).Pad(20)
	h := uitest.New(t, ui, 300, 200)

	papel := color.RGBA{R: 0xFA, G: 0xFA, B: 0xFA, A: 0xFF}
	tinta := color.RGBA{R: 0x11, G: 0x11, B: 0x11, A: 0xFF}

	img := h.Screenshot()
	if got := img.RGBAAt(2, 2); got != papel {
		t.Fatalf("o fundo deveria ser %v; veio %v", papel, got)
	}
	btn := h.Find(uitest.Text("Ação")).Bounds()
	dentro := image.Pt(btn.Min.X+8, btn.Min.Y+btn.Dy()/2)
	if got := img.RGBAAt(dentro.X, dentro.Y); got != tinta {
		t.Fatalf("o botão deveria ser tinta %v; veio %v", tinta, got)
	}

	escuro, err := juigo.DarkTheme()
	if err != nil {
		t.Fatal(err)
	}
	h.Session().SetTheme(escuro)
	img = h.Screenshot()
	if got := img.RGBAAt(2, 2); got != escuro.Background {
		t.Fatalf("no escuro o fundo deveria ser %v; veio %v", escuro.Background, got)
	}
	if got := img.RGBAAt(dentro.X, dentro.Y); got != escuro.Text {
		t.Fatalf("no escuro a ação deveria ser a tinta invertida %v; veio %v", escuro.Text, got)
	}
}

// TestTemaPorSementes: um tema feito com NewTheme pinta a ação escolhida e
// passa na auditoria de contraste.
func TestTemaPorSementes(t *testing.T) {
	cobalto := color.RGBA{R: 0x2B, G: 0x59, B: 0xD9, A: 0xFF}
	th, err := juigo.NewTheme(juigo.Palette{Primary: cobalto})
	if err != nil {
		t.Fatal(err)
	}
	if issues := th.Audit(); len(issues) > 0 {
		t.Fatalf("o tema por sementes deveria passar na auditoria: %v", issues)
	}
	ui := juigo.NewVBox(juigo.NewButton("Ação", nil)).Pad(20)
	h := uitest.NewWithTheme(t, ui, th, 300, 200)
	btn := h.Find(uitest.Text("Ação")).Bounds()
	if got := h.Screenshot().RGBAAt(btn.Min.X+8, btn.Min.Y+btn.Dy()/2); got != cobalto {
		t.Fatalf("o botão deveria ser a semente %v; veio %v", cobalto, got)
	}
}

// TestCheckboxMarcadoPreenche: marcado, a caixa inteira vira Accent com o
// check em AccentText no meio do traço.
func TestCheckboxMarcadoPreenche(t *testing.T) {
	cb := juigo.NewCheckbox("Avisos")
	ui := juigo.NewVBox(cb).Pad(20)
	h := uitest.New(t, ui, 300, 200)
	th := h.Session().Theme()
	side := th.LineHeight()
	b := cb.Bounds()
	box := image.Rect(b.Min.X, b.Min.Y+(b.Dy()-side)/2, b.Min.X+side, b.Min.Y+(b.Dy()-side)/2+side)
	canto := image.Pt(box.Min.X+side/4, box.Max.Y-side/4) // longe do check e do anel
	if got := h.Screenshot().RGBAAt(canto.X, canto.Y); got == th.Accent {
		t.Fatal("desmarcado, a caixa não deveria ser preenchida")
	}
	h.Click(uitest.Text("Avisos"))
	img := h.Screenshot()
	if got := img.RGBAAt(canto.X, canto.Y); got != th.Accent {
		t.Fatalf("marcado, a caixa deveria ser %v; veio %v", th.Accent, got)
	}
	vertice := image.Pt(box.Min.X+side*43/100, box.Min.Y+side*68/100)
	if got := img.RGBAAt(vertice.X, vertice.Y); got != th.AccentText {
		t.Fatalf("o vértice do check deveria ser %v; veio %v", th.AccentText, got)
	}
}

// TestListaConteudoDentroDaPilula: numa lista selecionável o texto da
// linha nasce dentro da pílula de realce, não colado na borda.
func TestListaConteudoDentroDaPilula(t *testing.T) {
	lista := juigo.NewTextList(5, func(i int) string { return "Linha" }).BindSelected(juigo.NewState(0))
	h := uitest.New(t, juigo.NewVBox(lista), 300, 200)
	th := h.Session().Theme()
	linha := h.FindAll(uitest.Text("Linha"))[0].Bounds()
	quer := lista.Bounds().Min.X + th.Px(th.RowPad) + th.PaddingPx()
	if linha.Min.X != quer {
		t.Fatalf("o conteúdo deveria começar em x=%d (pílula + respiro); veio %d", quer, linha.Min.X)
	}
}

// TestTonsDeEstado: Success e Warning desenham nas cores do tema e seguem
// a troca de tema em runtime.
func TestTonsDeEstado(t *testing.T) {
	ok := juigo.NewText("████").Success()
	aviso := juigo.NewText("████").Warning()
	h := uitest.New(t, juigo.NewVBox(ok, aviso).Pad(10), 300, 200)
	conta := func(w juigo.Widget, c color.RGBA) int {
		img := h.Screenshot()
		n := 0
		b := w.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if img.RGBAAt(x, y) == c {
					n++
				}
			}
		}
		return n
	}
	th := h.Session().Theme()
	if conta(ok, th.Success) == 0 || conta(aviso, th.Warning) == 0 {
		t.Fatal("Success/Warning deveriam desenhar nas cores de estado do tema")
	}
	escuro, _ := juigo.DarkTheme()
	h.Session().SetTheme(escuro)
	if conta(ok, escuro.Success) == 0 {
		t.Fatal("o tom deveria seguir a troca de tema")
	}
}

// contaCor conta os pixels de img com a cor exata c.
func contaCor(img *image.RGBA, c color.RGBA) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y) == c {
				n++
			}
		}
	}
	return n
}

// TestDropdownEscolhidaEmPilula: no popup, a opção escolhida é uma pílula
// de Selection — forma, não cor de texto (que somiria com ação em tinta).
func TestDropdownEscolhidaEmPilula(t *testing.T) {
	d := juigo.NewDropdown("Baixa", "Média", "Alta").BindValue(juigo.NewState("Média"))
	h := uitest.New(t, juigo.NewVBox(d).Pad(10), 300, 260)
	th := h.Session().Theme()
	if contaCor(h.Screenshot(), th.Selection) != 0 {
		t.Fatal("fechado, não deveria haver seleção na tela")
	}
	h.Click(uitest.OfType[*juigo.Dropdown]())
	h.MoveTo(image.Pt(1, 1)) // ponteiro fora do popup: sem hover por cima
	if contaCor(h.Screenshot(), th.Selection) < 50 {
		t.Fatal("aberto, a opção escolhida deveria estar numa pílula de Selection")
	}
}

// TestCalendarioHojeComAnel: o dia de hoje (não selecionado) ganha um anel
// na cor de ação.
func TestCalendarioHojeComAnel(t *testing.T) {
	cal := juigo.NewCalendar()
	h := uitest.New(t, juigo.NewVBox(cal).Pad(12), 360, 360)
	th := h.Session().Theme()
	if contaCor(h.Screenshot(), th.Accent) < 20 {
		t.Fatal("o dia de hoje deveria ter um anel na cor de ação")
	}
}

// TestBotaoDesabilitadoNeutro: desabilitado, o primário vira uma placa
// neutra (HoverBackground) com rótulo apagado — sem a ação e sem lavagem.
func TestBotaoDesabilitadoNeutro(t *testing.T) {
	btn := juigo.NewButton("Enviar", nil).BindDisabled(juigo.NewState(true))
	for _, criar := range []func() (*juigo.Theme, error){juigo.DefaultTheme, juigo.DarkTheme} {
		th, err := criar()
		if err != nil {
			t.Fatal(err)
		}
		h := uitest.NewWithTheme(t, juigo.NewVBox(btn).Pad(10), th, 300, 120)
		img := h.Screenshot()
		b := btn.Bounds()
		if got := img.RGBAAt(b.Min.X+6, b.Min.Y+b.Dy()/2); got != th.HoverBackground {
			t.Fatalf("o fundo desabilitado deveria ser %v; veio %v", th.HoverBackground, got)
		}
		if contaCor(img, th.ButtonNormal) > 0 && th.ButtonNormal != th.Text {
			t.Fatal("a cor de ação não deveria aparecer no botão desabilitado")
		}
	}
}

// TestListaLarguraContaORecuo: a largura preferida de uma lista
// selecionável inclui o recuo da pílula dos dois lados.
func TestListaLarguraContaORecuo(t *testing.T) {
	simples := juigo.NewTextList(3, func(int) string { return "Linha" })
	selecionavel := juigo.NewTextList(3, func(int) string { return "Linha" }).BindSelected(juigo.NewState(0))
	h := uitest.New(t, juigo.NewVBox(simples, selecionavel), 300, 200)
	th := h.Session().Theme()
	quer := simples.PreferredSize().X + 2*(th.Px(th.RowPad)+th.PaddingPx())
	if got := selecionavel.PreferredSize().X; got != quer {
		t.Fatalf("largura preferida = %d, esperado %d", got, quer)
	}
}
