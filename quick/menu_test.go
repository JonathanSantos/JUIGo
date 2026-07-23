package quick_test

import (
	"image"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/uitest"
	"github.com/JonathanSantos/JUIGo/widget"
)

// TestMenuDeContexto cobre o quick.Menu: navegação por teclado, seleção por
// Enter e por clique (fecha ANTES de chamar a ação), e fechamento sem
// seleção por Escape e por clique fora.
func TestMenuDeContexto(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")).Pad(8), 320, 240)
	th := h.Session().Theme()
	pad := th.PaddingPx()
	itemH := th.LineHeight() + pad
	at := image.Pt(60, 40)

	escolhido := ""
	abrir := func() *juigo.Popup {
		return quick.Menu(at,
			quick.Item("Renomear…", func() { escolhido = "renomear" }),
			quick.Item("Excluir", func() { escolhido = "excluir" }),
		)
	}

	// Teclado: abre focado, seta desce, Enter seleciona e fecha.
	m := abrir()
	if !m.Shown() {
		t.Fatal("o menu deveria abrir")
	}
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEnter)
	if escolhido != "excluir" || m.Shown() {
		t.Fatalf("Enter deveria selecionar Excluir e fechar; escolhido=%q shown=%v", escolhido, m.Shown())
	}

	// Clique: primeira entrada, no meio da linha (conteúdo = painel com
	// respiro do tema).
	escolhido = ""
	m = abrir()
	h.ClickAt(at.Add(image.Pt(pad+4, pad+itemH/2)))
	if escolhido != "renomear" || m.Shown() {
		t.Fatalf("clique deveria selecionar Renomear e fechar; escolhido=%q shown=%v", escolhido, m.Shown())
	}

	// Escape fecha sem selecionar.
	escolhido = ""
	m = abrir()
	h.Key(juigo.KeyEscape)
	if escolhido != "" || m.Shown() {
		t.Fatalf("Escape não deveria selecionar; escolhido=%q shown=%v", escolhido, m.Shown())
	}

	// Clique fora fecha sem selecionar (e é engolido).
	m = abrir()
	h.ClickAt(image.Pt(300, 220))
	if escolhido != "" || m.Shown() {
		t.Fatalf("clique fora não deveria selecionar; escolhido=%q shown=%v", escolhido, m.Shown())
	}
}

// TestToast cobre o aviso transitório: aparece na base da janela com as
// cores do tooltip, substitui o atual, some sozinho após Theme.ToastDuration
// e pode ser escondido à mão.
func TestToast(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("app")).Pad(8), 300, 200)
	th := h.Session().Theme()
	pad := th.PaddingPx()

	quick.Toast("Salvo!")
	if !h.Session().ToastVisible() {
		t.Fatal("o toast deveria estar visível")
	}
	prefY := th.LineHeight() + pad
	topo := 200 - 2*pad - prefY
	img := h.Screenshot()
	if got := img.RGBAAt(150, topo+1); got != th.TooltipBackground {
		t.Fatalf("caixa do toast deveria ser %v; veio %v", th.TooltipBackground, got)
	}

	// Um novo toast substitui o atual.
	quick.Toast("Outro aviso mais comprido")
	if !h.Session().ToastVisible() {
		t.Fatal("o toast substituto deveria estar visível")
	}

	// Some sozinho após a duração do tema (relógio virtual).
	h.Advance(th.ToastDuration + time.Millisecond)
	if h.Session().ToastVisible() {
		t.Fatal("o toast deveria sumir sozinho")
	}
	img = h.Screenshot()
	if got := img.RGBAAt(150, topo+1); got != th.Background {
		t.Fatalf("após sumir, o fundo deveria voltar; veio %v", got)
	}

	// HideToast esconde na hora.
	widget.ShowToast("De novo", 0)
	if !h.Session().ToastVisible() {
		t.Fatal("ShowToast deveria exibir")
	}
	widget.HideToast()
	if h.Session().ToastVisible() {
		t.Fatal("HideToast deveria esconder na hora")
	}
}
