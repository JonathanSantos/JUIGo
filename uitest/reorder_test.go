package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// move desloca o item de `de` para o índice final `para` — o contrato do
// OnReorder.
func move(s []string, de, para int) []string {
	item := s[de]
	s = append(s[:de], s[de+1:]...)
	return append(s[:para], append([]string{item}, s[para:]...)...)
}

// TestReordenarLista: arrastar uma linha da List chama OnReorder com o
// índice final, mostra o indicador de inserção na fronteira, e soltar no
// mesmo lugar (ou clicar) não move nada.
func TestReordenarLista(t *testing.T) {
	itens := []string{"um", "dois", "três", "quatro"}
	chamadas := 0
	var lista *juigo.List[*juigo.Text]
	lista = juigo.NewList(len(itens),
		func() *juigo.Text { return juigo.NewText("") },
		func(txt *juigo.Text, i int) { txt.SetText(itens[i]) },
	).OnReorder(
		func(i int) string { return itens[i] },
		func(de, para int) {
			chamadas++
			itens = move(itens, de, para)
			lista.Refresh()
		},
	)
	h := uitest.New(t, juigo.NewVBox(lista), 260, 220)
	th := h.Session().Theme()
	h.Layout()
	b := lista.Bounds()
	rowH := lista.PreferredSize().Y / len(itens)
	centroLinha := func(i int) image.Point {
		return image.Pt(b.Min.X+b.Dx()/2, b.Min.Y+i*rowH+rowH/2)
	}

	// Clique simples seleciona/não move.
	h.ClickAt(centroLinha(0))
	if chamadas != 0 {
		t.Fatal("clique simples não deveria reordenar")
	}

	// Passo a passo, soltando na METADE DE BAIXO da linha 2 (a fronteira
	// mais próxima fica depois dela): o indicador aparece na fronteira 3.
	s := h.Session()
	baixo := image.Pt(b.Min.X+b.Dx()/2, b.Min.Y+2*rowH+3*rowH/4)
	h.MoveTo(centroLinha(0))
	s.PointerDown(centroLinha(0), juigo.MouseButtonLeft)
	s.PointerMove(baixo)
	if !s.Dragging() {
		t.Fatal("segurar e mover deveria iniciar a reordenação")
	}
	img := h.Screenshot()
	fronteira := b.Min.Y + 3*rowH
	if got := img.RGBAAt(b.Min.X+b.Dx()/2, fronteira-1); got != th.Accent {
		t.Fatalf("o indicador de inserção deveria estar na fronteira; veio %v", got)
	}
	s.PointerUp(baixo, juigo.MouseButtonLeft)
	if chamadas != 1 || itens[2] != "um" {
		t.Fatalf("soltar deveria mover 'um' para o índice 2; itens=%v chamadas=%d", itens, chamadas)
	}

	// Soltar na própria posição não chama o OnReorder.
	h.Drag(centroLinha(1), image.Pt(b.Min.X+b.Dx()/2, b.Min.Y+rowH+rowH/2+rowH/4))
	if chamadas != 1 {
		t.Fatalf("soltar no mesmo lugar não deveria reordenar; chamadas=%d", chamadas)
	}
	if s.Dragging() {
		t.Fatal("o arrasto deveria ter terminado")
	}
}

// TestReordenarTabela: o OnReorder da Table usa a primeira célula como
// rótulo do fantasma e convive com a seleção por clique.
func TestReordenarTabela(t *testing.T) {
	linhas := []string{"Ana", "Bruno", "Carla"}
	selecionada := juigo.NewState(-1)
	chamadas := 0
	var tabela *juigo.Table
	tabela = juigo.NewTable([]string{"Nome"}, len(linhas), func(r, c int) string {
		return linhas[r]
	}).BindSelected(selecionada).OnReorder(func(de, para int) {
		chamadas++
		linhas = move(linhas, de, para)
		tabela.Refresh()
	})
	h := uitest.New(t, juigo.NewVBox(tabela), 260, 200)
	th := h.Session().Theme()
	h.Layout()

	centro := func(i int) image.Point {
		r := tabela.RowRect(i)
		return r.Min.Add(r.Size().Div(2))
	}

	// Clique simples só seleciona.
	h.ClickAt(centro(1))
	if selecionada.Get() != 1 || chamadas != 0 {
		t.Fatalf("clique deveria selecionar sem mover; sel=%d chamadas=%d", selecionada.Get(), chamadas)
	}

	// Arrasta Ana para o fim, soltando na METADE DE BAIXO da última linha
	// (a fronteira mais próxima fica depois dela); o fantasma usa a
	// primeira célula.
	s := h.Session()
	ultima := tabela.RowRect(2)
	baixo := image.Pt(ultima.Min.X+ultima.Dx()/2, ultima.Min.Y+3*ultima.Dy()/4)
	h.MoveTo(centro(0))
	s.PointerDown(centro(0), juigo.MouseButtonLeft)
	s.PointerMove(baixo)
	if !s.Dragging() {
		t.Fatal("segurar e mover deveria iniciar a reordenação da tabela")
	}
	// O fantasma mostra "Ana" (cores do tooltip em cena).
	img := h.Screenshot()
	pad := th.PaddingPx()
	fantasma := baixo.Add(image.Pt(pad, pad))
	if got := img.RGBAAt(fantasma.X+2, fantasma.Y+1); got != th.TooltipBackground {
		t.Fatalf("o fantasma da tabela deveria estar em cena; veio %v", got)
	}
	s.PointerUp(baixo, juigo.MouseButtonLeft)
	if chamadas != 1 || linhas[2] != "Ana" {
		t.Fatalf("soltar deveria mover Ana para o fim; linhas=%v", linhas)
	}
}
