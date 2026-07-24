package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
	"github.com/JonathanSantos/JUIGo/widget"
)

// TestPilhaDeOverlays é a regressão do slot único de overlay: abrir o popup
// de um Dropdown DENTRO de um Modal substituía o modal como overlay, e
// fechar o popup (Escape/seleção/clique fora) zerava a camada — o modal
// sumia da tela sem OnClose. Com a pilha, cada fechamento remove só a
// camada do topo e devolve o foco ao dono dela.
func TestPilhaDeOverlays(t *testing.T) {
	drop := juigo.NewDropdown("Um", "Dois", "Três")
	m := juigo.NewModal(juigo.NewVBox(juigo.NewText("Diálogo"), drop))
	fechado := 0
	m.OnClose(func() { fechado++ })

	h := uitest.New(t, juigo.NewVBox(juigo.NewButton("Abrir", m.Show)).Pad(8), 400, 300)
	th := h.Session().Theme()

	h.Click(uitest.Text("Abrir"))
	h.Layout() // painel centralizado
	h.Click(uitest.Text("Um")) // gatilho do dropdown dentro do modal
	if n := len(h.Session().Overlays()); n != 2 {
		t.Fatalf("popup dentro do modal deveria EMPILHAR; camadas=%d", n)
	}

	// Escape fecha SÓ o popup: o modal continua aberto, sem OnClose, e o
	// foco volta ao gatilho (era exatamente aqui que o modal sumia).
	h.Key(juigo.KeyEscape)
	if h.Session().Overlay() != juigo.Widget(m) {
		t.Fatalf("Escape deveria fechar só o popup; overlay=%T", h.Session().Overlay())
	}
	if !m.Shown() || fechado != 0 {
		t.Fatalf("modal não deveria fechar; shown=%v fechado=%d", m.Shown(), fechado)
	}
	if h.Focused() != juigo.Widget(drop) {
		t.Fatalf("foco deveria voltar ao gatilho; focado: %T", h.Focused())
	}

	// Seleção por clique fecha o popup e o modal segue aberto.
	h.Key(juigo.KeyEnter) // reabre pelo teclado (foco já está no gatilho)
	itemH := th.LineHeight() + th.PaddingPx()
	pb := h.Session().Overlay().Bounds()
	h.ClickAt(image.Pt(pb.Min.X+10, pb.Min.Y+th.BorderPx()+itemH+itemH/2))
	if drop.Value() != "Dois" {
		t.Fatalf("clique no 2º item: Value=%q", drop.Value())
	}
	if h.Session().Overlay() != juigo.Widget(m) || !m.Shown() {
		t.Fatal("selecionar no popup não deveria fechar o modal")
	}

	// Clique fora do popup (no painel do modal) fecha só o topo e é engolido.
	h.Key(juigo.KeyEnter)
	h.Click(uitest.Text("Diálogo")) // acima do gatilho, fora do popup
	if h.Session().Overlay() != juigo.Widget(m) || drop.Value() != "Dois" {
		t.Fatalf("clique fora deveria fechar só o popup; overlay=%T Value=%q",
			h.Session().Overlay(), drop.Value())
	}

	// Rolagem fora do popup também fecha só o topo.
	h.Key(juigo.KeyEnter)
	h.Session().Scroll(image.Pt(2, 2), 0, -1)
	if h.Session().Overlay() != juigo.Widget(m) {
		t.Fatal("rolagem fora deveria fechar só o popup")
	}

	// Sem popup, Escape fecha o modal e restaura o foco ao botão de abrir.
	h.Key(juigo.KeyEscape)
	if m.Shown() || h.Session().Overlay() != nil || fechado != 1 {
		t.Fatalf("Escape deveria fechar o modal; shown=%v fechado=%d", m.Shown(), fechado)
	}

	// Fechar o modal programaticamente com o popup aberto derruba a camada
	// de cima junto (aberta a partir do conteúdo dele, não faz sentido órfã).
	h.Click(uitest.Text("Abrir"))
	h.Layout()
	h.Click(uitest.Text("Dois"))
	m.Close()
	if len(h.Session().Overlays()) != 0 || fechado != 2 {
		t.Fatalf("Close deveria derrubar modal e popup; camadas=%d fechado=%d",
			len(h.Session().Overlays()), fechado)
	}
}

// TestTabCirculaNaCamadaDoTopo garante que Tab e o foco ficam restritos à
// camada do TOPO da pilha: num popup aberto de dentro de um modal, o Tab
// circula entre os focáveis do popup sem escapar para o modal.
func TestTabCirculaNaCamadaDoTopo(t *testing.T) {
	a := juigo.NewInput("a")
	b := juigo.NewInput("b")
	pop := juigo.NewPopup(juigo.NewVBox(a, b))
	interno := juigo.NewInput("interno")
	m := juigo.NewModal(juigo.NewVBox(
		interno,
		juigo.NewButton("Menu", func() { pop.ShowAt(image.Pt(50, 50)) }),
	))

	h := uitest.New(t, juigo.NewVBox(juigo.NewButton("Abrir", m.Show)).Pad(8), 400, 300)
	h.Click(uitest.Text("Abrir"))
	h.Layout()
	h.Click(uitest.Text("Menu"))
	if h.Focused() != juigo.Widget(a) {
		t.Fatalf("popup deveria abrir focando o 1º campo; focado: %T", h.Focused())
	}

	// Duas voltas completas: o foco nunca cai no campo do modal.
	for i := 0; i < 6; i++ {
		h.Key(juigo.KeyTab)
		if h.Focused() == juigo.Widget(interno) {
			t.Fatalf("Tab %d escapou do popup para o modal", i+1)
		}
	}
	if !pop.Shown() || len(h.Session().Overlays()) != 2 {
		t.Fatal("o Tab não deveria fechar nenhuma camada")
	}
}

