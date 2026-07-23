package widget

import (
	"image"
	"strings"
	"testing"

	"juigo/event"
	"juigo/render"
)

// TestInputRolagemHorizontal cobre o comportamento de campo clássico: o
// texto rola para manter o cursor visível quando passa da largura útil.
func TestInputRolagemHorizontal(t *testing.T) {
	th := newTestTheme(t)
	in := NewInput("")
	in.SetTheme(th)
	in.Layout(image.Rect(0, 0, 150, 36))
	in.HandleEvent(event.FocusEvent{Gained: true})
	buf := image.NewRGBA(image.Rect(0, 0, 300, 50))

	innerW := 150 - 2*th.PaddingPx()

	// Digita além da largura: rola e o cursor fica na borda direita útil.
	typeString(in, strings.Repeat("aç", 40))
	in.Draw(buf)
	if in.textW <= innerW {
		t.Fatal("pré-condição: o texto deveria ser maior que o campo")
	}
	if in.scrollX == 0 {
		t.Fatal("campo cheio deveria ter rolado (scrollX > 0)")
	}
	if vis := in.cursorX - in.scrollX; vis < 0 || vis > innerW {
		t.Fatalf("cursor fora da área visível: %d (útil 0..%d)", vis, innerW)
	}

	// Home volta ao início; End vai ao fim.
	in.HandleEvent(event.KeyEvent{Key: event.KeyHome})
	in.Draw(buf)
	if in.scrollX != 0 {
		t.Fatalf("após Home, scrollX = %d, esperado 0", in.scrollX)
	}
	in.HandleEvent(event.KeyEvent{Key: event.KeyEnd})
	in.Draw(buf)
	// Pode passar do limite do texto em até BorderPx: espaço reservado para
	// a própria linha do cursor após a última rune.
	if want := in.textW - innerW; in.scrollX < want || in.scrollX > want+th.BorderPx() {
		t.Fatalf("após End, scrollX = %d, esperado entre %d e %d", in.scrollX, want, want+th.BorderPx())
	}

	// Com rolagem, o clique mapeia para a rune correta (não a primeira).
	if idx := in.runeIndexAt(th.PaddingPx()); idx == 0 {
		t.Fatal("clique na borda esquerda de um campo rolado deveria cair no meio do texto")
	}
}

// TestInputNaoVazaDoCampo garante, pixel a pixel, que texto, seleção e
// cursor nunca pintam fora dos bounds do campo — o bug original.
func TestInputNaoVazaDoCampo(t *testing.T) {
	th := newTestTheme(t)
	in := NewInput("")
	in.SetTheme(th)
	const x0, y0, x1, y1 = 40, 10, 190, 46
	in.Layout(image.Rect(x0, y0, x1, y1))
	in.HandleEvent(event.FocusEvent{Gained: true})
	typeString(in, strings.Repeat("Wçã", 30))
	// Seleção ativa para também cobrir o retângulo de seleção.
	in.HandleEvent(event.KeyEvent{Key: event.KeyLeft, Mods: event.ModShift})
	in.HandleEvent(event.KeyEvent{Key: event.KeyLeft, Mods: event.ModShift})

	buf := image.NewRGBA(image.Rect(0, 0, 240, 60))
	bg := th.Background
	render.FillRect(buf, buf.Bounds(), bg)
	in.Draw(buf)

	for y := 0; y < 60; y++ {
		for x := 0; x < 240; x++ {
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				continue // dentro do campo, pode pintar
			}
			i := buf.PixOffset(x, y)
			if buf.Pix[i] != bg.R || buf.Pix[i+1] != bg.G || buf.Pix[i+2] != bg.B {
				t.Fatalf("pixel pintado fora do campo em (%d,%d)", x, y)
			}
		}
	}
}

// TestInputDrawSemAlocacao garante que o caminho de desenho do Input — com
// recorte e rolagem — continua sem alocar por frame.
func TestInputDrawSemAlocacao(t *testing.T) {
	th := newTestTheme(t)
	in := NewInput("")
	in.SetTheme(th)
	in.Layout(image.Rect(0, 0, 150, 36))
	in.HandleEvent(event.FocusEvent{Gained: true})
	typeString(in, strings.Repeat("aç", 40))
	buf := image.NewRGBA(image.Rect(0, 0, 300, 50))
	in.Draw(buf) // aquece cache de glyphs

	allocs := testing.AllocsPerRun(100, func() {
		in.Draw(buf)
	})
	if allocs != 0 {
		t.Fatalf("Input.Draw alocou %.1f vezes por chamada, esperado 0", allocs)
	}
}
