package main

// Cenas por componente para o site (docs/index.html). Cada cena é ao mesmo
// tempo a captura e o exemplo de uso: o gerador renderiza a função e copia
// o CORPO dela, do próprio fonte, para docs/site/cenas.js — o código que o
// site mostra é exatamente o código que produziu a imagem ao lado.

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/chart"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/syntax"
)

// cena descreve um componente do site: identificador (nome do PNG), título,
// grupo da navegação, uma frase de descrição, tamanho lógico e a função.
type cena struct {
	ID, Titulo, Grupo, Descricao string
	W, H                         int
	monta                        func() juigo.Widget
}

var cenas = []cena{
	{"tipografia", "Text e Label", "Estilo",
		"Papéis tipográficos do tema e o parágrafo que quebra sozinho.", 460, 166, cenaTipografia},
	{"botoes", "Button", "Controles",
		"Três hierarquias, estado desabilitado e carregando.", 460, 76, cenaBotoes},
	{"entradas", "Input e TextArea", "Controles",
		"Placeholder, texto, erro vinculado a um State e multilinha com soft wrap.", 460, 190, cenaEntradas},
	{"escolhas", "Checkbox, Radio e Dropdown", "Controles",
		"Escolhas ligadas a States com BindChecked e BindValue.", 460, 40, cenaEscolhas},
	{"slider", "Slider e ProgressBar", "Controles",
		"Dois widgets no mesmo State: mover um move o outro.", 460, 80, cenaSlider},
	{"calendario", "Calendar", "Controles",
		"Seleção de data com BindValue em time.Time.", 300, 250, cenaCalendario},
	{"editor", "CodeEditor", "Controles",
		"Fonte mono, gutter, highlight incremental e undo coalescido.", 460, 150, cenaEditor},
	{"lista", "List", "Coleções",
		"Lista virtualizada: milhares de linhas com meia dúzia de widgets.", 300, 170, cenaLista},
	{"tabela", "Table", "Coleções",
		"Cabeçalho fixo, seleção como State e reordenação por arrasto.", 460, 110, cenaTabela},
	{"arvore", "Tree", "Coleções",
		"Árvore virtualizada; o modelo são dois callbacks sobre um ID seu.", 300, 130, cenaArvore},
	{"layout", "VBox, HBox e Grid", "Estrutura",
		"Layout por composição: Grow, Spacer, Gap e grade alinhada.", 460, 120, cenaLayout},
	{"superficies", "Card e Divider", "Estrutura",
		"A superfície que agrupa e o fio que separa.", 460, 136, cenaSuperficies},
	{"abas", "Tabs", "Estrutura",
		"Abas: só a ativa participa de eventos e foco.", 460, 90, cenaAbas},
	{"split", "SplitPane", "Estrutura",
		"Painéis redimensionáveis com divisor arrastável.", 460, 110, cenaSplit},
	{"formulario", "quick.Form", "Formulários",
		"Campos com handles tipados, validação e erros por campo.", 460, 268, cenaFormulario},
	{"graficos", "chart", "Dados",
		"Linha, barras e sparkline, nas cores do tema.", 460, 300, cenaGraficos},
}

func cenaTipografia() juigo.Widget {
	return juigo.NewVBox(
		juigo.NewText("Relatório mensal").Title(),
		juigo.NewText("Resumo da equipe").Subtitle(),
		juigo.NewText("O corpo usa a fonte regular do tema."),
		juigo.NewText("Atualizado há 2 minutos").Caption(),
		juigo.NewLabel("Um Label quebra por palavras na largura disponível e "+
			"aceita os mesmos papéis tipográficos do Text."),
	).Gap(6)
}

func cenaBotoes() juigo.Widget {
	salvando := juigo.NewButton("Salvando", nil)
	salvando.SetLoading(true)
	return juigo.NewVBox(
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.NewButton("Cancelar", nil).Ghost(),
			juigo.NewButton("Rascunho", nil).Secondary(),
			juigo.NewButton("Publicar", publicar),
		).Gap(8),
		juigo.NewHBox(
			salvando,
			juigo.NewButton("Inativo", nil).BindDisabled(juigo.NewState(true)),
			juigo.NewSpacer(),
		).Gap(8),
	).Gap(12)
}

func cenaEntradas() juigo.Widget {
	nome := juigo.NewState("Ana Lima")
	invalido := juigo.NewState(true)
	return juigo.NewVBox(
		juigo.NewInput("Nome").BindValue(nome),
		juigo.NewInput("E-mail").BindInvalid(invalido),
		juigo.NewText("Informe um e-mail válido").Danger(),
		juigo.NewSized(juigo.NewTextArea("Notas…"), 0, 70),
	).Gap(8)
}

