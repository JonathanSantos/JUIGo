package tarefas

import (
	"path/filepath"
	"testing"
)

// O domínio testa-se sem UI nenhuma — o dividendo da separação de camadas.
func TestDominio(t *testing.T) {
	l := NovaLista(nil)
	if !l.Adicionar("  pão  ") {
		t.Fatal("Adicionar deveria aceitar título com espaços aparados")
	}
	if l.Adicionar("   ") {
		t.Fatal("Adicionar deveria recusar título vazio")
	}
	l.Adicionar("leite")
	if l.Restantes() != 2 {
		t.Fatalf("restantes = %d", l.Restantes())
	}

	pao := l.Todas()[0]
	if pao.Titulo != "pão" {
		t.Fatalf("título deveria estar aparado; got %q", pao.Titulo)
	}
	l.Alternar(pao.ID)
	if l.Restantes() != 1 || !l.TemConcluidas() {
		t.Fatal("Alternar deveria concluir o pão")
	}
	if got := l.Filtradas(FiltroConcluidas); len(got) != 1 || got[0].ID != pao.ID {
		t.Fatalf("Filtradas(Concluídas) errada: %v", got)
	}

	// Renomear para vazio REMOVE (semântica TodoMVC).
	l.Renomear(pao.ID, "   ")
	if len(l.Todas()) != 1 {
		t.Fatal("renomear para vazio deveria remover")
	}

	l.MarcarTodas(true)
	if !l.TodasConcluidas() {
		t.Fatal("MarcarTodas deveria concluir tudo")
	}
	l.LimparConcluidas()
	if len(l.Todas()) != 0 {
		t.Fatal("LimparConcluidas deveria esvaziar")
	}

	// IDs não se repetem após recarregar (proximoID vem do maior ID).
	l2 := NovaLista([]Tarefa{{ID: 7, Titulo: "x"}})
	l2.Adicionar("y")
	if l2.Todas()[1].ID != 8 {
		t.Fatalf("ID novo deveria ser 8; got %d", l2.Todas()[1].ID)
	}
}

func TestRepositorioJSON(t *testing.T) {
	repo := &RepositorioJSON{Caminho: filepath.Join(t.TempDir(), "todo.json")}

	// Primeira execução: arquivo inexistente = lista vazia, sem erro.
	itens, err := repo.Carregar()
	if err != nil || itens != nil {
		t.Fatalf("carregar sem arquivo deveria ser vazio; got %v %v", itens, err)
	}

	quer := []Tarefa{{ID: 0, Titulo: "pão", Concluida: true}, {ID: 1, Titulo: "leite"}}
	if err := repo.Salvar(quer); err != nil {
		t.Fatal(err)
	}
	itens, err = repo.Carregar()
	if err != nil || len(itens) != 2 || itens[0].Titulo != "pão" || !itens[0].Concluida {
		t.Fatalf("ida e volta divergiu: %v %v", itens, err)
	}
}
