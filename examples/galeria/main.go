// Galeria — o "storybook" do JUIGo, feito com o próprio JUIGo: a barra
// lateral (Tree) navega pelos componentes, cada página mostra instâncias
// VIVAS em cartões, e a barra do topo troca tema, fonte dos títulos, fonte
// mono e tamanho de texto em pleno voo. As páginas também viram comandos
// globais (menu Componentes), então Ctrl/Cmd+K é a busca de componentes.
// A própria galeria é a demo: SplitPane, Tree, MenuBar, Cards e o design
// system trabalhando juntos.
package main

import (
	"fmt"
	"image"
	"log"
	"strconv"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/chart"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/syntax"
	"github.com/JonathanSantos/JUIGo/theme"
	"golang.org/x/image/font/opentype"
)

func main() {
	app, err := juigo.New("Galeria — os componentes do JUIGo", 1020, 660)
	if err != nil {
		log.Fatal(err)
	}
	g, err := nova(func(th *juigo.Theme) {
		if err := app.SetTheme(th); err != nil {
			log.Println("galeria:", err)
		}
		app.Invalidate()
	})
	if err != nil {
		log.Fatal(err)
	}
	app.SetRoot(g.Raiz)
	g.aplicar() // tema inicial: papel e tinta
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// pagina é uma entrada do storybook: onde mora na árvore e como montar a
// demonstração (reconstruída a cada visita — estado fresco).
type pagina struct {
	categoria string
	nome      string
	build     func(g *galeria) juigo.Widget
}

// paginas define a ordem da barra lateral e do menu Componentes.
var paginas = []pagina{
	{"Estilo", "Tipografia", pgTipografia},
	{"Estilo", "Cores do tema", pgCores},
	{"Controles", "Botões", pgBotoes},
	{"Controles", "Entradas", pgEntradas},
	{"Controles", "Seleção", pgSelecao},
	{"Coleções", "Lista e Tabela", pgColecoes},
	{"Coleções", "Árvore", pgArvore},
	{"Estrutura", "Superfícies", pgSuperficies},
	{"Estrutura", "Navegação", pgNavegacao},
	{"Estrutura", "Formulários", pgFormularios},
	{"Dados", "Gráficos", pgGraficos},
}

// galeria é a aplicação: temas prontos, knobs de estilo e a página aberta.
type galeria struct {
	Raiz juigo.Widget

	// aoTema entrega o tema efetivo ao dono (App.SetTheme no app real,
	// Session.SetTheme nos testes).
	aoTema func(*juigo.Theme)

	temas    map[string]*juigo.Theme
	temaNome *juigo.State[string]
	titulos  *juigo.State[string]
	mono     *juigo.State[string]
	tamanho  *juigo.State[string]

	selecao  *juigo.State[string]
	conteudo *juigo.VBox
}

// nova monta a galeria; aoTema recebe o tema a cada mudança de knob.
func nova(aoTema func(*juigo.Theme)) (*galeria, error) {
	g := &galeria{
		aoTema:   aoTema,
		temaNome: juigo.NewState("Papel e tinta"),
		titulos:  juigo.NewState("Lora"),
		mono:     juigo.NewState("Go Mono"),
		tamanho:  juigo.NewState("16"),
		selecao:  juigo.NewState("Tipografia"),
	}

	g.temas = map[string]*juigo.Theme{}
	for nome, criar := range map[string]func() (*juigo.Theme, error){
		"Papel e tinta":        juigo.ClaudeTheme,
		"Papel e tinta escuro": juigo.ClaudeDarkTheme,
		"Padrão":               juigo.DefaultTheme,
		"Padrão escuro":        juigo.DarkTheme,
		"Clássico":             juigo.ClassicTheme,
	} {
		th, err := criar()
		if err != nil {
			return nil, err
		}
		g.temas[nome] = th
	}

	// Barra lateral: a árvore de categorias/páginas (o Tree navegando por
	// ele mesmo, inclusive).
	porCategoria := map[string][]string{}
	var categorias []string
	for _, p := range paginas {
		if len(porCategoria[p.categoria]) == 0 {
			categorias = append(categorias, p.categoria)
		}
		porCategoria[p.categoria] = append(porCategoria[p.categoria], p.nome)
	}
	arvore := juigo.NewTree(
		func() []string { return categorias },
		func(id string) []string { return porCategoria[id] },
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, id string) { t.SetText(id) },
	).BindSelected(g.selecao)
	for _, c := range categorias {
		arvore.Expand(c)
	}
	g.selecao.Watch(func(string) { g.reprojeta() })

	// Barra do topo: os knobs de estilo, aplicados ao vivo.
	knob := func(rotulo string, st *juigo.State[string], opcoes ...string) juigo.Widget {
		d := juigo.NewDropdown(opcoes...).BindValue(st)
		return juigo.NewHBox(
			juigo.Centered(juigo.NewText(rotulo).Caption()),
			d,
		).Gap(4)
	}
	for _, st := range []*juigo.State[string]{g.temaNome, g.titulos, g.mono, g.tamanho} {
		st.Watch(func(string) { g.aplicar() })
	}
	barraEstilo := juigo.NewHBox(
		knob("Tema", g.temaNome, "Papel e tinta", "Papel e tinta escuro", "Padrão", "Padrão escuro", "Clássico"),
		knob("Títulos", g.titulos, "Lora", "Lora Bold", "Go Bold"),
		knob("Mono", g.mono, "Go Mono", "Fira Code"),
		knob("Tamanho", g.tamanho, "14", "16", "18"),
		juigo.NewSpacer(),
		juigo.Centered(juigo.NewText("Ctrl/Cmd+K busca componentes").Caption()),
	).Pad(8).Gap(10)

	// Menu: as páginas viram comandos globais — e a paleta, uma busca.
	comandos := make([]juigo.Command, 0, len(paginas))
	for _, p := range paginas {
		nome := p.nome
		comandos = append(comandos, juigo.Command{
			Title:  "Abrir: " + nome,
			Action: func() { g.selecao.Set(nome) },
		})
	}
	barraMenu := juigo.NewMenuBar().
		Menu("Galeria",
			juigo.Command{Title: "Alternar claro/escuro", Key: juigo.LetterKey('t'),
				Mods: juigo.ModControl, Action: g.alternarEscuro},
			juigo.MenuSeparator(),
			juigo.Command{Title: "Sobre a galeria", Action: func() {
				quick.Alert("Um storybook do JUIGo feito com o próprio JUIGo.")
			}},
		).
		Menu("Componentes", comandos...)

	g.conteudo = juigo.NewVBox().Pad(16).Gap(10)
	painel := juigo.NewScroll(g.conteudo)

	lateral := juigo.NewVBox(
		juigo.NewText("Componentes").Subtitle(),
		juigo.Grow(juigo.NewScroll(arvore), 1),
	).Pad(10).Gap(8)

	g.Raiz = juigo.NewVBox(
		barraMenu,
		barraEstilo,
		juigo.NewDivider(),
		juigo.Grow(juigo.NewSplitPane(lateral, painel).Ratio(0.22).Min(140), 1),
	)
	g.reprojeta()
	return g, nil
}

