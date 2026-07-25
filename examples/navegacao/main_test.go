package main

import (
	"os"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestFluxoDeNavegacao percorre o fluxo inteiro com o relógio virtual
// atravessando as animações: Perfil → Preferências → Concluir (PopToRoot),
// estado preservado ao voltar e Ajuda subindo e descendo.
func TestFluxoDeNavegacao(t *testing.T) {
	nav := ui()
	h := uitest.New(t, nav, 480, 360)

	h.Click(uitest.Text("Ver perfil →"))
	h.Advance(time.Second)
	h.Click(uitest.Placeholder("seu nome…"))
	h.Type("Ada")
	h.Click(uitest.Text("Preferências →"))
	h.Advance(time.Second)

	h.Click(uitest.Text("Concluir"))
	h.Advance(time.Second)
	if nav.Depth() != 1 {
		t.Fatalf("Concluir deveria voltar à raiz numa transição só; profundidade %d", nav.Depth())
	}

	// Telas dormentes preservam o estado: o campo continua preenchido.
	h.Click(uitest.Text("Ver perfil →"))
	h.Advance(time.Second)
	campo := h.Find(uitest.Placeholder("seu nome…")).(*juigo.Input)
	if campo.Text() != "Ada" {
		t.Fatalf("o campo do Perfil deveria preservar o texto; veio %q", campo.Text())
	}
	h.Click(uitest.Text("← Voltar"))
	h.Advance(time.Second)

	// Ajuda sobe (SlideUp) e Fechar a desce (Pop reverte a entrada).
	h.Click(uitest.Text("Ajuda"))
	h.Advance(time.Second)
	h.Click(uitest.Text("Fechar"))
	if !nav.Animating() {
		t.Fatal("Fechar deveria animar a descida da Ajuda")
	}
	h.Advance(time.Second)
	if nav.Depth() != 1 || nav.Animating() {
		t.Fatalf("após Fechar: profundidade %d, animando %v", nav.Depth(), nav.Animating())
	}
}

// TestCapturaVisual salva docs/navegacao.png em PLENO DESLIZE quando a
// variável NAVEGACAO_CAPTURA aponta um caminho — só para inspeção manual:
//
//	NAVEGACAO_CAPTURA=docs/navegacao.png go test ./examples/navegacao
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("NAVEGACAO_CAPTURA")
	if caminho == "" {
		t.Skip("defina NAVEGACAO_CAPTURA para salvar o frame")
	}
	// Captura em ESCALA 2 (retina): o PNG sai nítido como o App real.
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatal(err)
	}
	if err := th.SetScale(2); err != nil {
		t.Fatal(err)
	}
	nav := ui()
	h := uitest.NewWithTheme(t, nav, th, 1280, 800)
	h.Click(uitest.Text("Ver perfil →"))
	h.Advance(144 * time.Millisecond) // ~metade dos 280ms padrão do tema
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
