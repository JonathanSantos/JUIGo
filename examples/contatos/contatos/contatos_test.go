package contatos

import "testing"

func agendaTeste() *Agenda {
	a := NovaAgenda(nil)
	ana := a.Adicionar("Ana")
	a.Atualizar(Contato{ID: ana.ID, Nome: "Ana", Email: "ana@plataforma.dev", Empresa: "Plataforma"})
	a.Adicionar("bruno")
	a.Adicionar("Carla")
	return a
}

func TestAdicionarAtualizarRemover(t *testing.T) {
	a := NovaAgenda(nil)
	c := a.Adicionar("  ")
	if c.Nome != "Novo contato" || c.ID != 0 {
		t.Fatalf("Adicionar vazio deveria virar 'Novo contato' com ID 0; veio %+v", c)
	}
	d := a.Adicionar("Dani")
	if d.ID != 1 {
		t.Fatalf("IDs deveriam ser sequenciais; veio %d", d.ID)
	}

	if a.Atualizar(Contato{ID: d.ID, Nome: "   "}) {
		t.Fatal("Atualizar com nome vazio deveria ser recusado")
	}
	if !a.Atualizar(Contato{ID: d.ID, Nome: "Daniela", Email: "d@x.dev"}) {
		t.Fatal("Atualizar deveria aceitar o contato válido")
	}
	if got, _ := a.Buscar(d.ID); got.Nome != "Daniela" || got.Email != "d@x.dev" {
		t.Fatalf("Buscar após Atualizar: %+v", got)
	}
	if a.Atualizar(Contato{ID: 99, Nome: "Z"}) {
		t.Fatal("Atualizar de ID inexistente deveria devolver false")
	}

	a.Remover(d.ID)
	if _, ok := a.Buscar(d.ID); ok {
		t.Fatal("Remover deveria excluir o contato")
	}
	a.Remover(99) // inexistente: ignorado, sem pânico
}

func TestOrdenacaoEFavoritos(t *testing.T) {
	a := agendaTeste()
	todos := a.Todos()
	if todos[0].Nome != "Ana" || todos[1].Nome != "bruno" || todos[2].Nome != "Carla" {
		t.Fatalf("ordenação por nome sem diferenciar maiúsculas; veio %v", nomes(todos))
	}

	a.AlternarFavorito(todos[2].ID) // Carla
	todos = a.Todos()
	if todos[0].Nome != "Carla" || !todos[0].Favorito {
		t.Fatalf("favoritos deveriam vir primeiro; veio %v", nomes(todos))
	}
	a.AlternarFavorito(todos[0].ID)
	if a.Todos()[0].Nome != "Ana" {
		t.Fatal("desfavoritar deveria voltar à ordem por nome")
	}
}

func TestFiltrar(t *testing.T) {
	a := agendaTeste()
	if got := a.Filtrar("BRU"); len(got) != 1 || got[0].Nome != "bruno" {
		t.Fatalf("filtro por nome sem diferenciar maiúsculas; veio %v", nomes(got))
	}
	if got := a.Filtrar("plataforma"); len(got) != 1 || got[0].Nome != "Ana" {
		t.Fatalf("filtro deveria casar empresa; veio %v", nomes(got))
	}
	if got := a.Filtrar("ana@"); len(got) != 1 {
		t.Fatalf("filtro deveria casar e-mail; veio %v", nomes(got))
	}
	if got := a.Filtrar("  "); len(got) != 3 {
		t.Fatalf("filtro vazio deveria devolver todos; veio %v", nomes(got))
	}
	if got := a.Filtrar("xyz"); len(got) != 0 {
		t.Fatalf("filtro sem resultado deveria vir vazio; veio %v", nomes(got))
	}
}

func nomes(cs []Contato) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Nome
	}
	return out
}