// temaAtual devolve o tema selecionado nos knobs.
func (g *galeria) temaAtual() *juigo.Theme {
	return g.temas[g.temaNome.Get()]
}

// aplicar leva TODOS os knobs ao tema selecionado e o entrega ao dono: o
// tamanho re-rasteriza as faces (SetScale na escala corrente), e as fontes
// de display/mono trocam por cima.
func (g *galeria) aplicar() {
	th := g.temaAtual()
	if th == nil {
		return
	}
	if v, err := strconv.Atoi(g.tamanho.Get()); err == nil {
		th.FontSize = float64(v)
	}
	if err := th.SetScale(th.Scale()); err != nil {
		log.Println("galeria:", err)
		return
	}
	if f, err := fonteTitulos(g.titulos.Get()); err == nil {
		if err := th.UseDisplayFont(f); err != nil {
			log.Println("galeria:", err)
		}
	}
	if f, err := fonteMono(g.mono.Get()); err == nil {
		if err := th.UseMonoFont(f); err != nil {
			log.Println("galeria:", err)
		}
	}
	g.aoTema(th)
}

// alternarEscuro troca entre o par claro/escuro da família corrente
// (comando Ctrl/Cmd+T do menu).
func (g *galeria) alternarEscuro() {
	switch g.temaNome.Get() {
	case "Papel e tinta":
		g.temaNome.Set("Papel e tinta escuro")
	case "Papel e tinta escuro":
		g.temaNome.Set("Papel e tinta")
	case "Padrão", "Clássico":
		g.temaNome.Set("Padrão escuro")
	default:
		g.temaNome.Set("Padrão")
	}
}

