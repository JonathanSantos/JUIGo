package quick

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/widget"
)

// O diálogo de arquivos é 100% JUIGo (os.ReadDir + List virtualizada) —
// nada de dependência de plataforma. Navegação: clicar numa pasta entra,
// "↑" sobe, clicar num arquivo o seleciona; Escape, Cancel e clique fora
// cancelam. Arquivos ocultos (prefixo ".") ficam de fora da listagem.

// entrada é um item do diretório corrente do diálogo.
type entrada struct {
	nome string
	dir  bool
}

// lerDir lista o diretório com pastas primeiro e ordem alfabética
// case-insensitive, sem os ocultos.
func lerDir(caminho string) ([]entrada, error) {
	itens, err := os.ReadDir(caminho)
	if err != nil {
		return nil, err
	}
	var out []entrada
	for _, it := range itens {
		if strings.HasPrefix(it.Name(), ".") {
			continue
		}
		out = append(out, entrada{nome: it.Name(), dir: it.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].dir != out[j].dir {
			return out[i].dir
		}
		return strings.ToLower(out[i].nome) < strings.ToLower(out[j].nome)
	})
	return out, nil
}

// fileDialog é o miolo compartilhado por OpenFile e SaveFile: o diretório
// corrente projetado numa List virtualizada, com navegação por seleção.
type fileDialog struct {
	dir     string
	itens   []entrada
	caminho *state.State[string]
	erro    *state.State[string]
	// selArq é o NOME do arquivo selecionado ("" = nenhum) — o que habilita
	// o botão de confirmação do OpenFile.
	selArq *state.State[string]
	selIdx *state.State[int]
	lista  *widget.List[*widget.Text]
	rolar  *widget.Scroll
	// aoEscolherArquivo avisa o SaveFile para copiar o nome ao campo.
	aoEscolherArquivo func(nome string)
	// navegando guarda contra reentrada dos watches durante a troca de
	// diretório.
	navegando bool
}

// novoDialogo monta o miolo já listando o diretório inicial.
func novoDialogo(inicio string) *fileDialog {
	d := &fileDialog{
		caminho: state.New(""),
		erro:    state.New(""),
		selArq:  state.New(""),
		selIdx:  state.New(-1),
	}
	d.lista = widget.NewList(0,
		func() *widget.Text { return widget.NewText("") },
		func(t *widget.Text, i int) {
			if i < 0 || i >= len(d.itens) {
				t.SetText("")
				return
			}
			e := d.itens[i]
			if e.dir {
				t.SetText(e.nome + "/")
			} else {
				t.SetText(e.nome)
			}
		},
	).BindSelected(d.selIdx)
	d.rolar = widget.NewScroll(d.lista)
	d.selIdx.Watch(d.aoSelecionar)
	d.navega(inicio)
	return d
}

// inicioPadrao devolve a pasta pessoal do usuário (ou "." sem ela).
func inicioPadrao() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	return "."
}

// aoSelecionar reage ao clique numa linha: pasta navega, arquivo seleciona.
func (d *fileDialog) aoSelecionar(i int) {
	if d.navegando || i < 0 || i >= len(d.itens) {
		return
	}
	e := d.itens[i]
	if e.dir {
		d.navega(filepath.Join(d.dir, e.nome))
		return
	}
	d.selArq.Set(e.nome)
	if d.aoEscolherArquivo != nil {
		d.aoEscolherArquivo(e.nome)
	}
}

// navega troca o diretório corrente; em erro (permissão, remoção), mantém o
// atual e exibe a causa.
func (d *fileDialog) navega(caminho string) {
	itens, err := lerDir(caminho)
	if err != nil {
		d.erro.Set("Não deu para abrir: " + err.Error())
		return
	}
	d.navegando = true
	defer func() { d.navegando = false }()
	d.dir = caminho
	d.itens = itens
	d.erro.Set("")
	d.selArq.Set("")
	d.selIdx.Set(-1)
	d.caminho.Set(caminho)
	d.lista.SetCount(len(itens))
	d.lista.Refresh()
	d.rolar.ScrollTo(0)
}