func cenaEscolhas() juigo.Widget {
	avisos := juigo.NewState(true)
	plano := juigo.NewState("pro")
	return juigo.NewHBox(
		juigo.NewCheckbox("Avisos").BindChecked(avisos),
		juigo.NewSpacer(),
		juigo.NewRadio("Grátis", "free").BindValue(plano),
		juigo.NewRadio("Pro", "pro").BindValue(plano),
		juigo.NewSpacer(),
		juigo.NewDropdown("Baixa", "Média", "Alta"),
	).Gap(8)
}

func cenaSlider() juigo.Widget {
	volume := juigo.NewState(0.62)
	return juigo.NewVBox(
		juigo.NewSlider(0, 1).Step(0.01).BindValue(volume),
		juigo.NewProgressBar(0, 1).BindValue(volume),
		juigo.NewText("").BindText(juigo.Map(volume, func(v float64) string {
			return fmt.Sprintf("Volume: %.0f%%", v*100)
		})).Caption(),
	).Gap(10)
}

func cenaCalendario() juigo.Widget {
	data := juigo.NewState(time.Date(2026, 7, 24, 0, 0, 0, 0, time.Local))
	return juigo.NewVBox(
		juigo.NewCalendar().BindValue(data),
		juigo.NewText("").BindText(juigo.Map(data, func(t time.Time) string {
			return "Selecionado: " + t.Format("02/01/2006")
		})).Caption(),
	).Gap(6)
}

func cenaEditor() juigo.Widget {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go())
	ed.SetText("// saúda n vezes\nfunc saudar(n int) {\n\tfor i := 0; i < n; i++ {\n\t\tfmt.Println(\"olá, JUIGo\")\n\t}\n}")
	return juigo.NewVBox(ed.Grow(1))
}

func cenaLista() juigo.Widget {
	lista := juigo.NewTextList(10_000, func(i int) string {
		return fmt.Sprintf("Pedido #%05d", i+1)
	}).BindSelected(juigo.NewState(2))
	return juigo.NewScroll(lista)
}

func cenaTabela() juigo.Widget {
	dados := [][3]string{
		{"Ana Lima", "Plataforma", "Engenheira"},
		{"Bruno Reis", "Design", "Designer"},
		{"Carla Dias", "Dados", "Cientista"},
		{"Davi Rocha", "Plataforma", "SRE"},
	}
	return juigo.NewTable([]string{"Nome", "Equipe", "Função"}, len(dados),
		func(l, c int) string { return dados[l][c] },
	).BindSelected(juigo.NewState(1))
}

func cenaArvore() juigo.Widget {
	filhos := map[string][]string{
		"src":  {"main.go", "ui.go"},
		"docs": {"DESIGN.md"},
	}
	tr := juigo.NewTree(
		func() []string { return []string{"src", "docs", "README.md"} },
		func(id string) []string { return filhos[id] },
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, id string) { t.SetText(id) },
	).BindSelected(juigo.NewState("ui.go"))
	tr.Expand("src")
	return juigo.NewScroll(tr)
}

func cenaLayout() juigo.Widget {
	return juigo.NewVBox(
		juigo.NewHBox(
			juigo.NewInput("Buscar…").Grow(1),
			juigo.NewButton("Buscar", nil),
		).Gap(8),
		juigo.NewGrid(2,
			juigo.NewText("Cidade:"), juigo.NewInput("São Paulo"),
			juigo.NewText("CEP:"), juigo.NewInput("00000-000"),
		).Gap(8),
	).Gap(12)
}

func cenaSuperficies() juigo.Widget {
	return juigo.NewCard(juigo.NewVBox(
		juigo.NewText("Cobrança automática").Subtitle(),
		juigo.NewText("O cartão final 4242 renova no dia 12."),
		juigo.NewDivider(),
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.NewButton("Cancelar", nil).Ghost(),
			juigo.NewButton("Pagar agora", nil),
		).Gap(8),
	).Gap(8)).Pad(14)
}

func cenaAbas() juigo.Widget {
	return juigo.NewTabs().
		Add("Resumo", juigo.NewVBox(
			juigo.NewText("O conteúdo da primeira aba."),
			juigo.NewText("Só a aba ativa recebe eventos.").Caption(),
		).Pad(8).Gap(4)).
		Add("Detalhes", juigo.NewText("Segunda aba")).
		Add("Histórico", juigo.NewText("Terceira aba"))
}

func cenaSplit() juigo.Widget {
	pastas := juigo.NewTextList(3, func(i int) string {
		return []string{"Entrada", "Enviados", "Arquivo"}[i]
	}).BindSelected(juigo.NewState(0))
	leitura := juigo.NewVBox(
		juigo.NewText("Reunião de quinta").Subtitle(),
		juigo.NewLabel("Arraste o divisor para redimensionar os painéis."),
	).Pad(10).Gap(6)
	return juigo.NewSplitPane(pastas, leitura).SetRatio(0.32)
}