// fonteTitulos resolve o knob da fonte de display.
func fonteTitulos(nome string) (*opentype.Font, error) {
	switch nome {
	case "Lora Bold":
		return theme.LoraBold()
	case "Go Bold":
		return theme.GoBold()
	default:
		return theme.Lora()
	}
}

// fonteMono resolve o knob da fonte mono.
func fonteMono(nome string) (*opentype.Font, error) {
	if nome == "Fira Code" {
		return theme.FiraCode()
	}
	return theme.GoMono()
}

// reprojeta reconstrói o painel com a página selecionada (categorias na
// árvore não têm página — ignoradas).
func (g *galeria) reprojeta() {
	var atual *pagina
	for i := range paginas {
		if paginas[i].nome == g.selecao.Get() {
			atual = &paginas[i]
			break
		}
	}
	if atual == nil {
		return
	}
	g.conteudo.Clear()
	g.conteudo.Add(
		juigo.NewText(atual.nome).Title(),
		atual.build(g),
	)
}

// bloco embala uma demonstração num cartão com legenda — a unidade visual
// das páginas.
func bloco(legenda string, w juigo.Widget) juigo.Widget {
	return juigo.NewVBox(
		juigo.NewText(legenda).Caption(),
		juigo.NewCard(w).Pad(12),
	).Gap(4)
}

// ---- Páginas ----

func pgTipografia(g *galeria) juigo.Widget {
	return juigo.NewVBox(
		bloco("Papéis do tema — Title, Subtitle, corpo e Caption", juigo.NewVBox(
			juigo.NewText("Título em fonte de display").Title(),
			juigo.NewText("Subtítulo abre seções e cartões").Subtitle(),
			juigo.NewText("O corpo é a fonte regular do tema, para o dia a dia."),
			juigo.NewText("A legenda apaga metadados e notas.").Caption(),
		).Gap(6)),
		bloco("Label — o parágrafo que quebra sozinho", juigo.NewLabel(
			"Um Label quebra por palavras na largura disponível, respeita quebras "+
				"duras e aceita os mesmos papéis tipográficos. Troque o tamanho do "+
				"texto na barra acima e veja o reflow acompanhar.")),
	).Gap(10)
}

func pgCores(g *galeria) juigo.Widget {
	return juigo.NewVBox(
		bloco("O texto nas cores semânticas do tema", juigo.NewVBox(
			juigo.NewText("Texto padrão (tinta)"),
			juigo.NewText("Apagado, para sugestões").Caption(),
			juigo.NewText("Perigo — erros e ações destrutivas").Danger(),
		).Gap(6)),
		bloco("Seleção e realce vêm do tema, nunca do widget", juigo.NewVBox(
			juigo.NewText("Selecione o texto abaixo para ver a cor Selection:"),
			campoComTexto("papel, tinta e terracota"),
		).Gap(6)),
	).Gap(10)
}

// campoComTexto cria um Input já preenchido.
func campoComTexto(s string) *juigo.Input {
	in := juigo.NewInput("")
	in.SetText(s)
	return in
}

