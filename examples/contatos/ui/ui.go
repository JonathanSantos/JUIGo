// Package ui é a camada de interface do exemplo Contatos: mestre-detalhe
// clássico. À esquerda, busca + tabela (o MESTRE); à direita, o DETALHE em
// abas (Dados como quick.Form validado, Notas como TextArea), desabilitado
// sem seleção. O fluxo é o mesmo ciclo único do TodoMVC:
//
//	evento → aplica(mutação) → domínio muda → repositório salva → reprojeta()
//
// A seleção é preservada por IDENTIDADE (ID) através de filtro e
// reordenação; botão direito numa linha abre quick.Menu (favoritar,
// excluir com quick.Confirm) e as gravações confirmam com quick.Toast.
package ui

import (
	"image"
	"log"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/contatos/contatos"
	"github.com/JonathanSantos/JUIGo/quick"
)

// Vista é o view-model: estados derivados do domínio e a raiz da árvore.
type Vista struct {
	agenda *contatos.Agenda
	repo   contatos.Repositorio

	busca       *juigo.State[string]
	selecionado *juigo.State[int] // índice na projeção corrente; -1 = nenhum
	// selecionadoID é a IDENTIDADE selecionada — sobrevive a filtro e
	// reordenação, que mudam os índices.
	selecionadoID int
	visiveis      []contatos.Contato
	semSelecao    *juigo.State[bool]

	tabela                     *tabelaComMenu
	nome, email, fone, empresa *quick.TextField
	notas                      *juigo.State[string]

	// Raiz é a árvore de widgets pronta para App.SetRoot.
	Raiz juigo.Widget
}

// New monta a Vista sobre o domínio e o repositório dados.
func New(agenda *contatos.Agenda, repo contatos.Repositorio) *Vista {
	v := &Vista{
		agenda:        agenda,
		repo:          repo,
		busca:         juigo.NewState(""),
		selecionado:   juigo.NewState(-1),
		selecionadoID: -1,
		semSelecao:    juigo.NewState(true),
		notas:         juigo.NewState(""),
	}

	// Mestre: busca + tabela com menu de contexto.
	v.tabela = &tabelaComMenu{
		Table: juigo.NewTable([]string{"Nome", "Empresa"}, 0, v.celula).
			BindSelected(v.selecionado).
			Widths(170, 110),
	}
	v.tabela.aoMenu = v.abreMenu
	busca := juigo.NewInput("Buscar…").BindValue(v.busca)
	v.busca.Watch(func(string) { v.reprojeta() })
	adicionar := juigo.NewButton("Adicionar", v.adicionar)

	// Detalhe: abas Dados (formulário validado) e Notas.
	v.nome = quick.Text("Nome:").Placeholder("nome").Required("Informe o nome")
	v.email = quick.Text("E-mail:").Placeholder("email").Email("E-mail inválido")
	v.fone = quick.Text("Telefone:").Placeholder("telefone")
	v.empresa = quick.Text("Empresa:").Placeholder("empresa")
	formulario := quick.Form(v.nome, v.email, v.fone, v.empresa).Submit("Salvar", v.salvar)
	notas := juigo.NewTextArea("Anotações sobre o contato…").BindValue(v.notas)
	abas := juigo.NewTabs().
		Add("Dados", formulario).
		Add("Notas", juigo.NewVBox(juigo.Grow(notas, 1)))
	detalhe := juigo.BindDisabled(abas, v.semSelecao)

	v.selecionado.Watch(func(i int) {
		if i >= 0 && i < len(v.visiveis) {
			v.selecionadoID = v.visiveis[i].ID
		} else {
			v.selecionadoID = -1
		}
		v.carregaDetalhe()
	})

	v.Raiz = juigo.NewHBox(
		juigo.NewVBox(
			juigo.NewHBox(juigo.Grow(busca, 1), adicionar).Gap(8),
			juigo.Grow(juigo.NewScroll(v.tabela), 1),
		).Gap(8),
		juigo.Grow(detalhe, 1),
	).Pad(12).Gap(16)

	v.reprojeta()
	return v
}

// celula projeta a célula da tabela a partir da projeção corrente; o nome
// ganha ♥ quando favorito.
func (v *Vista) celula(linha, coluna int) string {
	if linha < 0 || linha >= len(v.visiveis) {
		return ""
	}
	c := v.visiveis[linha]
	if coluna == 0 {
		if c.Favorito {
			return "♥ " + c.Nome
		}
		return c.Nome
	}
	return c.Empresa
}

// aplica executa a mutação no domínio, persiste e reprojeta — todo evento
// que muda dados passa por aqui (fluxo único).
func (v *Vista) aplica(muta func()) {
	muta()
	if err := v.repo.Salvar(v.agenda.Todos()); err != nil {
		log.Printf("contatos: falha ao salvar: %v", err)
	}
	v.reprojeta()
}