// sobe navega ao diretório pai.
func (d *fileDialog) sobe() {
	d.navega(filepath.Dir(d.dir))
}

// vista monta o corpo do diálogo: título, cabeçalho de navegação, a lista
// dimensionada, a linha de erro e a barra de ações.
func (d *fileDialog) vista(title string, extra widget.Widget, acoes *widget.HBox) *widget.VBox {
	cab := widget.NewHBox(
		widget.Tooltip(widget.NewButton("↑", d.sobe), "Pasta acima"),
		widget.NewText("").BindText(d.caminho),
	).Gap(8)
	v := widget.NewVBox(
		widget.NewText(title).Subtitle(),
		cab,
		widget.NewSized(d.rolar, 380, 240),
		widget.NewText("").BindText(d.erro).Danger(),
	).Gap(8)
	if extra != nil {
		v.Add(extra)
	}
	v.Add(acoes)
	return v
}

// OpenFile abre um seletor de arquivo sobre a janela, começando na pasta
// pessoal: clique entra nas pastas ("↑" sobe), selecione um arquivo e
// confirme em Open. onResult é chamado exatamente uma vez quando o diálogo
// fechar: (caminho completo, true) no Open; ("", false) no Cancel, no
// Escape ou no clique fora.
func OpenFile(title string, onResult func(path string, ok bool)) *widget.Modal {
	return OpenFileIn(inicioPadrao(), title, onResult)
}

// OpenFileIn é o OpenFile começando no diretório dado — útil para começar
// onde a aplicação trabalha (e para testes determinísticos).
func OpenFileIn(dir, title string, onResult func(path string, ok bool)) *widget.Modal {
	d := novoDialogo(dir)
	ok := false
	caminho := ""
	var m *widget.Modal
	abrir := widget.NewButton("Open", func() {
		if d.selArq.Get() == "" {
			return
		}
		caminho = filepath.Join(d.dir, d.selArq.Get())
		ok = true
		m.Close()
	})
	widget.BindDisabled(abrir, state.Map(d.selArq, func(s string) bool { return s == "" }))
	m = widget.NewModal(d.vista(title, nil, Buttons(
		widget.NewButton("Cancel", func() { m.Close() }).Secondary(),
		abrir,
	))).OnClose(func() {
		if onResult != nil {
			onResult(caminho, ok)
		}
	})
	m.Show()
	return m
}

// SaveFile abre um diálogo de salvar sobre a janela, começando na pasta
// pessoal, com o campo de nome preenchido com initialName; clicar num
// arquivo existente copia o nome para o campo. Save (ou Enter no campo)
// chama onResult com (diretório corrente + nome, true) — o arquivo NÃO é
// criado nem verificado: sobrescrever ou não é decisão do chamador.
// Cancelamentos chamam ("", false). onResult roda exatamente uma vez.
func SaveFile(title, initialName string, onResult func(path string, ok bool)) *widget.Modal {
	return SaveFileIn(inicioPadrao(), title, initialName, onResult)
}

// SaveFileIn é o SaveFile começando no diretório dado.
func SaveFileIn(dir, title, initialName string, onResult func(path string, ok bool)) *widget.Modal {
	d := novoDialogo(dir)
	nome := state.New(initialName)
	d.aoEscolherArquivo = func(n string) { nome.Set(n) }
	ok := false
	caminho := ""
	var m *widget.Modal
	salvar := func() {
		n := strings.TrimSpace(nome.Get())
		if n == "" {
			return
		}
		caminho = filepath.Join(d.dir, n)
		ok = true
		m.Close()
	}
	confirmar := widget.NewButton("Save", salvar)
	widget.BindDisabled(confirmar, state.Map(nome, func(s string) bool {
		return strings.TrimSpace(s) == ""
	}))
	campo := Labeled("Nome:", widget.NewInput("nome do arquivo…").BindValue(nome).OnSubmit(salvar))
	m = widget.NewModal(d.vista(title, campo, Buttons(
		widget.NewButton("Cancel", func() { m.Close() }).Secondary(),
		confirmar,
	))).OnClose(func() {
		if onResult != nil {
			onResult(caminho, ok)
		}
	})
	m.Show()
	return m
}
