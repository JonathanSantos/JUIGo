package uitest_test

import (
	"fmt"
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestComandoGlobalPorAtalho: o atalho dispara com o modificador de
// comando (Ctrl≡Cmd), não dispara na digitação comum e respeita o widget
// focado que consome a tecla.
func TestComandoGlobalPorAtalho(t *testing.T) {
	salvos := 0
	campo := juigo.NewInput("nome…")
	h := uitest.New(t, juigo.NewVBox(campo), 300, 200)
	h.Session().AddCommand(juigo.Command{
		Title: "Salvar", Key: juigo.LetterKey('s'), Mods: juigo.ModControl,
		Action: func() { salvos++ },
	})

	h.Click(uitest.Placeholder("nome…"))
	h.Type("s") // digitação comum: não é atalho
	h.Key(juigo.KeyS)
	if salvos != 0 {
		t.Fatalf("tecla sem modificador não deveria acionar; acionou %d", salvos)
	}
	h.Key(juigo.KeyS, juigo.ModControl)
	if salvos != 1 {
		t.Fatalf("Ctrl+S deveria acionar; contou %d", salvos)
	}
	h.Key(juigo.KeyS, juigo.ModSuper) // Cmd também (equivalência)
	if salvos != 2 {
		t.Fatalf("Cmd+S deveria acionar; contou %d", salvos)
	}
	if campo.Text() != "s" {
		t.Fatalf("o campo deveria ter só o texto digitado; tem %q", campo.Text())
	}
}

// TestPaletaDeComandos: Ctrl+K abre, digitar filtra, Enter executa o
// selecionado com a paleta já fechada; Escape fecha sem executar.
func TestPaletaDeComandos(t *testing.T) {
	var executado string
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("app")), 480, 400)
	for _, nome := range []string{"Abrir arquivo", "Salvar como", "Sair"} {
		nome := nome
		h.Session().AddCommand(juigo.Command{Title: nome, Action: func() { executado = nome }})
	}

	h.Key(juigo.KeyK, juigo.ModControl)
	if h.Session().Overlay() == nil {
		t.Fatal("Ctrl+K deveria abrir a paleta")
	}
	h.Type("salvar")
	h.Key(juigo.KeyEnter)
	if executado != "Salvar como" {
		t.Fatalf("Enter deveria executar o filtrado; executou %q", executado)
	}
	if h.Session().Overlay() != nil {
		t.Fatal("a paleta deveria fechar antes da ação")
	}

	// Navegação por setas + Escape sem executar.
	executado = ""
	h.Key(juigo.KeyK, juigo.ModSuper)
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEscape)
	if executado != "" || h.Session().Overlay() != nil {
		t.Fatalf("Escape deveria fechar sem executar; executou %q", executado)
	}
}

// TestMenuBar: clique abre o menu, teclado navega e executa (com o menu já
// fechado), o atalho do item se registra sozinho e o hover troca de menu.
func TestMenuBar(t *testing.T) {
	var executado string
	bar := juigo.NewMenuBar().
		Menu("Arquivo",
			juigo.Command{Title: "Novo", Key: juigo.LetterKey('n'), Mods: juigo.ModControl,
				Action: func() { executado = "Novo" }},
			juigo.MenuSeparator(),
			juigo.Command{Title: "Salvar", Action: func() { executado = "Salvar" }},
		).
		Menu("Ajuda",
			juigo.Command{Title: "Sobre", Action: func() { executado = "Sobre" }},
		)
	h := uitest.New(t, juigo.NewVBox(bar, juigo.NewText("conteúdo")), 480, 360)
	th := h.Session().Theme()

	// Atalho registrado no mount, sem abrir menu nenhum.
	h.Key(juigo.KeyN, juigo.ModControl)
	if executado != "Novo" {
		t.Fatalf("Ctrl+N deveria acionar o item do menu; veio %q", executado)
	}

	// Clique no título abre; Baixo pula ao primeiro item, Baixo de novo
	// PULA o separador e Enter executa com o menu fechado.
	executado = ""
	tituloArquivo := image.Pt(th.Px(4)+(th.MeasureString("Arquivo")+2*th.PaddingPx())/2, th.LineHeight()/2)
	h.ClickAt(tituloArquivo)
	if h.Session().Overlay() == nil {
		t.Fatal("clicar no título deveria abrir o menu")
	}
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEnter)
	if executado != "Salvar" {
		t.Fatalf("o segundo item acionável é Salvar; veio %q", executado)
	}
	if h.Session().Overlay() != nil {
		t.Fatal("executar deveria fechar o menu")
	}

	// Hover troca de menu com um aberto: Direita (teclado) também.
	executado = ""
	h.ClickAt(tituloArquivo)
	h.Key(juigo.KeyRight) // vai ao menu Ajuda
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEnter)
	if executado != "Sobre" {
		t.Fatalf("Direita deveria trocar para Ajuda; executou %q", executado)
	}
}

// TestModalRolavel: conteúdo mais alto que a janela ganha rolagem — o
// botão abaixo da dobra fica alcançável em vez de virar área morta.
func TestModalRolavel(t *testing.T) {
	cliques := 0
	linhas := make([]juigo.Widget, 0, 24)
	for i := 0; i < 24; i++ {
		linhas = append(linhas, juigo.NewText(fmt.Sprintf("linha %02d", i)))
	}
	linhas = append(linhas, juigo.NewButton("Confirmar", func() { cliques++ }))
	m := juigo.NewModal(juigo.NewVBox(linhas...))
	h := uitest.New(t, juigo.NewVBox(juigo.NewButton("Abrir", m.Show)), 360, 240)

	h.Click(uitest.Text("Abrir"))
	// Rola o miolo do modal até o fim (âncora no PRÓPRIO modal: o alvo não
	// sai da janela) e clica no botão.
	for i := 0; i < 40; i++ {
		h.Scroll(uitest.OfType[*juigo.Modal](), -40)
	}
	h.Layout()
	btn := h.Find(uitest.Text("Confirmar"))
	if btn == nil {
		t.Fatal("o botão deveria existir na árvore")
	}
	c := btn.Bounds()
	if c.Min.Y >= 240 {
		t.Fatalf("após rolar, o botão deveria estar visível; está em %v", c)
	}
	h.ClickAt(image.Pt(c.Min.X+c.Dx()/2, c.Min.Y+c.Dy()/2))
	if cliques != 1 {
		t.Fatalf("o botão abaixo da dobra deveria responder; cliques=%d", cliques)
	}
}
