package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestTabs cobre o widget de abas: só a página ativa existe para o
// roteamento e os seletores, clique e setas trocam a aba, o State vinculado
// sincroniza nas duas vias e o sublinhado da ativa aparece nos pixels.
func TestTabs(t *testing.T) {
	aba := juigo.NewState(0)
	trocas := 0
	campoB := juigo.NewInput("Campo B")
	tabs := juigo.NewTabs().
		Add("Geral", juigo.NewVBox(juigo.NewText("Conteúdo A"), juigo.NewButton("Ação A", nil))).
		Add("Extras", juigo.NewVBox(campoB)).
		BindSelected(aba).
		OnChange(func(int) { trocas++ })
	h := uitest.New(t, tabs, 420, 240)

	th := h.Session().Theme()
	pad := th.PaddingPx()
	h.Layout()
	b := tabs.Bounds()
	w1 := th.MeasureString("Geral") + 2*pad
	w2 := th.MeasureString("Extras") + 2*pad
	barH := th.LineHeight() + 2*pad
	meioBarra := b.Min.Y + barH/2

	// Página dormente fora dos seletores e do hit-test.
	if h.Find(uitest.Text("Conteúdo A")) == nil {
		t.Fatal("a página ativa deveria estar visível")
	}
	if got := h.FindAll(uitest.Placeholder("Campo B")); len(got) != 0 {
		t.Fatal("a página dormente não deveria aparecer nos seletores")
	}

	// Sublinhado da aba ativa nos pixels.
	img := h.Screenshot()
	if got := img.RGBAAt(b.Min.X+w1/2, b.Min.Y+barH-1); got != th.Accent {
		t.Fatalf("sublinhado da aba ativa deveria ser %v; veio %v", th.Accent, got)
	}
	if got := img.RGBAAt(b.Min.X+w1+w2/2, b.Min.Y+barH-1); got == th.Accent {
		t.Fatal("a aba inativa não deveria ter sublinhado")
	}

	// Clique na segunda aba troca página, State e dispara OnChange.
	h.ClickAt(image.Pt(b.Min.X+w1+w2/2, meioBarra))
	if tabs.Selected() != 1 || aba.Get() != 1 || trocas != 1 {
		t.Fatalf("clique na aba: selected=%d state=%d trocas=%d", tabs.Selected(), aba.Get(), trocas)
	}
	if h.Find(uitest.Placeholder("Campo B")) == nil {
		t.Fatal("a página Extras deveria estar visível após o clique")
	}
	if got := h.FindAll(uitest.Text("Conteúdo A")); len(got) != 0 {
		t.Fatal("a página Geral deveria ficar dormente após a troca")
	}

	// A página ativa é interativa.
	h.Click(uitest.Placeholder("Campo B"))
	h.Type("olá")
	if campoB.Text() != "olá" {
		t.Fatalf("digitação na página ativa: %q", campoB.Text())
	}

	// Teclado na barra focada: setas movem com limite nas pontas; Home/End
	// vão aos extremos.
	h.ClickAt(image.Pt(b.Min.X+w2/2, meioBarra)) // foca a barra (aba 0 de novo)
	if tabs.Selected() != 0 || h.Focused() != juigo.Widget(tabs) {
		t.Fatalf("clique na primeira aba deveria selecionar e focar a barra; selected=%d", tabs.Selected())
	}
	h.Key(juigo.KeyRight)
	if tabs.Selected() != 1 {
		t.Fatal("seta direita deveria avançar a aba")
	}
	h.Key(juigo.KeyRight)
	if tabs.Selected() != 1 {
		t.Fatal("seta direita na última aba deveria ficar nela")
	}
	h.Key(juigo.KeyHome)
	if tabs.Selected() != 0 {
		t.Fatal("Home deveria voltar à primeira aba")
	}
	h.Key(juigo.KeyEnd)
	if tabs.Selected() != 1 {
		t.Fatal("End deveria ir à última aba")
	}

	// Set externo troca a aba; fora do intervalo ajusta e ressincroniza.
	aba.Set(0)
	if tabs.Selected() != 0 {
		t.Fatal("Set externo deveria trocar a aba")
	}
	aba.Set(7)
	if tabs.Selected() != 1 || aba.Get() != 1 {
		t.Fatalf("Set fora do intervalo deveria ajustar: selected=%d state=%d", tabs.Selected(), aba.Get())
	}
}