// reprojeta refiltra a agenda e restaura a seleção pela identidade: o mesmo
// contato continua selecionado ainda que o índice tenha mudado; filtrado
// para fora, a seleção limpa.
func (v *Vista) reprojeta() {
	id := v.selecionadoID
	v.visiveis = v.agenda.Filtrar(v.busca.Get())
	idx := -1
	for i, c := range v.visiveis {
		if c.ID == id {
			idx = i
			break
		}
	}
	v.tabela.SetCount(len(v.visiveis))
	if v.selecionado.Get() != idx {
		v.selecionado.Set(idx) // o watcher rederiva selecionadoID e o detalhe
	} else {
		v.selecionadoID = id
		v.carregaDetalhe()
	}
	v.tabela.Refresh()
}

// Seleciona seleciona o contato com o ID dado pela identidade (-1 limpa) —
// o caminho programático usado por atalhos e pela captura de documentação.
func (v *Vista) Seleciona(id int) {
	v.selecionadoID = id
	v.reprojeta()
}

// carregaDetalhe projeta o contato selecionado nos campos do detalhe (ou os
// limpa e desabilita as abas sem seleção).
func (v *Vista) carregaDetalhe() {
	i := v.selecionado.Get()
	if i < 0 || i >= len(v.visiveis) {
		v.semSelecao.Set(true)
		v.nome.Set("")
		v.email.Set("")
		v.fone.Set("")
		v.empresa.Set("")
		v.notas.Set("")
		return
	}
	c := v.visiveis[i]
	v.semSelecao.Set(false)
	v.nome.Set(c.Nome)
	v.email.Set(c.Email)
	v.fone.Set(c.Telefone)
	v.empresa.Set(c.Empresa)
	v.notas.Set(c.Notas)
}

// salvar aplica os campos do detalhe ao contato selecionado — o Submit do
// formulário já validou.
func (v *Vista) salvar() {
	i := v.selecionado.Get()
	if i < 0 || i >= len(v.visiveis) {
		return
	}
	c := v.visiveis[i]
	c.Nome = v.nome.Value()
	c.Email = v.email.Value()
	c.Telefone = v.fone.Value()
	c.Empresa = v.empresa.Value()
	c.Notas = v.notas.Get()
	v.aplica(func() { v.agenda.Atualizar(c) })
	quick.Toast("Contato salvo")
}

// adicionar cria um contato, o seleciona (limpando a busca para ele ficar
// visível) e abre o campo Nome focado, pronto para digitar.
func (v *Vista) adicionar() {
	v.busca.Set("")
	v.aplica(func() {
		novo := v.agenda.Adicionar("")
		v.selecionadoID = novo.ID
	})
	juigo.Focus(v.nome.Control())
}

// abreMenu seleciona a linha e abre o menu de contexto do contato.
func (v *Vista) abreMenu(linha int, pos image.Point) {
	if linha < 0 || linha >= len(v.visiveis) {
		return
	}
	v.selecionado.Set(linha)
	c := v.visiveis[linha]
	favorito := "Favoritar"
	if c.Favorito {
		favorito = "Desfavoritar"
	}
	quick.Menu(pos,
		quick.Item(favorito, func() {
			v.aplica(func() { v.agenda.AlternarFavorito(c.ID) })
		}),
		quick.Item("Excluir…", func() {
			quick.Confirm("Excluir "+c.Nome+"?", func(ok bool) {
				if !ok {
					return
				}
				v.aplica(func() { v.agenda.Remover(c.ID) })
				quick.Toast("Contato excluído")
			})
		}),
	)
}

// tabelaComMenu decora a Table com um menu de contexto: intercepta o botão
// direito e delega o resto — o padrão para acrescentar uma interação a um
// widget pronto.
type tabelaComMenu struct {
	*juigo.Table
	aoMenu func(linha int, pos image.Point)
}

// HandleEvent abre o menu no botão direito sobre uma linha; os demais
// eventos seguem para a Table.
func (t *tabelaComMenu) HandleEvent(ev juigo.Event) bool {
	if e, ok := ev.(juigo.MouseEvent); ok && e.Kind == juigo.MouseDown && e.Button == juigo.MouseButtonRight {
		if linha := t.linhaEm(e.Pos); linha >= 0 && t.aoMenu != nil {
			t.aoMenu(linha, e.Pos)
		}
		return true
	}
	return t.Table.HandleEvent(ev)
}

// linhaEm devolve a linha sob o ponto, ou -1.
func (t *tabelaComMenu) linhaEm(p image.Point) int {
	for i := 0; i < t.Count(); i++ {
		if p.In(t.RowRect(i)) {
			return i
		}
	}
	return -1
}