func pgBotoes(g *galeria) juigo.Widget {
	carregando := juigo.NewButton("Simular trabalho", nil)
	carregando.OnClick(func() {
		carregando.SetLoading(true)
		juigo.After(1500*time.Millisecond, func() { carregando.SetLoading(false) })
	})
	desabilitado := juigo.NewState(true)
	return juigo.NewVBox(
		bloco("A hierarquia: um primário por bloco, à direita", juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.NewButton("Ghost", nil).Ghost(),
			juigo.NewButton("Secundário", nil).Secondary(),
			juigo.NewButton("Primário", nil),
		).Gap(8)),
		bloco("Estados: loading implica desabilitado", juigo.NewHBox(
			carregando,
			juigo.NewSpacer(),
			juigo.NewCheckbox("Desabilitar exemplo").BindChecked(desabilitado),
			juigo.BindDisabled(juigo.NewButton("Exemplo", nil).Secondary(), desabilitado),
		).Gap(8)),
	).Gap(10)
}

func pgEntradas(g *galeria) juigo.Widget {
	ed := juigo.NewCodeEditor().Highlight(syntax.Go())
	ed.SetText("// A fonte mono vem do knob acima\nfunc ola() string {\n\treturn \"galeria\"\n}")
	return juigo.NewVBox(
		bloco("Input — placeholder, foco e borda de erro", juigo.NewVBox(
			juigo.NewInput("Digite aqui…"),
			juigo.NewInput("inválido").BindInvalid(juigo.NewState(true)),
		).Gap(8)),
		bloco("TextArea com soft wrap", juigo.NewTextArea("Notas multilinha…")),
		bloco("CodeEditor — gutter, highlight e tab stops", juigo.NewSized(ed, 0, 120)),
	).Gap(10)
}

func pgSelecao(g *galeria) juigo.Widget {
	plano := juigo.NewState("pro")
	volume := juigo.NewState(0.6)
	data := juigo.NewState(time.Time{})
	rotuloData := juigo.Map(data, func(t time.Time) string {
		if t.IsZero() {
			return "Clique num dia do calendário."
		}
		return "Selecionado: " + t.Format("02/01/2006")
	})
	return juigo.NewVBox(
		bloco("Escolhas", juigo.NewHBox(
			juigo.NewCheckbox("Notificações"),
			juigo.NewSpacer(),
			juigo.NewRadio("Grátis", "free").BindValue(plano),
			juigo.NewRadio("Pro", "pro").BindValue(plano),
			juigo.NewSpacer(),
			juigo.NewDropdown("Baixa", "Média", "Alta"),
		).Gap(8)),
		bloco("Slider e ProgressBar no MESMO State", juigo.NewVBox(
			juigo.NewSlider(0, 1).BindValue(volume),
			juigo.NewProgressBar(0, 1).BindValue(volume),
		).Gap(8)),
		bloco("Calendar — BindValue em time.Time", juigo.NewVBox(
			juigo.NewCalendar().BindValue(data),
			juigo.NewText("").BindText(rotuloData).Caption(),
		).Gap(6)),
	).Gap(10)
}

func pgColecoes(g *galeria) juigo.Widget {
	sel := juigo.NewState(2)
	lista := juigo.NewList(200,
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, i int) { t.SetText(fmt.Sprintf("Item %03d", i)) },
	).BindSelected(sel)
	tabela := juigo.NewTable([]string{"Nome", "Equipe", "Função"}, 4, func(l, c int) string {
		dados := [][3]string{
			{"Ana Lima", "Plataforma", "Engenheira"},
			{"Bruno Reis", "Design", "Designer"},
			{"Carla Dias", "Dados", "Cientista"},
			{"Davi Rocha", "Plataforma", "SRE"},
		}
		return dados[l][c]
	})
	return juigo.NewVBox(
		bloco("List virtualizada (200 itens, meia dúzia de widgets)",
			juigo.NewSized(juigo.NewScroll(lista), 0, 140)),
		bloco("Table — cabeçalho fixo em legenda", tabela),
	).Gap(10)
}

func pgArvore(g *galeria) juigo.Widget {
	filhos := map[string][]string{
		"src":     {"src/main.go", "src/ui.go"},
		"docs":    {"docs/DESIGN.md", "docs/BENCHMARK.md"},
		"src/ui2": nil,
	}
	tr := juigo.NewTree(
		func() []string { return []string{"src", "docs", "README.md"} },
		func(id string) []string { return filhos[id] },
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, id string) { t.SetText(id) },
	).BindSelected(juigo.NewState(""))
	tr.Expand("src")
	return juigo.NewVBox(
		bloco("Tree — expansão por chevron, teclado e pílulas",
			juigo.NewSized(juigo.NewScroll(tr), 0, 160)),
		juigo.NewLabel("A barra lateral desta galeria é este mesmo widget — o modelo "+
			"são dois callbacks sobre um ID seu.").Caption(),
	).Gap(10)
}

