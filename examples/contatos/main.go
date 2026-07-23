// Contatos — o exemplo MESTRE-DETALHE do JUIGo, em três camadas: contatos/
// é o domínio puro (agregado + repositório JSON), ui/ é a Vista que o
// projeta em widgets, e este main é a raiz de composição.
//
// À esquerda, a busca e a tabela de contatos (♥ = favorito); à direita, o
// detalhe em abas — Dados (formulário validado, Salvar) e Notas. Botão
// direito numa linha abre o menu de contexto (favoritar, excluir com
// confirmação); as gravações confirmam com um toast. Tudo persiste num
// JSON no diretório de configuração do usuário.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/contatos/contatos"
	"github.com/JonathanSantos/JUIGo/examples/contatos/ui"
)

func main() {
	caminho := "juigo-contatos.json"
	if dir, err := os.UserConfigDir(); err == nil {
		caminho = filepath.Join(dir, "juigo-contatos.json")
	}
	repo := &contatos.RepositorioJSON{Caminho: caminho}
	itens, err := repo.Carregar()
	if err != nil {
		log.Fatalf("contatos: carregar %s: %v", caminho, err)
	}

	vista := ui.New(contatos.NovaAgenda(itens), repo)
	app, err := juigo.New("Contatos — mestre-detalhe em JUIGo", 760, 480)
	if err != nil {
		log.Fatal(err)
	}
	app.SetRoot(vista.Raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
