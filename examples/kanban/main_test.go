package main

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// cartaoPorTitulo encontra o widget do cartão com o título dado.
func cartaoPorTitulo(t *testing.T, h *uitest.Harness, titulo string) *cartaoView {
	t.Helper()
	for _, w := range h.FindAll(uitest.OfType[*cartaoView]()) {
		if k := w.(*cartaoView); k.c.titulo == titulo {
			return k
		}
	}
	t.Fatalf("cartão %q não encontrado", titulo)
	return nil
}

// centro devolve o centro dos bounds de um widget.
func centro(w juigo.Widget) image.Point {
	return w.Bounds().Min.Add(w.Bounds().Size().Div(2))
}

// TestArrastarCartaoEntreColunas: o arrasto move o cartão no modelo e a UI
// reconstruída o mostra na coluna de destino; clique simples não move.
func TestArrastarCartaoEntreColunas(t *testing.T) {
	q := exemplo()
	v := nova(q)
	h := uitest.New(t, v.Raiz, 640, 360)

	// Clique simples (sem passar do limiar) não inicia arrasto nem move.
	alvo := cartaoPorTitulo(t, h, "Revisar o texto")
	h.ClickAt(centro(alvo))
	if h.Session().Dragging() || len(q.colunas[0]) != 3 {
		t.Fatal("clique simples não deveria arrastar")
	}

	// Arrasta "Revisar o texto" (coluna 0) para a coluna "Feito" (2).
	h.Drag(centro(alvo), centro(v.colunas[2]))
	if len(q.colunas[0]) != 2 || len(q.colunas[2]) != 2 {
		t.Fatalf("o cartão deveria ir para Feito; colunas=%d/%d/%d",
			len(q.colunas[0]), len(q.colunas[1]), len(q.colunas[2]))
	}
	if q.colunas[2][1].titulo != "Revisar o texto" {
		t.Fatalf("o cartão deveria entrar no fim de Feito; veio %q", q.colunas[2][1].titulo)
	}

	// A UI reconstruída posiciona o cartão dentro da coluna de destino.
	movido := cartaoPorTitulo(t, h, "Revisar o texto")
	if !movido.Bounds().In(v.colunas[2].Bounds()) {
		t.Fatal("o widget do cartão deveria estar dentro da coluna Feito")
	}

	// Soltar fora de qualquer coluna não move nada.
	outro := cartaoPorTitulo(t, h, "Ajustar o layout")
	h.Drag(centro(outro), image.Pt(636, 4)) // canto: fora das colunas (Pad 12)
	if len(q.colunas[1]) != 1 {
		t.Fatal("soltar fora de coluna não deveria mover o cartão")
	}
	if h.Session().Dragging() {
		t.Fatal("o arrasto deveria terminar mesmo sem alvo")
	}
}
