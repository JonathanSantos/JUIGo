package widget

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo/theme"
)

// navSessao monta um Navigator como raiz de uma Session headless e renderiza
// o primeiro frame (geometria válida para as navegações do teste).
func navSessao(t *testing.T) (*Navigator, *Session, *image.RGBA) {
	t.Helper()
	tema, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}
	nav := NewNavigator()
	s := NewSession(tema)
	s.Resize(image.Pt(300, 200))
	s.SetRoot(nav)
	buf := image.NewRGBA(image.Rect(0, 0, 300, 200))
	s.Render(buf)
	return nav, s, buf
}

// TestNavigatorPilha exercita a semântica da pilha sem aplicação: sem
// hooks.Schedule o Tween salta ao alvo e toda navegação é um corte seco —
// Push/Pop/PopToRoot/Replace trocam o topo na hora, e a raiz nunca sai.
func TestNavigatorPilha(t *testing.T) {
	nav, s, buf := navSessao(t)

	a, b, c := NewText("tela A"), NewText("tela B"), NewText("tela C")
	nav.Push(a)
	if nav.Top() != a || nav.Depth() != 1 || nav.CanPop() {
		t.Fatalf("após o primeiro Push: topo %v, profundidade %d", nav.Top(), nav.Depth())
	}
	s.Render(buf)

	nav.Push(b)
	if nav.Animating() {
		t.Fatal("sem aplicação a transição deveria saltar ao alvo")
	}
	if nav.Top() != b || !nav.CanPop() {
		t.Fatalf("após Push(b): topo %v", nav.Top())
	}
	if got := nav.Children(); len(got) != 1 || got[0] != b {
		t.Fatalf("Children deveria devolver só o topo; veio %v", got)
	}

	nav.Pop()
	if nav.Top() != a || nav.Depth() != 1 {
		t.Fatalf("após Pop: topo %v, profundidade %d", nav.Top(), nav.Depth())
	}
	nav.Pop() // a raiz não sai da pilha
	if nav.Depth() != 1 {
		t.Fatalf("Pop na raiz deveria ser inócuo; profundidade %d", nav.Depth())
	}

	nav.Push(b)
	nav.Push(c)
	nav.PopToRoot()
	if nav.Top() != a || nav.Depth() != 1 {
		t.Fatalf("após PopToRoot: topo %v, profundidade %d", nav.Top(), nav.Depth())
	}
}

// TestNavigatorReplacePreservaEntrada verifica que Replace troca a tela sem
// crescer a pilha e MANTÉM a transição de entrada do nível — o Pop seguinte
// reverte a navegação que criou o nível, não o Replace.
func TestNavigatorReplacePreservaEntrada(t *testing.T) {
	nav, _, _ := navSessao(t)

	a, b, c := NewText("A"), NewText("B"), NewText("C")
	nav.Push(a)
	nav.Push(b, TransitionSlideUp)
	nav.Replace(c)
	if nav.Depth() != 2 || nav.Top() != c {
		t.Fatalf("após Replace: topo %v, profundidade %d", nav.Top(), nav.Depth())
	}
	if entered := nav.stack[1].entered; entered != TransitionSlideUp {
		t.Fatalf("Replace deveria preservar a transição de entrada do nível; ficou %v", entered)
	}
	if reversa := nav.stack[1].entered.reversed(); reversa != TransitionSlideDown {
		t.Fatalf("a reversa de SlideUp deveria ser SlideDown; veio %v", reversa)
	}
}

// TestNavigatorTiraFocoDaTelaQueSai: o foco de teclado não pode ficar preso
// num widget que deixou a árvore.
func TestNavigatorTiraFocoDaTelaQueSai(t *testing.T) {
	nav, s, buf := navSessao(t)

	campo := NewInput("nome…")
	nav.Push(NewVBox(campo))
	s.Render(buf)
	s.Focus(campo)
	if s.Focused() != campo {
		t.Fatal("o campo deveria estar focado antes da navegação")
	}
	nav.Push(NewText("tela B"))
	if s.Focused() != nil {
		t.Fatalf("a navegação deveria tirar o foco da tela que saiu; ficou %v", s.Focused())
	}
}

// TestNavigatorTamanhoPreferido: o máximo entre TODAS as telas da pilha,
// para o tamanho não pular ao navegar.
func TestNavigatorTamanhoPreferido(t *testing.T) {
	nav, s, buf := navSessao(t)

	longa := NewText("uma tela bem mais comprida que a primeira")
	curta := NewText("curta")
	nav.Push(longa)
	nav.Push(curta)
	s.Render(buf)

	prefLonga := longa.PreferredSize()
	if curta.PreferredSize().X >= prefLonga.X {
		t.Fatal("pré-condição furada: a tela 'longa' deveria medir mais larga")
	}
	if got := nav.PreferredSize(); got.X != prefLonga.X {
		t.Fatalf("com a tela larga DORMENTE na pilha, PreferredSize deveria medi-la: %v ≠ %v", got, prefLonga)
	}
}