func cenaFormulario() juigo.Widget {
	nome := quick.Text("Nome:").Required("Informe o nome").Min(3, "Mínimo de 3")
	email := quick.Text("E-mail:").Email("E-mail inválido")
	idade := quick.Number("Idade:", 18).Max(120, "Confere a idade?")

	nome.Set("Jo")
	email.Set("ana@")
	f := quick.Form(nome, email, idade).Submit("Salvar", func() {
		salvar(nome.Value(), email.Value(), idade.Value())
	})
	f.Model().Submit(nil) // revela os erros
	return f
}

func cenaGraficos() juigo.Widget {
	vendas := []float64{4, 6, 5, 9, 7, 11, 10, 14, 12, 16, 15, 19}
	saldo := []float64{3, -1, 4, 2, -2, 5, 3}
	return juigo.NewVBox(
		juigo.NewText("Vendas do trimestre").Subtitle(),
		chart.NewLine(vendas).Min(0).Grow(1),
		chart.NewBars(saldo).Grow(1),
		juigo.NewHBox(
			juigo.NewText("Receita").Caption().Centered(),
			juigo.NewSpacer(),
			chart.NewSpark(vendas),
		).Gap(8),
	).Gap(8)
}

// publicar e salvar só existem para os exemplos lerem como código real.
func publicar()                  {}
func salvar(string, string, int) {}

// geraComponentes renderiza cada cena nos dois temas papel e tinta e
// escreve docs/site/cenas.js com os metadados e o código de cada uma.
func geraComponentes(claro, escuro *juigo.Theme) {
	if err := os.MkdirAll(filepath.Join("docs", "componentes"), 0o755); err != nil {
		log.Fatal(err)
	}
	corpos, err := corposDasCenas(filepath.Join("docs", "gerar", "componentes.go"))
	if err != nil {
		log.Fatal(err)
	}
	type saida struct {
		ID        string `json:"id"`
		Titulo    string `json:"titulo"`
		Grupo     string `json:"grupo"`
		Descricao string `json:"descricao"`
		W         int    `json:"w"`
		H         int    `json:"h"`
		Codigo    string `json:"codigo"`
	}
	var todas []saida
	for _, c := range cenas {
		nome := runtime.FuncForPC(reflect.ValueOf(c.monta).Pointer()).Name()
		nome = nome[strings.LastIndex(nome, ".")+1:]
		codigo, ok := corpos[nome]
		if !ok {
			log.Fatalf("cena %s: corpo de %s não encontrado", c.ID, nome)
		}
		w, h := c.W+32, c.H+32 // a moldura de 16 px de cada lado
		salvaCena("componentes/"+c.ID+".png", c.monta, claro, w, h)
		salvaCena("componentes/"+c.ID+"-escuro.png", c.monta, escuro, w, h)
		todas = append(todas, saida{c.ID, c.Titulo, c.Grupo, c.Descricao, w, h, codigo})
	}
	js, err := json.MarshalIndent(todas, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	conteudo := "// Gerado por `go run ./docs/gerar` — não edite à mão.\nwindow.CENAS = " + string(js) + ";\n"
	caminho := filepath.Join("docs", "site", "cenas.js")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("gerado %s (%d cenas)", caminho, len(todas))
}

// salvaCena emoldura a cena (padding de 16, crescendo para ocupar o quadro)
// e a renderiza em escala 2. Renderiza DUAS vezes: o Label quebra na
// largura do último layout, então o primeiro quadro só a mede.
func salvaCena(nome string, monta func() juigo.Widget, th *juigo.Theme, w, h int) {
	th2, err := th.Clone()
	if err != nil {
		log.Fatalf("gerar %s: %v", nome, err)
	}
	if err := th2.SetScale(2); err != nil {
		log.Fatalf("gerar %s: %v", nome, err)
	}
	raiz := juigo.NewVBox(juigo.Grow(monta(), 1)).Pad(16)
	offscreen.Render(raiz, th2, 2*w, 2*h)
	caminho := filepath.Join("docs", nome)
	if err := offscreen.SavePNG(caminho, offscreen.Render(raiz, th2, 2*w, 2*h)); err != nil {
		log.Fatalf("gerar %s: %v", caminho, err)
	}
}

// corposDasCenas lê o fonte e devolve, por nome de função, o corpo de cada
// cena sem as chaves e com um nível de indentação a menos.
func corposDasCenas(caminho string) (map[string]string, error) {
	fonte, err := os.ReadFile(caminho)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	arq, err := parser.ParseFile(fset, caminho, fonte, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	corpos := map[string]string{}
	for _, d := range arq.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "cena") {
			continue
		}
		ini := fset.Position(fn.Body.Lbrace).Offset + 1
		fim := fset.Position(fn.Body.Rbrace).Offset
		linhas := strings.Split(strings.Trim(string(fonte[ini:fim]), "\n"), "\n")
		for i, l := range linhas {
			linhas[i] = strings.TrimPrefix(l, "\t")
		}
		corpos[fn.Name.Name] = strings.Join(linhas, "\n")
	}
	return corpos, nil
}
