// Package crud é o 7GUIs nº 5: cadastro com lista filtrável, seleção e
// criar/atualizar/excluir. O modelo de domínio (Pessoa) vive separado da
// interface; a lista usa a List virtualizada da lib com um WIDGET CUSTOM de
// linha — seleção de lista não é nativa do JUIGo (ver ../GAPS.md), então a
// linha desenha o próprio realce e trata o clique, e o modelo chama
// lista.Refresh() a cada mudança.
package crud

import (
	"fmt"
	"image"
	"strings"

	"juigo"
	"juigo/render"
)

// Pessoa é o registro do domínio.
type Pessoa struct {
	ID        int
	Nome      string
	Sobrenome string
}

// Modelo concentra o estado da aplicação e as operações de CRUD — a UI só
// liga eventos a estes métodos (separação domínio/interface).
type Modelo struct {
	pessoas   []Pessoa
	proximoID int
	visiveis  []Pessoa

	filtro      *juigo.State[string]
	selecionado *juigo.State[int] // ID selecionado; -1 = nenhum
	nome        *juigo.State[string]
	sobrenome   *juigo.State[string]

	lista *juigo.List[*linha]
}

// atualizaVisiveis refaz a projeção filtrada (prefixo do sobrenome, sem
// caixa) e sincroniza a lista virtualizada.
func (m *Modelo) atualizaVisiveis() {
	prefixo := strings.ToLower(m.filtro.Get())
	m.visiveis = m.visiveis[:0]
	aindaExiste := false
	for _, p := range m.pessoas {
		if strings.HasPrefix(strings.ToLower(p.Sobrenome), prefixo) {
			m.visiveis = append(m.visiveis, p)
			if p.ID == m.selecionado.Get() {
				aindaExiste = true
			}
		}
	}
	if !aindaExiste {
		m.selecionado.Set(-1)
	}
	m.lista.SetCount(len(m.visiveis))
	m.lista.Refresh()
}

// selecionar marca a pessoa e espelha os campos de edição.
func (m *Modelo) selecionar(p Pessoa) {
	m.selecionado.Set(p.ID)
	m.nome.Set(p.Nome)
	m.sobrenome.Set(p.Sobrenome)
	m.lista.Refresh()
}

// Criar insere uma pessoa com os campos atuais.
func (m *Modelo) Criar() {
	m.pessoas = append(m.pessoas, Pessoa{ID: m.proximoID, Nome: m.nome.Get(), Sobrenome: m.sobrenome.Get()})
	m.proximoID++
	m.atualizaVisiveis()
}

// Atualizar aplica os campos à pessoa selecionada.
func (m *Modelo) Atualizar() {
	for i := range m.pessoas {
		if m.pessoas[i].ID == m.selecionado.Get() {
			m.pessoas[i].Nome = m.nome.Get()
			m.pessoas[i].Sobrenome = m.sobrenome.Get()
		}
	}
	m.atualizaVisiveis()
}

// Excluir remove a pessoa selecionada.
func (m *Modelo) Excluir() {
	id := m.selecionado.Get()
	for i := range m.pessoas {
		if m.pessoas[i].ID == id {
			m.pessoas = append(m.pessoas[:i], m.pessoas[i+1:]...)
			break
		}
	}
	m.selecionado.Set(-1)
	m.atualizaVisiveis()
}

// Pessoas devolve uma cópia do cadastro (para testes e inspeção).
func (m *Modelo) Pessoas() []Pessoa {
	return append([]Pessoa(nil), m.pessoas...)
}

// linha é o widget custom de uma linha da lista: desenha "Sobrenome, Nome"
// com realce quando selecionada e trata o clique. Não é focável (regra das
// linhas de List: o pool é reciclado na rolagem).
type linha struct {
	juigo.BaseWidget
	m      *Modelo
	pessoa Pessoa
}

func (l *linha) PreferredSize() image.Point {
	th := l.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Point{X: th.InputMinWidthPx(), Y: th.LineHeight() + th.Px(4)}
}

func (l *linha) HandleEvent(ev juigo.Event) bool {
	if e, ok := ev.(juigo.MouseEvent); ok && e.Kind == juigo.MouseDown && e.Button == juigo.MouseButtonLeft {
		l.m.selecionar(l.pessoa)
		return true
	}
	return false
}

func (l *linha) Draw(dst *image.RGBA) {
	th := l.Theme()
	if th == nil {
		return
	}
	b := l.Bounds()
	if l.pessoa.ID == l.m.selecionado.Get() {
		render.FillRect(dst, b, th.Selection)
	}
	baseline := b.Min.Y + (b.Dy()-th.LineHeight())/2 + th.Ascent()
	texto := fmt.Sprintf("%s, %s", l.pessoa.Sobrenome, l.pessoa.Nome)
	th.DrawText(dst, texto, image.Pt(b.Min.X+th.PaddingPx(), baseline), th.Text)
}

// New monta o modelo e a interface.
func New() (*Modelo, juigo.Widget) {
	m := &Modelo{
		proximoID:   3,
		filtro:      juigo.NewState(""),
		selecionado: juigo.NewState(-1),
		nome:        juigo.NewState(""),
		sobrenome:   juigo.NewState(""),
		pessoas: []Pessoa{
			{ID: 0, Nome: "Hans", Sobrenome: "Emil"},
			{ID: 1, Nome: "Max", Sobrenome: "Mustermann"},
			{ID: 2, Nome: "Roman", Sobrenome: "Tisch"},
		},
	}
	m.lista = juigo.NewList(0,
		func() *linha { return &linha{m: m} },
		func(row *linha, i int) { row.pessoa = m.visiveis[i] },
	)
	m.filtro.Watch(func(string) { m.atualizaVisiveis() })

	semSelecao := juigo.Map(m.selecionado, func(id int) bool { return id < 0 })
	ui := juigo.NewVBox(
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("Filtrar sobrenome:")),
			juigo.Grow(juigo.NewInput("prefixo…").BindValue(m.filtro), 1),
		),
		juigo.Grow(juigo.NewHBox(
			juigo.Grow(juigo.NewScroll(m.lista), 1),
			juigo.NewGrid(2,
				juigo.NewText("Nome:"), juigo.Grow(juigo.NewInput("nome").BindValue(m.nome), 1),
				juigo.NewText("Sobrenome:"), juigo.Grow(juigo.NewInput("sobrenome").BindValue(m.sobrenome), 1),
			),
		), 1),
		juigo.NewHBox(
			juigo.NewButton("Criar", m.Criar),
			juigo.BindDisabled(juigo.NewButton("Atualizar", m.Atualizar), semSelecao),
			juigo.BindDisabled(juigo.NewButton("Excluir", m.Excluir), semSelecao),
		),
	).Pad(16)

	m.atualizaVisiveis()
	return m, ui
}

// UI monta a tela (conveniência para o launcher).
func UI() juigo.Widget {
	_, w := New()
	return w
}
