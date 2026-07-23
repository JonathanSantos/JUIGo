// Package tarefas é o DOMÍNIO do TodoMVC: o agregado Lista com as
// operações de negócio e a interface de persistência. Não importa juigo —
// a separação é deliberada: o domínio testa-se sozinho e a UI é só uma
// projeção dele.
package tarefas

import "strings"

// Tarefa é o registro do domínio.
type Tarefa struct {
	ID        int    `json:"id"`
	Titulo    string `json:"titulo"`
	Concluida bool   `json:"concluida"`
}

// Filtro identifica as projeções clássicas do TodoMVC.
type Filtro string

const (
	FiltroTodas      Filtro = "Todas"
	FiltroAtivas     Filtro = "Ativas"
	FiltroConcluidas Filtro = "Concluídas"
)

// Lista é o agregado: as tarefas e as operações sobre elas.
type Lista struct {
	itens     []Tarefa
	proximoID int
}

// NovaLista cria o agregado a partir das tarefas persistidas.
func NovaLista(itens []Tarefa) *Lista {
	l := &Lista{itens: append([]Tarefa(nil), itens...)}
	for _, t := range l.itens {
		if t.ID >= l.proximoID {
			l.proximoID = t.ID + 1
		}
	}
	return l
}

// Adicionar insere uma tarefa ativa com o título dado (espaços aparados);
// título vazio é ignorado. Devolve true se inseriu.
func (l *Lista) Adicionar(titulo string) bool {
	titulo = strings.TrimSpace(titulo)
	if titulo == "" {
		return false
	}
	l.itens = append(l.itens, Tarefa{ID: l.proximoID, Titulo: titulo})
	l.proximoID++
	return true
}

// Alternar inverte a conclusão da tarefa.
func (l *Lista) Alternar(id int) {
	for i := range l.itens {
		if l.itens[i].ID == id {
			l.itens[i].Concluida = !l.itens[i].Concluida
		}
	}
}

// Renomear troca o título da tarefa; título vazio a REMOVE (semântica
// TodoMVC: editar para vazio apaga).
func (l *Lista) Renomear(id int, titulo string) {
	titulo = strings.TrimSpace(titulo)
	if titulo == "" {
		l.Remover(id)
		return
	}
	for i := range l.itens {
		if l.itens[i].ID == id {
			l.itens[i].Titulo = titulo
		}
	}
}

// Remover apaga a tarefa.
func (l *Lista) Remover(id int) {
	for i := range l.itens {
		if l.itens[i].ID == id {
			l.itens = append(l.itens[:i], l.itens[i+1:]...)
			return
		}
	}
}

// MarcarTodas define a conclusão de todas as tarefas de uma vez.
func (l *Lista) MarcarTodas(concluidas bool) {
	for i := range l.itens {
		l.itens[i].Concluida = concluidas
	}
}

// LimparConcluidas remove todas as tarefas concluídas.
func (l *Lista) LimparConcluidas() {
	vivas := l.itens[:0]
	for _, t := range l.itens {
		if !t.Concluida {
			vivas = append(vivas, t)
		}
	}
	l.itens = vivas
}

// Filtradas devolve a projeção do filtro (cópia; a ordem de inserção é
// preservada).
func (l *Lista) Filtradas(f Filtro) []Tarefa {
	var out []Tarefa
	for _, t := range l.itens {
		switch f {
		case FiltroAtivas:
			if t.Concluida {
				continue
			}
		case FiltroConcluidas:
			if !t.Concluida {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

// Todas devolve uma cópia de todas as tarefas (para persistir).
func (l *Lista) Todas() []Tarefa {
	return append([]Tarefa(nil), l.itens...)
}

// Restantes conta as tarefas ativas.
func (l *Lista) Restantes() int {
	n := 0
	for _, t := range l.itens {
		if !t.Concluida {
			n++
		}
	}
	return n
}

// TemConcluidas informa se há tarefas concluídas (habilita o "limpar").
func (l *Lista) TemConcluidas() bool {
	for _, t := range l.itens {
		if t.Concluida {
			return true
		}
	}
	return false
}

// TodasConcluidas informa se TODAS as tarefas estão concluídas (estado do
// "marcar todas"); false com a lista vazia.
func (l *Lista) TodasConcluidas() bool {
	return len(l.itens) > 0 && l.Restantes() == 0
}
