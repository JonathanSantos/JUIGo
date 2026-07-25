package main

import (
	"os"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// harness monta a galeria com aoTema ligado à sessão do teste.
func harness(t *testing.T) (*galeria, *uitest.Harness) {
	t.Helper()
	var h *uitest.Harness
	g, err := nova(func(th *juigo.Theme) {
		if h != nil {
			h.Session().SetTheme(th)
			h.Session().InvalidateAll()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	h = uitest.NewLazy(t, func() juigo.Widget { return g.Raiz }, 1020, 660)
	return g, h
}

// TestGaleriaNavegaEBusca: a árvore troca de página, e a paleta Ctrl+K
// (alimentada pelo menu Componentes) salta para um componente pelo nome.
func TestGaleriaNavegaEBusca(t *testing.T) {
	g, h := harness(t)

	if h.Find(uitest.Text("Título em fonte de display")) == nil {
		t.Fatal("a página inicial deveria ser Tipografia")
	}
	h.Click(uitest.Text("Botões"))
	h.Layout()
	if h.Find(uitest.Text("Primário")) == nil {
		t.Fatal("clicar em Botões na árvore deveria abrir a página")
	}

	h.Key(juigo.KeyK, juigo.ModControl)
	h.Type("gráficos")
	h.Key(juigo.KeyEnter)
	h.Layout()
	if g.selecao.Get() != "Gráficos" {
		t.Fatalf("a paleta deveria abrir Gráficos; abriu %q", g.selecao.Get())
	}
	if h.Find(uitest.Text("Bars — negativos descem da linha do zero")) == nil {
		t.Fatal("a página de gráficos deveria estar em cena")
	}
}

// TestGaleriaKnobsDeEstilo: trocar o tema muda o papel de fundo; o tamanho
// re-rasteriza a fonte; Ctrl+T alterna claro/escuro.
func TestGaleriaKnobsDeEstilo(t *testing.T) {
	g, h := harness(t)
	g.aplicar() // tema inicial

	claro := g.temas["Papel e tinta"]
	img := h.Screenshot()
	if got := img.RGBAAt(510, 650); got != claro.Background {
		t.Fatalf("o fundo inicial deveria ser papel %v; veio %v", claro.Background, got)
	}

	g.temaNome.Set("Papel e tinta escuro")
	escuro := g.temas["Papel e tinta escuro"]
	img = h.Screenshot()
	if got := img.RGBAAt(510, 650); got != escuro.Background {
		t.Fatalf("trocar o tema deveria escurecer o fundo; veio %v", got)
	}

	g.tamanho.Set("18")
	if got := g.temaAtual().FontSize; got != 18 {
		t.Fatalf("o knob de tamanho deveria valer no tema; FontSize=%v", got)
	}

	h.Key(juigo.KeyT, juigo.ModControl) // atalho do menu: alternar
	if g.temaNome.Get() != "Papel e tinta" {
		t.Fatalf("Ctrl+T deveria voltar ao claro; ficou %q", g.temaNome.Get())
	}
}

// TestCapturaVisual salva docs/galeria.png (página Botões, papel e tinta)
// quando GALERIA_CAPTURA aponta um caminho:
//
//	GALERIA_CAPTURA=docs/galeria.png go test ./examples/galeria -run TestCapturaVisual
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("GALERIA_CAPTURA")
	if caminho == "" {
		t.Skip("defina GALERIA_CAPTURA para salvar o frame")
	}
	// Captura em ESCALA 2 (retina): o PNG sai nítido como o App real.
	var h *uitest.Harness
	g, err := nova(func(th *juigo.Theme) {
		if h != nil {
			h.Session().SetTheme(th)
			h.Session().InvalidateAll()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, th := range g.temas {
		if err := th.SetScale(2); err != nil {
			t.Fatal(err)
		}
	}
	h = uitest.NewLazy(t, func() juigo.Widget { return g.Raiz }, 2040, 1320)
	g.aplicar()
	h.Click(uitest.Text("Botões"))
	h.Layout()
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
