// O TodoMVC em JUIGo — o exercício clássico de lista dinâmica, em três
// camadas: tarefas/ é o domínio puro (agregado + repositório JSON), ui/ é a
// Vista que o projeta em widgets, e este main é a raiz de composição que
// liga as peças (carrega, monta, roda).
//
// Funciona como o TodoMVC canônico: Enter adiciona; o checkbox conclui
// (título riscado); DUPLO CLIQUE no título edita inline (Enter/clicar fora
// confirmam; vazio remove); filtros Todas/Ativas/Concluídas; "marcar
// todas"; "Limpar concluídas". Tudo persiste num JSON no diretório de
// configuração do usuário.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/todo/tarefas"
	"github.com/JonathanSantos/JUIGo/examples/todo/ui"
)

func main() {
	caminho := "juigo-todo.json"
	if dir, err := os.UserConfigDir(); err == nil {
		caminho = filepath.Join(dir, "juigo-todo.json")
	}
	repo := &tarefas.RepositorioJSON{Caminho: caminho}
	itens, err := repo.Carregar()
	if err != nil {
		log.Fatalf("todo: carregar %s: %v", caminho, err)
	}

	vista := ui.New(tarefas.NovaLista(itens), repo)
	app, err := juigo.New("Tarefas — TodoMVC em JUIGo", 480, 560)
	if err != nil {
		log.Fatal(err)
	}
	app.SetRoot(vista.Raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