// TestModalSobreDropdownCedeLugar cobre o empilhamento na ordem inversa: com
// o popup de um Dropdown aberto na raiz, um modal aberto POR CIMA rouba o
// foco; o popup fecha sozinho (comportamento de menu), sem derrubar o modal
// recém-nascido, e a restauração de foco dele é herdada pelo modal.
func TestModalSobreDropdownCedeLugar(t *testing.T) {
	raiz := juigo.NewDropdown("Alfa", "Beta")
	m := juigo.NewModal(juigo.NewVBox(juigo.NewText("Sobreposto")))
	h := uitest.New(t, juigo.NewVBox(raiz).Pad(8), 400, 300)

	h.Click(uitest.Text("Alfa")) // abre o popup na raiz
	m.Show()                     // modal empilha por cima e rouba o foco
	ovs := h.Session().Overlays()
	if len(ovs) != 1 || ovs[0] != juigo.Widget(m) || !m.Shown() {
		t.Fatalf("popup deveria ceder lugar ao modal; camadas=%v", ovs)
	}
	// Fechar o modal devolve o foco ao gatilho do dropdown, herdado do
	// popup (a cadeia não pode apontar para a camada já fechada).
	h.Key(juigo.KeyEscape)
	if h.Focused() != juigo.Widget(raiz) {
		t.Fatalf("foco deveria voltar ao gatilho na raiz; focado: %T", h.Focused())
	}
}

// camadaAncorada é uma overlay mínima focável e NÃO-SpansWindow — o análogo
// de um popup ancorado custom — para exercitar a remoção de camadas do MEIO
// da pilha no Resize (os ancorados embutidos fecham sozinhos ao perder o
// foco, nunca ficando sob outra camada).
type camadaAncorada struct{ widget.BaseWidget }

func (c *camadaAncorada) Focusable() bool { return true }

// TestResizeFechaSoCamadasAncoradas cobre o Resize com a pilha: camadas
// ancoradas no layout antigo fecham em QUALQUER posição da pilha; as de
// janela inteira (Modal) permanecem — inclusive herdando a restauração de
// foco de uma camada removida embaixo delas.
func TestResizeFechaSoCamadasAncoradas(t *testing.T) {
	// Popup do dropdown ACIMA do modal: só ele fecha no resize.
	drop := juigo.NewDropdown("Um", "Dois")
	m := juigo.NewModal(juigo.NewVBox(juigo.NewText("Diálogo"), drop))
	h := uitest.New(t, juigo.NewVBox(juigo.NewButton("Abrir", m.Show)).Pad(8), 400, 300)

	h.Click(uitest.Text("Abrir"))
	h.Layout()
	h.Click(uitest.Text("Um"))
	h.Session().Resize(image.Pt(400, 300))
	if h.Session().Overlay() != juigo.Widget(m) || !m.Shown() {
		t.Fatalf("resize deveria fechar só o popup; overlay=%T", h.Session().Overlay())
	}
	h.Key(juigo.KeyEscape)

	// Camada ancorada ABAIXO do modal: o resize remove a camada do meio e o
	// modal herda dela o foco a restaurar.
	campo := juigo.NewInput("campo")
	m2 := juigo.NewModal(juigo.NewVBox(juigo.NewText("Sobreposto")))
	h2 := uitest.New(t, juigo.NewVBox(campo).Pad(8), 400, 300)

	h2.Click(uitest.Placeholder("campo"))
	anc := &camadaAncorada{}
	anc.Layout(image.Rect(50, 50, 150, 120))
	h2.Session().OpenOverlay(anc)
	m2.Show()
	if n := len(h2.Session().Overlays()); n != 2 {
		t.Fatalf("modal sobre a camada ancorada deveria empilhar; camadas=%d", n)
	}
	h2.Session().Resize(image.Pt(400, 300))
	ovs := h2.Session().Overlays()
	if len(ovs) != 1 || ovs[0] != juigo.Widget(m2) {
		t.Fatalf("resize deveria remover só a camada ancorada do meio; camadas=%v", ovs)
	}
	// Fechar o modal devolve o foco ao campo, herdado da camada removida.
	h2.Key(juigo.KeyEscape)
	if h2.Focused() != juigo.Widget(campo) {
		t.Fatalf("foco deveria voltar ao campo; focado: %T", h2.Focused())
	}
}