func pgSuperficies(g *galeria) juigo.Widget {
	var menuBtn *juigo.Button
	menuBtn = juigo.NewButton("Menu de contexto", func() {
		b := menuBtn.Bounds()
		quick.Menu(image.Pt(b.Min.X, b.Max.Y+2),
			quick.Item("Primeira ação", func() { quick.Toast("Primeira!") }),
			quick.Item("Segunda ação", func() { quick.Toast("Segunda!") }),
		)
	}).Secondary()
	return juigo.NewVBox(
		bloco("Card e Divider — a superfície e o fio", juigo.NewVBox(
			juigo.NewText("Cartão é agrupamento, não decoração.").Caption(),
			juigo.NewDivider(),
			juigo.Tooltip(juigo.NewText("Passe o ponteiro aqui para a dica."), "Tooltips falam em legenda."),
		).Gap(6)),
		bloco("Camadas: modal, menu e toast", juigo.NewHBox(
			juigo.NewButton("Confirmar…", func() {
				quick.Confirm("Apagar o rascunho?", func(ok bool) {
					if ok {
						quick.Toast("Apagado.")
					}
				})
			}).Secondary(),
			menuBtn,
			juigo.NewButton("Toast", func() { quick.Toast("Salvo!") }).Ghost(),
		).Gap(8)),
	).Gap(10)
}

func pgNavegacao(g *galeria) juigo.Widget {
	abas := juigo.NewTabs().
		Add("Primeira", juigo.NewVBox(juigo.NewText("O conteúdo da primeira aba.")).Pad(8)).
		Add("Segunda", juigo.NewVBox(juigo.NewText("Só a aba ativa participa de tudo.")).Pad(8))

	nav := juigo.NewNavigator()
	var telaA, telaB juigo.Widget
	telaA = juigo.NewVBox(
		juigo.NewText("Tela A"),
		juigo.NewButton("Avançar →", func() { nav.Push(telaB) }),
	).Pad(10).Gap(8)
	telaB = juigo.NewVBox(
		juigo.NewText("Tela B"),
		juigo.NewButton("← Voltar", func() { nav.Pop() }).Secondary(),
	).Pad(10).Gap(8)
	nav.Push(telaA)

	return juigo.NewVBox(
		bloco("Tabs", abas),
		bloco("Navigator — transições animadas numa área de 200px",
			juigo.NewSized(nav, 0, 120)),
	).Gap(10)
}

func pgFormularios(g *galeria) juigo.Widget {
	nome := quick.Text("Nome:").Required("Informe o nome")
	mail := quick.Text("E-mail:").Email("E-mail inválido")
	ida := quick.Date("Ida:", "data inválida")
	f := quick.Form(nome, mail, ida).Submit("Enviar", func() {
		quick.Toast("Enviado: " + nome.Value())
	})
	return juigo.NewVBox(
		bloco("quick.Form — handles tipados, erros por campo, calendário no botão", f),
	).Gap(10)
}

func pgGraficos(g *galeria) juigo.Widget {
	vendas := []float64{4, 6, 5, 9, 7, 11, 10, 14, 12, 16, 15, 19}
	saldo := []float64{3, -1, 4, 2, -2, 5, 3}
	return juigo.NewVBox(
		bloco("Line — série AA com extremos em legenda",
			juigo.NewSized(chart.NewLine(vendas).Min(0), 0, 140)),
		bloco("Bars — negativos descem da linha do zero",
			juigo.NewSized(chart.NewBars(saldo), 0, 120)),
		bloco("Spark — do tamanho de uma palavra", juigo.NewHBox(
			juigo.Centered(juigo.NewText("Receita").Caption()),
			juigo.NewSpacer(),
			chart.NewSpark(vendas),
		).Gap(8)),
	).Gap(10)
}
