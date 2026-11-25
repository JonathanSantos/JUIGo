// Gerado por `go run ./docs/gerar` — não edite à mão.
window.CENAS = [
  {
    "id": "tipografia",
    "titulo": "Text e Label",
    "grupo": "Estilo",
    "descricao": "Papéis tipográficos do tema e o parágrafo que quebra sozinho.",
    "w": 492,
    "h": 198,
    "codigo": "return juigo.NewVBox(\n\tjuigo.NewText(\"Relatório mensal\").Title(),\n\tjuigo.NewText(\"Resumo da equipe\").Subtitle(),\n\tjuigo.NewText(\"O corpo usa a fonte regular do tema.\"),\n\tjuigo.NewText(\"Atualizado há 2 minutos\").Caption(),\n\tjuigo.NewLabel(\"Um Label quebra por palavras na largura disponível e \"+\n\t\t\"aceita os mesmos papéis tipográficos do Text.\"),\n).Gap(6)"
  },
  {
    "id": "botoes",
    "titulo": "Button",
    "grupo": "Controles",
    "descricao": "Três hierarquias, estado desabilitado e carregando.",
    "w": 492,
    "h": 108,
    "codigo": "salvando := juigo.NewButton(\"Salvando\", nil)\nsalvando.SetLoading(true)\nreturn juigo.NewVBox(\n\tjuigo.NewHBox(\n\t\tjuigo.NewSpacer(),\n\t\tjuigo.NewButton(\"Cancelar\", nil).Ghost(),\n\t\tjuigo.NewButton(\"Rascunho\", nil).Secondary(),\n\t\tjuigo.NewButton(\"Publicar\", publicar),\n\t).Gap(8),\n\tjuigo.NewHBox(\n\t\tsalvando,\n\t\tjuigo.NewButton(\"Inativo\", nil).BindDisabled(juigo.NewState(true)),\n\t\tjuigo.NewSpacer(),\n\t).Gap(8),\n).Gap(12)"
  },
  {
    "id": "entradas",
    "titulo": "Input e TextArea",
    "grupo": "Controles",
    "descricao": "Placeholder, texto, erro vinculado a um State e multilinha com soft wrap.",
    "w": 492,
    "h": 222,
    "codigo": "nome := juigo.NewState(\"Ana Lima\")\ninvalido := juigo.NewState(true)\nreturn juigo.NewVBox(\n\tjuigo.NewInput(\"Nome\").BindValue(nome),\n\tjuigo.NewInput(\"E-mail\").BindInvalid(invalido),\n\tjuigo.NewText(\"Informe um e-mail válido\").Danger(),\n\tjuigo.NewSized(juigo.NewTextArea(\"Notas…\"), 0, 70),\n).Gap(8)"
  },
  {
    "id": "escolhas",
    "titulo": "Checkbox, Radio e Dropdown",
    "grupo": "Controles",
    "descricao": "Escolhas ligadas a States com BindChecked e BindValue.",
    "w": 492,
    "h": 72,
    "codigo": "avisos := juigo.NewState(true)\nplano := juigo.NewState(\"pro\")\nreturn juigo.NewHBox(\n\tjuigo.NewCheckbox(\"Avisos\").BindChecked(avisos),\n\tjuigo.NewSpacer(),\n\tjuigo.NewRadio(\"Grátis\", \"free\").BindValue(plano),\n\tjuigo.NewRadio(\"Pro\", \"pro\").BindValue(plano),\n\tjuigo.NewSpacer(),\n\tjuigo.NewDropdown(\"Baixa\", \"Média\", \"Alta\"),\n).Gap(8)"
  },
  {
    "id": "slider",
    "titulo": "Slider e ProgressBar",
    "grupo": "Controles",
    "descricao": "Dois widgets no mesmo State: mover um move o outro.",
    "w": 492,
    "h": 112,
    "codigo": "volume := juigo.NewState(0.62)\nreturn juigo.NewVBox(\n\tjuigo.NewSlider(0, 1).Step(0.01).BindValue(volume),\n\tjuigo.NewProgressBar(0, 1).BindValue(volume),\n\tjuigo.NewText(\"\").BindText(juigo.Map(volume, func(v float64) string {\n\t\treturn fmt.Sprintf(\"Volume: %.0f%%\", v*100)\n\t})).Caption(),\n).Gap(10)"
  },
  {
    "id": "calendario",
    "titulo": "Calendar",
    "grupo": "Controles",
    "descricao": "Seleção de data com BindValue em time.Time.",
    "w": 332,
    "h": 282,
    "codigo": "data := juigo.NewState(time.Date(2026, 7, 24, 0, 0, 0, 0, time.Local))\nreturn juigo.NewVBox(\n\tjuigo.NewCalendar().BindValue(data),\n\tjuigo.NewText(\"\").BindText(juigo.Map(data, func(t time.Time) string {\n\t\treturn \"Selecionado: \" + t.Format(\"02/01/2006\")\n\t})).Caption(),\n).Gap(6)"
  },
  {
    "id": "editor",
    "titulo": "CodeEditor",
    "grupo": "Controles",
    "descricao": "Fonte mono, gutter, highlight incremental e undo coalescido.",
    "w": 492,
    "h": 182,
    "codigo": "ed := juigo.NewCodeEditor().Highlight(syntax.Go())\ned.SetText(\"// saúda n vezes\\nfunc saudar(n int) {\\n\\tfor i := 0; i \u003c n; i++ {\\n\\t\\tfmt.Println(\\\"olá, JUIGo\\\")\\n\\t}\\n}\")\nreturn juigo.NewVBox(ed.Grow(1))"
  },
  {
    "id": "lista",
    "titulo": "List",
    "grupo": "Coleções",
    "descricao": "Lista virtualizada: milhares de linhas com meia dúzia de widgets.",
    "w": 332,
    "h": 202,
    "codigo": "lista := juigo.NewTextList(10_000, func(i int) string {\n\treturn fmt.Sprintf(\"Pedido #%05d\", i+1)\n}).BindSelected(juigo.NewState(2))\nreturn juigo.NewScroll(lista)"
  },
  {
    "id": "tabela",
    "titulo": "Table",
    "grupo": "Coleções",
    "descricao": "Cabeçalho fixo, seleção como State e reordenação por arrasto.",
    "w": 492,
    "h": 142,
    "codigo": "dados := [][3]string{\n\t{\"Ana Lima\", \"Plataforma\", \"Engenheira\"},\n\t{\"Bruno Reis\", \"Design\", \"Designer\"},\n\t{\"Carla Dias\", \"Dados\", \"Cientista\"},\n\t{\"Davi Rocha\", \"Plataforma\", \"SRE\"},\n}\nreturn juigo.NewTable([]string{\"Nome\", \"Equipe\", \"Função\"}, len(dados),\n\tfunc(l, c int) string { return dados[l][c] },\n).BindSelected(juigo.NewState(1))"
  },
  {
    "id": "arvore",
    "titulo": "Tree",
    "grupo": "Coleções",
    "descricao": "Árvore virtualizada; o modelo são dois callbacks sobre um ID seu.",
    "w": 332,
    "h": 162,
    "codigo": "filhos := map[string][]string{\n\t\"src\":  {\"main.go\", \"ui.go\"},\n\t\"docs\": {\"DESIGN.md\"},\n}\ntr := juigo.NewTree(\n\tfunc() []string { return []string{\"src\", \"docs\", \"README.md\"} },\n\tfunc(id string) []string { return filhos[id] },\n\tfunc() *juigo.Text { return juigo.NewText(\"\") },\n\tfunc(t *juigo.Text, id string) { t.SetText(id) },\n).BindSelected(juigo.NewState(\"ui.go\"))\ntr.Expand(\"src\")\nreturn juigo.NewScroll(tr)"
  },
  {
    "id": "layout",
    "titulo": "VBox, HBox e Grid",
    "grupo": "Estrutura",
    "descricao": "Layout por composição: Grow, Spacer, Gap e grade alinhada.",
    "w": 492,
    "h": 152,
    "codigo": "return juigo.NewVBox(\n\tjuigo.NewHBox(\n\t\tjuigo.NewInput(\"Buscar…\").Grow(1),\n\t\tjuigo.NewButton(\"Buscar\", nil),\n\t).Gap(8),\n\tjuigo.NewGrid(2,\n\t\tjuigo.NewText(\"Cidade:\"), juigo.NewInput(\"São Paulo\"),\n\t\tjuigo.NewText(\"CEP:\"), juigo.NewInput(\"00000-000\"),\n\t).Gap(8),\n).Gap(12)"
  },
  {
    "id": "superficies",
    "titulo": "Card e Divider",
    "grupo": "Estrutura",
    "descricao": "A superfície que agrupa e o fio que separa.",
    "w": 492,
    "h": 168,
    "codigo": "return juigo.NewCard(juigo.NewVBox(\n\tjuigo.NewText(\"Cobrança automática\").Subtitle(),\n\tjuigo.NewText(\"O cartão final 4242 renova no dia 12.\"),\n\tjuigo.NewDivider(),\n\tjuigo.NewHBox(\n\t\tjuigo.NewSpacer(),\n\t\tjuigo.NewButton(\"Cancelar\", nil).Ghost(),\n\t\tjuigo.NewButton(\"Pagar agora\", nil),\n\t).Gap(8),\n).Gap(8)).Pad(14)"
  },
  {
    "id": "abas",
    "titulo": "Tabs",
    "grupo": "Estrutura",
    "descricao": "Abas: só a ativa participa de eventos e foco.",
    "w": 492,
    "h": 122,
    "codigo": "return juigo.NewTabs().\n\tAdd(\"Resumo\", juigo.NewVBox(\n\t\tjuigo.NewText(\"O conteúdo da primeira aba.\"),\n\t\tjuigo.NewText(\"Só a aba ativa recebe eventos.\").Caption(),\n\t).Pad(8).Gap(4)).\n\tAdd(\"Detalhes\", juigo.NewText(\"Segunda aba\")).\n\tAdd(\"Histórico\", juigo.NewText(\"Terceira aba\"))"
  },
  {
    "id": "split",
    "titulo": "SplitPane",
    "grupo": "Estrutura",
    "descricao": "Painéis redimensionáveis com divisor arrastável.",
    "w": 492,
    "h": 142,
    "codigo": "pastas := juigo.NewTextList(3, func(i int) string {\n\treturn []string{\"Entrada\", \"Enviados\", \"Arquivo\"}[i]\n}).BindSelected(juigo.NewState(0))\nleitura := juigo.NewVBox(\n\tjuigo.NewText(\"Reunião de quinta\").Subtitle(),\n\tjuigo.NewLabel(\"Arraste o divisor para redimensionar os painéis.\"),\n).Pad(10).Gap(6)\nreturn juigo.NewSplitPane(pastas, leitura).SetRatio(0.32)"
  },
  {
    "id": "formulario",
    "titulo": "quick.Form",
    "grupo": "Formulários",
    "descricao": "Campos com handles tipados, validação e erros por campo.",
    "w": 492,
    "h": 300,
    "codigo": "nome := quick.Text(\"Nome:\").Required(\"Informe o nome\").Min(3, \"Mínimo de 3\")\nemail := quick.Text(\"E-mail:\").Email(\"E-mail inválido\")\nidade := quick.Number(\"Idade:\", 18).Max(120, \"Confere a idade?\")\n\nnome.Set(\"Jo\")\nemail.Set(\"ana@\")\nf := quick.Form(nome, email, idade).Submit(\"Salvar\", func() {\n\tsalvar(nome.Value(), email.Value(), idade.Value())\n})\nf.Model().Submit(nil) // revela os erros\nreturn f"
  },
  {
    "id": "graficos",
    "titulo": "chart",
    "grupo": "Dados",
    "descricao": "Linha, barras e sparkline, nas cores do tema.",
    "w": 492,
    "h": 332,
    "codigo": "vendas := []float64{4, 6, 5, 9, 7, 11, 10, 14, 12, 16, 15, 19}\nsaldo := []float64{3, -1, 4, 2, -2, 5, 3}\nreturn juigo.NewVBox(\n\tjuigo.NewText(\"Vendas do trimestre\").Subtitle(),\n\tchart.NewLine(vendas).Min(0).Grow(1),\n\tchart.NewBars(saldo).Grow(1),\n\tjuigo.NewHBox(\n\t\tjuigo.NewText(\"Receita\").Caption().Centered(),\n\t\tjuigo.NewSpacer(),\n\t\tchart.NewSpark(vendas),\n\t).Gap(8),\n).Gap(8)"
  }
];
