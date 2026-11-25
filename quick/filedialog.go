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
type entry struct {
	name string
	dir  bool
}

// lerDir lista o diretório com pastas primeiro e ordem alfabética
// case-insensitive, sem os ocultos.
func readDir(dirPath string) ([]entry, error) {
	items, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, it := range items {
		if strings.HasPrefix(it.Name(), ".") {
			continue
		}
		out = append(out, entry{name: it.Name(), dir: it.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].dir != out[j].dir {
			return out[i].dir
		}
		return strings.ToLower(out[i].name) < strings.ToLower(out[j].name)
	})
	return out, nil
}

// fileDialog é o miolo compartilhado por OpenFile e SaveFile: o diretório
// corrente projetado numa List virtualizada, com navegação por seleção.
type fileDialog struct {
	dir     string
	items   []entry
	dirPath *state.State[string]
	errText *state.State[string]
	// selArq é o NOME do arquivo selecionado ("" = nenhum) — o que habilita
	// o botão de confirmação do OpenFile.
	selFile  *state.State[string]
	selIdx   *state.State[int]
	list     *widget.List[*widget.HBox]
	scroller *widget.Scroll
	// aoEscolherArquivo avisa o SaveFile para copiar o nome ao campo.
	onFilePicked func(name string)
	// navegando guarda contra reentrada dos watches durante a troca de
	// diretório.
	navigating bool
}

// novoDialogo monta o miolo já listando o diretório inicial.
func newFileDialog(start string) *fileDialog {
	d := &fileDialog{
		dirPath: state.New(""),
		errText: state.New(""),
		selFile: state.New(""),
		selIdx:  state.New(-1),
	}
	// Cada linha é o texto com recuo interno (a pílula de seleção/hover da
	// List ganha respiro dos dois lados — layout do design system).
	d.list = widget.NewList(0,
		func() *widget.HBox { return widget.NewHBox(widget.NewText("")).Pad(6) },
		func(row *widget.HBox, i int) {
			t := row.Children()[0].(*widget.Text)
			if i < 0 || i >= len(d.items) {
				t.SetText("")
				return
			}
			e := d.items[i]
			if e.dir {
				t.SetText(e.name + "/")
			} else {
				t.SetText(e.name)
			}
		},
	).BindSelected(d.selIdx)
	d.scroller = widget.NewScroll(d.list)
	d.selIdx.Watch(d.aoSelecionar)
	d.navigate(start)
	return d
}

// inicioPadrao devolve a pasta pessoal do usuário (ou "." sem ela).
func defaultStart() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	return "."
}

// aoSelecionar reage ao clique numa linha: pasta navega, arquivo seleciona.
func (d *fileDialog) aoSelecionar(i int) {
	if d.navigating || i < 0 || i >= len(d.items) {
		return
	}
	e := d.items[i]
	if e.dir {
		d.navigate(filepath.Join(d.dir, e.name))
		return
	}
	d.selFile.Set(e.name)
	if d.onFilePicked != nil {
		d.onFilePicked(e.name)
	}
}

// navega troca o diretório corrente; em erro (permissão, remoção), mantém o
// atual e exibe a causa.
func (d *fileDialog) navigate(dirPath string) {
	items, err := readDir(dirPath)
	if err != nil {
		d.errText.Set(widget.CurrentLocale().OpenError + err.Error())
		return
	}
	d.navigating = true
	defer func() { d.navigating = false }()
	d.dir = dirPath
	d.items = items
	d.errText.Set("")
	d.selFile.Set("")
	d.selIdx.Set(-1)
	d.dirPath.Set(dirPath)
	d.list.SetCount(len(items))
	d.list.Refresh()
	d.scroller.ScrollTo(0)
}

// sobe navega ao diretório pai.
func (d *fileDialog) goUp() {
	d.navigate(filepath.Dir(d.dir))
}

// vista monta o corpo do diálogo na linguagem do design system: título em
// Subtitle, navegação com "↑" discreto (Ghost) e o caminho em legenda, a
// lista num POÇO de superfície (Card), a linha de erro, e a barra de ações
// separada por um Divider.
func (d *fileDialog) view(title string, extra widget.Widget, actionBar *widget.HBox) *widget.VBox {
	header := widget.NewHBox(
		widget.NewButton("↑", d.goUp).Ghost().Tooltip(widget.CurrentLocale().FolderUp),
		widget.NewText("").BindText(d.dirPath).Caption().Centered(),
	).Gap(8)
	v := widget.NewVBox(
		widget.NewText(title).Subtitle(),
		header,
		widget.NewCard(widget.NewSized(d.scroller, 380, 236)).Pad(4),
		widget.NewText("").BindText(d.errText).Danger().Caption(),
	).Gap(8)
	if extra != nil {
		v.Add(extra)
	}
	v.Add(widget.NewDivider())
	v.Add(actionBar)
	return v
}

// OpenFile abre um seletor de arquivo sobre a janela, começando na pasta
// pessoal: clique entra nas pastas ("↑" sobe), selecione um arquivo e
// confirme em Open. onResult é chamado exatamente uma vez quando o diálogo
// fechar: (caminho completo, true) no Open; ("", false) no Cancel, no
// Escape ou no clique fora.
func OpenFile(title string, onResult func(path string, ok bool)) *widget.Modal {
	return OpenFileIn(defaultStart(), title, onResult)
}

// OpenFileIn é o OpenFile começando no diretório dado — útil para começar
// onde a aplicação trabalha (e para testes determinísticos).
func OpenFileIn(dir, title string, onResult func(path string, ok bool)) *widget.Modal {
	d := newFileDialog(dir)
	ok := false
	dirPath := ""
	var m *widget.Modal
	openAction := widget.NewButton(widget.CurrentLocale().Open, func() {
		if d.selFile.Get() == "" {
			return
		}
		dirPath = filepath.Join(d.dir, d.selFile.Get())
		ok = true
		m.Close()
	})
	openAction.BindDisabled(state.Map(d.selFile, func(s string) bool { return s == "" }))
	m = widget.NewModal(d.view(title, nil, Buttons(
		widget.NewButton(widget.CurrentLocale().Cancel, func() { m.Close() }).Secondary(),
		openAction,
	))).OnClose(func() {
		if onResult != nil {
			onResult(dirPath, ok)
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
	return SaveFileIn(defaultStart(), title, initialName, onResult)
}

// SaveFileIn é o SaveFile começando no diretório dado.
func SaveFileIn(dir, title, initialName string, onResult func(path string, ok bool)) *widget.Modal {
	d := newFileDialog(dir)
	name := state.New(initialName)
	d.onFilePicked = func(n string) { name.Set(n) }
	ok := false
	dirPath := ""
	var m *widget.Modal
	save := func() {
		n := strings.TrimSpace(name.Get())
		if n == "" {
			return
		}
		dirPath = filepath.Join(d.dir, n)
		ok = true
		m.Close()
	}
	confirm := widget.NewButton(widget.CurrentLocale().Save, save)
	confirm.BindDisabled(state.Map(name, func(s string) bool {
		return strings.TrimSpace(s) == ""
	}))
	field := Labeled(widget.CurrentLocale().FileName, widget.NewInput(widget.CurrentLocale().FileNamePlaceholder).BindValue(name).OnSubmit(func(string) { save() }))
	m = widget.NewModal(d.view(title, field, Buttons(
		widget.NewButton(widget.CurrentLocale().Cancel, func() { m.Close() }).Secondary(),
		confirm,
	))).OnClose(func() {
		if onResult != nil {
			onResult(dirPath, ok)
		}
	})
	m.Show()
	return m
}
