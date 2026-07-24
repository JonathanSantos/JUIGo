package uitest_test

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// arvoreDemo monta uma Tree sobre um modelo de caminhos sintéticos:
// "a" → "a/1", "a/2"; "b" → "b/1"; "c" é folha.
func arvoreDemo() (*juigo.Tree[string, *juigo.Text], *juigo.State[string]) {
	filhos := map[string][]string{
		"a": {"a/1", "a/2"},
		"b": {"b/1"},
	}
	sel := juigo.NewState("")
	tree := juigo.NewTree(
		func() []string { return []string{"a", "b", "c"} },
		func(id string) []string { return filhos[id] },
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, id string) { t.SetText(id[strings.LastIndex(id, "/")+1:]) },
	).BindSelected(sel)
	return tree, sel
}

// visiveis devolve os textos das linhas atualmente vinculadas ao pool.
func visiveis(h *uitest.Harness) []string {
	var out []string
	for _, w := range h.FindAll(uitest.OfType[*juigo.Text]()) {
		out = append(out, w.(*juigo.Text).Text())
	}
	return out
}

// TestTreeExpansaoESelecao cobre chevron, seleção por clique e o modelo
// achatado.
func TestTreeExpansaoESelecao(t *testing.T) {
	tree, sel := arvoreDemo()
	h := uitest.New(t, tree, 240, 320)

	if got := visiveis(h); len(got) != 3 {
		t.Fatalf("recolhida, a árvore deveria mostrar 3 raízes; mostrou %v", got)
	}

	// Clique na coluna do chevron de "a" expande.
	th := h.Session().Theme()
	linhaA := image.Pt(th.Px(8), 4) // dentro da coluna de recuo da raiz
	h.ClickAt(linhaA)
	h.Layout()
	if got := visiveis(h); len(got) != 5 {
		t.Fatalf("com \"a\" expandida deveria haver 5 linhas; veio %v", got)
	}
	if !tree.Expanded("a") {
		t.Fatal("clicar no chevron deveria expandir \"a\"")
	}

	// Clique no conteúdo de "a/1" (linha 1) seleciona.
	h.Click(uitest.Text("1"))
	if sel.Get() != "a/1" {
		t.Fatalf("seleção deveria ser a/1; veio %q", sel.Get())
	}

	// Duplo clique numa linha interna alterna (recolhe "a").
	h.Advance(time.Second) // não herdar o clique anterior na janela do duplo
	h.DoubleClick(uitest.Text("a"))
	h.Layout()
	if tree.Expanded("a") {
		t.Fatal("duplo clique em \"a\" aberta deveria recolhê-la")
	}
	// O primeiro clique do duplo clique selecionou a própria linha.
	if sel.Get() != "a" {
		t.Fatalf("o duplo clique deveria ter selecionado a linha; veio %q", sel.Get())
	}
}

// TestTreeTeclado navega com setas: expandir/recolher/subir ao pai/ativar.
func TestTreeTeclado(t *testing.T) {
	tree, sel := arvoreDemo()
	ativado := ""
	tree.OnActivate(func(id string) { ativado = id })
	h := uitest.New(t, tree, 240, 320)

	h.Click(uitest.Text("b")) // foca a árvore e seleciona "b"
	if h.Focused() != tree || sel.Get() != "b" {
		t.Fatalf("clique deveria focar a árvore e selecionar b; foco %v, sel %q", h.Focused(), sel.Get())
	}
	h.Key(juigo.KeyRight) // expande b
	if !tree.Expanded("b") {
		t.Fatal("KeyRight deveria expandir b")
	}
	h.Key(juigo.KeyRight) // já aberto: desce ao primeiro filho
	if sel.Get() != "b/1" {
		t.Fatalf("KeyRight de novo deveria descer a b/1; veio %q", sel.Get())
	}
	h.Key(juigo.KeyEnter) // folha: ativa
	if ativado != "b/1" {
		t.Fatalf("Enter deveria ativar b/1; ativou %q", ativado)
	}
	h.Key(juigo.KeyLeft) // folha: sobe ao pai
	if sel.Get() != "b" {
		t.Fatalf("KeyLeft numa folha deveria subir ao pai; veio %q", sel.Get())
	}
	h.Key(juigo.KeyLeft) // aberto: recolhe
	if tree.Expanded("b") {
		t.Fatal("KeyLeft num nó aberto deveria recolhê-lo")
	}
	h.Key(juigo.KeyDown)
	if sel.Get() != "c" {
		t.Fatalf("KeyDown deveria descer a c; veio %q", sel.Get())
	}
}

// TestTreeVirtualizada: dez mil raízes, pool de meia dúzia de linhas.
func TestTreeVirtualizada(t *testing.T) {
	sel := juigo.NewState("")
	tree := juigo.NewTree(
		func() []string {
			out := make([]string, 10000)
			for i := range out {
				out[i] = fmt.Sprintf("nó %04d", i)
			}
			return out
		},
		func(string) []string { return nil },
		func() *juigo.Text { return juigo.NewText("") },
		func(tx *juigo.Text, id string) { tx.SetText(id) },
	).BindSelected(sel)
	h := uitest.New(t, juigo.NewScroll(tree), 240, 200)

	n := len(tree.Children())
	if n == 0 || n > 20 {
		t.Fatalf("o pool deveria ter só as linhas visíveis; tem %d", n)
	}
	h.Scroll(uitest.OfType[*juigo.Tree[string, *juigo.Text]](), -40)
	h.Layout()
	if got := visiveis(h); got[0] == "nó 0000" {
		t.Fatal("após rolar, a primeira linha vinculada deveria ter avançado")
	}
}

// TestIncrementalPaineisArvore é a rede de segurança das dirty regions para
// SplitPane e Tree: arraste do divisor, expansão e seleção — o frame
// incremental deve bater byte a byte com a repintura completa.
func TestIncrementalPaineisArvore(t *testing.T) {
	tree, _ := arvoreDemo()
	direita := juigo.NewVBox(
		juigo.NewText("Detalhe"),
		juigo.NewInput("filtro…"),
	).Pad(8)
	sp := juigo.NewSplitPane(juigo.NewScroll(tree), direita).Ratio(0.4).Min(40)
	h := uitest.New(t, sp, 400, 300)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			t.Fatalf("%s: render incremental divergiu do completo", passo)
		}
	}

	verifica("frame inicial")
	tree.Expand("a")
	verifica("expansão")
	h.Click(uitest.Text("1"))
	verifica("seleção")
	x := int(0.4 * float64(400)) // ~faixa do divisor
	h.Drag(image.Pt(x, 150), image.Pt(x+60, 150))
	verifica("arraste do divisor")
	h.Hover(uitest.Text("Detalhe"))
	verifica("hover fora da faixa")
}
