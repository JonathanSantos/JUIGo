package uitest_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestScrollToEndEStickToEnd: ScrollToEnd alcança conteúdo acrescentado na
// mesma rodada; StickToEnd acompanha o crescimento até o usuário subir.
func TestScrollToEndEStickToEnd(t *testing.T) {
	box := juigo.NewVBox()
	for i := range 30 {
		box.Add(juigo.NewText(fmt.Sprintf("linha %d", i)))
	}
	scroll := juigo.NewScroll(box)
	h := uitest.New(t, scroll, 300, 120)

	box.Add(juigo.NewText("última"))
	scroll.ScrollToEnd()
	h.Layout()
	fim := box.PreferredSize().Y - 120
	if scroll.Offset() != fim {
		t.Fatalf("ScrollToEnd: offset = %d, esperado %d (fim com a linha nova)", scroll.Offset(), fim)
	}

	scroll.StickToEnd(true)
	box.Add(juigo.NewText("mais uma"))
	h.Layout()
	if want := box.PreferredSize().Y - 120; scroll.Offset() != want {
		t.Fatalf("StickToEnd deveria acompanhar o conteúdo: offset = %d, esperado %d", scroll.Offset(), want)
	}

	// O usuário rola para cima: conteúdo novo não o arrasta de volta.
	h.Scroll(uitest.OfType[*juigo.Scroll](), 3)
	parado := scroll.Offset()
	box.Add(juigo.NewText("chegou enquanto lia"))
	h.Layout()
	if scroll.Offset() != parado {
		t.Fatalf("rolado para cima, o Scroll deveria ficar em %d; foi para %d", parado, scroll.Offset())
	}

	// De volta ao fim, volta a acompanhar.
	scroll.ScrollToEnd()
	h.Layout()
	box.Add(juigo.NewText("e mais"))
	h.Layout()
	if want := box.PreferredSize().Y - 120; scroll.Offset() != want {
		t.Fatalf("de volta ao fim, deveria acompanhar: offset = %d, esperado %d", scroll.Offset(), want)
	}
}

// TestListaDeTextoComSelecaoPropria: NewTextList + OnChange dispensam a
// fábrica de linhas, o State de seleção e o Watch à parte.
func TestListaDeTextoComSelecaoPropria(t *testing.T) {
	nomes := []string{"Ana", "Bia", "Caio"}
	var escolhas []int
	lista := juigo.NewTextList(len(nomes), func(i int) string { return nomes[i] }).
		OnChange(func(i int) { escolhas = append(escolhas, i) })
	h := uitest.New(t, juigo.NewScroll(lista), 300, 200)

	if lista.Selected() != -1 {
		t.Fatalf("sem seleção inicial; Selected() = %d", lista.Selected())
	}
	h.Click(uitest.Text("Bia"))
	if lista.Selected() != 1 || len(escolhas) != 1 || escolhas[0] != 1 {
		t.Fatalf("clique: Selected() = %d, OnChange = %v", lista.Selected(), escolhas)
	}
	// Seleção programática não dispara OnChange.
	lista.Select(2)
	if lista.Selected() != 2 || len(escolhas) != 1 {
		t.Fatalf("Select: Selected() = %d, OnChange = %v (não deveria disparar)", lista.Selected(), escolhas)
	}
}

// TestTabelaEArvoreComOnChange cobre a mesma API de seleção nos irmãos.
func TestTabelaEArvoreComOnChange(t *testing.T) {
	linha := -1
	tab := juigo.NewTable([]string{"Nome"}, 3, func(r, c int) string { return fmt.Sprintf("item %d", r) }).
		OnChange(func(i int) { linha = i })
	no := ""
	arvore := juigo.NewTextTree(
		func() []string { return []string{"a", "b"} },
		func(string) []string { return nil },
		func(id string) string { return "nó " + id },
	).OnChange(func(id string) { no = id })
	h := uitest.New(t, juigo.NewVBox(juigo.NewScroll(tab).Grow(1), juigo.NewScroll(arvore).Grow(1)), 300, 300)

	h.ClickAt(tab.RowRect(1).Min.Add(tab.RowRect(1).Size().Div(2)))
	if linha != 1 || tab.Selected() != 1 {
		t.Fatalf("Table: OnChange = %d, Selected() = %d; esperado 1", linha, tab.Selected())
	}
	h.Click(uitest.Text("nó b"))
	if no != "b" || arvore.Selected() != "b" {
		t.Fatalf("Tree: OnChange = %q, Selected() = %q; esperado b", no, arvore.Selected())
	}
}

// TestBindChoiceTipado: Dropdown ligado a um State[int], sem strconv na
// aplicação.
func TestBindChoiceTipado(t *testing.T) {
	tamanho := juigo.NewState(16)
	d := juigo.BindChoice(juigo.NewDropdown(), []int{14, 16, 18}, strconv.Itoa, tamanho)
	h := uitest.New(t, juigo.NewVBox(d), 300, 200)

	if d.Value() != "16" {
		t.Fatalf("valor inicial exibido = %q, esperado 16", d.Value())
	}
	tamanho.Set(18)
	if d.Value() != "18" {
		t.Fatalf("Set externo: exibido %q, esperado 18", d.Value())
	}
	h.Click(uitest.OfType[*juigo.Dropdown]())
	h.Key(juigo.KeyHome)
	h.Key(juigo.KeyEnter)
	if tamanho.Get() != 14 {
		t.Fatalf("escolha do usuário: State = %d, esperado 14", tamanho.Get())
	}
}

// TestModalBindOpen: a visibilidade do diálogo como estado da aplicação.
func TestModalBindOpen(t *testing.T) {
	aberto := juigo.NewState(false)
	modal := juigo.NewModal(juigo.NewText("conteúdo")).BindOpen(aberto)
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")), 400, 300)

	aberto.Set(true)
	if !modal.Shown() || h.Session().Overlay() == nil {
		t.Fatal("Set(true) deveria abrir o modal")
	}
	h.Key(juigo.KeyEscape)
	if modal.Shown() || aberto.Get() {
		t.Fatalf("Escape deveria fechar e refletir no State; shown=%v state=%v", modal.Shown(), aberto.Get())
	}
	modal.Show()
	if !aberto.Get() {
		t.Fatal("Show deveria refletir no State")
	}
	aberto.Set(false)
	if modal.Shown() {
		t.Fatal("Set(false) deveria fechar o modal")
	}
}

// TestOnFocusOnBlurEmQualquerFocavel: os callbacks de foco valem para todo
// widget focável, não só para os campos de texto.
func TestOnFocusOnBlurEmQualquerFocavel(t *testing.T) {
	var log []string
	btn := juigo.NewButton("OK", nil).
		OnFocus(func() { log = append(log, "btn+") }).
		OnBlur(func() { log = append(log, "btn-") })
	check := juigo.NewCheckbox("C").OnFocus(func() { log = append(log, "check+") })
	h := uitest.New(t, juigo.NewVBox(btn, check), 300, 200)

	h.Key(juigo.KeyTab)
	h.Key(juigo.KeyTab)
	want := []string{"btn+", "btn-", "check+"}
	if fmt.Sprint(log) != fmt.Sprint(want) {
		t.Fatalf("callbacks de foco = %v, esperado %v", log, want)
	}
}
