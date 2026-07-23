// Package contatos é o DOMÍNIO do exemplo mestre-detalhe: a Agenda com as
// operações de negócio e a interface de persistência. Não importa juigo —
// a separação é deliberada: o domínio testa-se sozinho e a UI é só uma
// projeção dele.
package contatos

import (
	"sort"
	"strings"
)

// Contato é o registro do domínio.
type Contato struct {
	ID       int    `json:"id"`
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Telefone string `json:"telefone"`
	Empresa  string `json:"empresa"`
	Notas    string `json:"notas"`
	Favorito bool   `json:"favorito"`
}

// Agenda é o agregado: os contatos e as operações sobre eles.
type Agenda struct {
	itens     []Contato
	proximoID int
}

// NovaAgenda cria o agregado a partir dos contatos persistidos.
func NovaAgenda(itens []Contato) *Agenda {
	a := &Agenda{itens: append([]Contato(nil), itens...)}
	for _, c := range a.itens {
		if c.ID >= a.proximoID {
			a.proximoID = c.ID + 1
		}
	}
	return a
}

// Adicionar insere um contato com o nome dado (espaços aparados; vazio vira
// "Novo contato") e o devolve com o ID atribuído.
func (a *Agenda) Adicionar(nome string) Contato {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		nome = "Novo contato"
	}
	c := Contato{ID: a.proximoID, Nome: nome}
	a.itens = append(a.itens, c)
	a.proximoID++
	return c
}

// Atualizar substitui o contato de mesmo ID pelo dado (nome aparado).
// Devolve false — sem mudar nada — se o ID não existe ou o nome fica vazio.
func (a *Agenda) Atualizar(c Contato) bool {
	c.Nome = strings.TrimSpace(c.Nome)
	if c.Nome == "" {
		return false
	}
	for i := range a.itens {
		if a.itens[i].ID == c.ID {
			a.itens[i] = c
			return true
		}
	}
	return false
}

// Remover exclui o contato com o ID dado; inexistente é ignorado.
func (a *Agenda) Remover(id int) {
	for i := range a.itens {
		if a.itens[i].ID == id {
			a.itens = append(a.itens[:i], a.itens[i+1:]...)
			return
		}
	}
}

// AlternarFavorito inverte o favorito do contato com o ID dado.
func (a *Agenda) AlternarFavorito(id int) {
	for i := range a.itens {
		if a.itens[i].ID == id {
			a.itens[i].Favorito = !a.itens[i].Favorito
		}
	}
}

// Buscar devolve o contato com o ID dado.
func (a *Agenda) Buscar(id int) (Contato, bool) {
	for _, c := range a.itens {
		if c.ID == id {
			return c, true
		}
	}
	return Contato{}, false
}

// Todos devolve uma cópia ordenada dos contatos: favoritos primeiro, depois
// por nome (sem diferenciar maiúsculas).
func (a *Agenda) Todos() []Contato {
	out := append([]Contato(nil), a.itens...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Favorito != out[j].Favorito {
			return out[i].Favorito
		}
		return strings.ToLower(out[i].Nome) < strings.ToLower(out[j].Nome)
	})
	return out
}

// Filtrar devolve os contatos (na ordem de Todos) cujo nome, e-mail ou
// empresa contém o termo, sem diferenciar maiúsculas. Termo vazio devolve
// todos.
func (a *Agenda) Filtrar(termo string) []Contato {
	termo = strings.ToLower(strings.TrimSpace(termo))
	todos := a.Todos()
	if termo == "" {
		return todos
	}
	out := todos[:0]
	for _, c := range todos {
		if strings.Contains(strings.ToLower(c.Nome), termo) ||
			strings.Contains(strings.ToLower(c.Email), termo) ||
			strings.Contains(strings.ToLower(c.Empresa), termo) {
			out = append(out, c)
		}
	}
	return out
}
