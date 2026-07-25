package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestFluxoDoChat: enviar publica o balão, a resposta chega palavra a
// palavra pelo relógio virtual, Ctrl+N abre conversa nova e trocar de
// conversa restaura o histórico.
func TestFluxoDoChat(t *testing.T) {
	v := nova()
	h := uitest.NewLazy(t, func() juigo.Widget { return v.Raiz }, 900, 560)

	h.Click(uitest.Placeholder("Escreva para a Tinta…"))
	h.Type("como organizo a tela?")
	h.Key(juigo.KeyEnter)

	if h.Find(uitest.Text("como organizo a tela?")) == nil {
		t.Fatal("a mensagem enviada deveria virar um balão")
	}
	// No meio do fluxo: resposta parcial.
	h.Advance(200 * time.Millisecond)
	if !v.digitando {
		t.Fatal("a Tinta deveria estar digitando")
	}
	parcial := v.fluxo.Get()
	if parcial == "" || strings.HasSuffix(parcial, ".") {
		t.Fatalf("a resposta deveria estar no meio; %q", parcial)
	}
	// Até o fim: o texto integral fica no histórico.
	h.Advance(5 * time.Second)
	if v.digitando {
		t.Fatal("o fluxo deveria ter terminado")
	}
	c := v.atualConversa()
	if got := c.msgs[len(c.msgs)-1].texto; got != respostas[0] {
		t.Fatalf("a resposta gravada deveria ser a carta inteira; veio %q", got)
	}

	// Ctrl+N abre conversa vazia (o atalho veio do MenuBar).
	h.Key(juigo.KeyN, juigo.ModControl)
	h.Layout()
	if len(v.atualConversa().msgs) != 0 {
		t.Fatal("a conversa nova deveria nascer vazia")
	}
	if h.Find(uitest.Text("como organizo a tela?")) != nil {
		t.Fatal("o feed deveria estar limpo na conversa nova")
	}

	// Voltar à primeira restaura o histórico.
	h.Click(uitest.Text("Primeiras ideias"))
	h.Layout()
	if h.Find(uitest.Text("como organizo a tela?")) == nil {
		t.Fatal("trocar de volta deveria restaurar o balão")
	}
}

// TestPaletaDoChat: os comandos do menu aparecem na paleta Ctrl/Cmd+K.
func TestPaletaDoChat(t *testing.T) {
	v := nova()
	h := uitest.NewLazy(t, func() juigo.Widget { return v.Raiz }, 900, 560)

	h.Key(juigo.KeyK, juigo.ModControl)
	if h.Session().Overlay() == nil {
		t.Fatal("Ctrl+K deveria abrir a paleta")
	}
	h.Type("nova")
	h.Key(juigo.KeyEnter)
	h.Layout()
	if len(v.conversas) != 2 {
		t.Fatalf("a paleta deveria executar Nova conversa; há %d conversas", len(v.conversas))
	}
}

// TestCapturaVisual salva docs/chat.png com a conversa em cena quando
// CHAT_CAPTURA aponta um caminho:
//
//	CHAT_CAPTURA=docs/chat.png go test ./examples/chat -run TestCapturaVisual
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("CHAT_CAPTURA")
	if caminho == "" {
		t.Skip("defina CHAT_CAPTURA para salvar o frame")
	}
	th, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	v := nova()
	h := uitest.NewLazy(t, func() juigo.Widget { return v.Raiz }, 900, 560)
	h.Session().SetTheme(th)

	h.Click(uitest.Placeholder("Escreva para a Tinta…"))
	h.Type("Como divido esta tela sem exagerar nos cartões?")
	h.Key(juigo.KeyEnter)
	h.Advance(10 * time.Second)
	h.Click(uitest.Placeholder("Escreva para a Tinta…"))
	h.Type("E quando entra a terracota?")
	h.Key(juigo.KeyEnter)
	h.Advance(400 * time.Millisecond) // em pleno "digitando"
	h.Layout()
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
