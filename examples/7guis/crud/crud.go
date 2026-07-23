// Package crud é o 7GUIs nº 5: cadastro com lista filtrável, seleção e
// criar/atualizar/excluir. O modelo de domínio (Pessoa) vive separado da
// interface, e a lista é uma Table da lib — colunas Nome/Sobrenome,
// cabeçalho fixo e seleção nativa via BindSelected (o índice selecionado é
// um State como outro qualquer: os botões prendem-se a ele com
// BindDisabled e os campos o espelham com Watch).
package crud

import (
	"strings"

	"juigo"
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
	selecionada *juigo.State[int] // índice em visiveis; -1 = nenhuma
	nome        *juigo.State[string]
	sobrenome   *juigo.State[string]

	tabela *juigo.Table
}

// atualizaVisiveis refaz a projeção filtrada (prefixo do sobrenome, sem
// caixa), preservando a seleção pela IDENTIDADE da pessoa quando ela
// continua visível.
func (m *Modelo) atualizaVisiveis() {
	idSelecionado := -1
	if i := m.selecionada.Get(); i >= 0 && i < len(m.visiveis) {
		idSelecionado = m.visiveis[i].ID
	}
	prefixo := strings.ToLower(m.filtro.Get())
	m.visiveis = m.visiveis[:0]
	novoIdx := -1
	for _, p := range m.pessoas {
		if strings.HasPrefix(strings.ToLower(p.Sobrenome), prefixo) {
			if p.ID == idSelecionado {
				novoIdx = len(m.visiveis)
			}
			m.visiveis = append(m.visiveis, p)
		}
	}
	m.tabela.SetCount(len(m.visiveis))
	if m.selecionada.Get() != novoIdx {
		m.selecionada.Set(novoIdx)
	}
	m.tabela.Refresh()
}

// Criar insere uma pessoa com os campos atuais.
func (m *Modelo) Criar() {
	m.pessoas = append(m.pessoas, Pessoa{ID: m.proximoID, Nome: m.nome.Get(), Sobrenome: m.sobrenome.Get()})
	m.proximoID++
	m.atualizaVisiveis()
}

// Atualizar aplica os campos à pessoa selecionada.
func (m *Modelo) Atualizar() {
	i := m.selecionada.Get()
	if i < 0 || i >= len(m.visiveis) {
		return
	}
	id := m.visiveis[i].ID
	for j := range m.pessoas {
		if m.pessoas[j].ID == id {
			m.pessoas[j].Nome = m.nome.Get()
			m.pessoas[j].Sobrenome = m.sobrenome.Get()
		}
	}
	m.atualizaVisiveis()
}

// Excluir remove a pessoa selecionada.
func (m *Modelo) Excluir() {
	i := m.selecionada.Get()
	if i < 0 || i >= len(m.visiveis) {
		return
	}
	id := m.visiveis[i].ID
	for j := range m.pessoas {
		if m.pessoas[j].ID == id {
			m.pessoas = append(m.pessoas[:j], m.pessoas[j+1:]...)
			break
		}
	}
	m.selecionada.Set(-1)
	m.atualizaVisiveis()
}

// Pessoas devolve uma cópia do cadastro (para testes e inspeção).
func (m *Modelo) Pessoas() []Pessoa {
	return append([]Pessoa(nil), m.pessoas...)
}

// New monta o modelo e a interface.
func New() (*Modelo, juigo.Widget) {
	m := &Modelo{
		proximoID:   3,
		filtro:      juigo.NewState(""),
		selecionada: juigo.NewState(-1),
		nome:        juigo.NewState(""),
		sobrenome:   juigo.NewState(""),
		pessoas: []Pessoa{
			{ID: 0, Nome: "Hans", Sobrenome: "Emil"},
			{ID: 1, Nome: "Max", Sobrenome: "Mustermann"},
			{ID: 2, Nome: "Roman", Sobrenome: "Tisch"},
		},
	}
	m.tabela = juigo.NewTable([]string{"Nome", "Sobrenome"}, 0, func(linha, coluna int) string {
		p := m.visiveis[linha]
		if coluna == 0 {
			return p.Nome
		}
		return p.Sobrenome
	}).BindSelected(m.selecionada)

	m.filtro.Watch(func(string) { m.atualizaVisiveis() })
	// Selecionar espelha os campos de edição.
	m.selecionada.Watch(func(i int) {
		if i >= 0 && i < len(m.visiveis) {
			m.nome.Set(m.visiveis[i].Nome)
			m.sobrenome.Set(m.visiveis[i].Sobrenome)
		}
	})

	semSelecao := juigo.Map(m.selecionada, func(i int) bool { return i < 0 })
	ui := juigo.NewVBox(
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("Filtrar sobrenome:")),
			juigo.Grow(juigo.NewInput("prefixo…").BindValue(m.filtro), 1),
		),
		juigo.Grow(juigo.NewHBox(
			juigo.Grow(juigo.NewScroll(m.tabela), 1),
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
