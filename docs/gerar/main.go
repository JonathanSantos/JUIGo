// Este programa gera as screenshots da documentação (docs/*.png) usando a
// própria renderização offscreen da lib — determinística: a mesma árvore
// produz os mesmos bytes. Rode-o após mudanças visuais:
//
//	go run ./docs/gerar
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/7guis/cells"
	"github.com/JonathanSantos/JUIGo/examples/contatos/contatos"
	contatosui "github.com/JonathanSantos/JUIGo/examples/contatos/ui"
	"github.com/JonathanSantos/JUIGo/examples/todo/tarefas"
	"github.com/JonathanSantos/JUIGo/examples/todo/ui"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/quick"
)

// vitrine monta a árvore de apresentação com os widgets principais.
func vitrine() juigo.Widget {
	volume := juigo.NewState(0.62)
	plano := juigo.NewState("pro")
	prioridade := juigo.NewState("Média")
	notif := juigo.NewCheckbox("Notificações")
	notif.SetChecked(true)
	selecionada := juigo.NewState(1)
	tabela := juigo.NewTable([]string{"Nome", "Equipe"}, 3, func(l, c int) string {
		dados := [][2]string{{"Ana", "Plataforma"}, {"Bruno", "Design"}, {"Carla", "Dados"}}
		return dados[l][c]
	}).BindSelected(selecionada)

	return juigo.NewVBox(
		juigo.NewText("JUIGo — vitrine de widgets").Center(),
		juigo.NewHBox(
			juigo.Grow(juigo.NewInput("Digite aqui…"), 1),
			juigo.NewButton("Enviar", nil),
		),
		juigo.NewHBox(
			juigo.NewInput("E-mail").BindInvalid(juigo.NewState(true)),
			juigo.Centered(juigo.NewText("E-mail inválido").Danger()),
			juigo.NewSpacer(),
			juigo.BindDisabled(juigo.NewButton("Inativo", nil), juigo.NewState(true)),
		),
		juigo.NewHBox(
			notif,
			juigo.NewSpacer(),
			juigo.NewRadio("Grátis", "free").BindValue(plano),
			juigo.NewRadio("Pro", "pro").BindValue(plano),
		),
		juigo.NewSlider(0, 1).BindValue(volume),
		juigo.NewProgressBar(0, 1).BindValue(volume),
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("Prioridade:")),
			juigo.NewDropdown("Baixa", "Média", "Alta").BindValue(prioridade),
			juigo.NewSpacer(),
		),
		tabela,
	).Pad(16).Gap(8)
}

// formulario monta o quick.Form com erros revelados (validação em cena).
func formulario() juigo.Widget {
	nome := quick.Text("Nome:").Placeholder("Nome").
		Required("Informe o nome").Min(3, "Mínimo de 3 caracteres")
	mail := quick.Text("E-mail:").Placeholder("E-mail").Email("E-mail inválido")
	idade := quick.Number("Idade:", 18).Max(120, "Confere a idade?")
	rua := quick.Text("Rua:").Placeholder("opcional")
	cidade := quick.Options("Cidade:", "São Paulo", "Recife", "Manaus")

	nome.Set("Jo")
	mail.Set("email-invalido")
	idade.Set(150)

	f := quick.Form(nome, mail, idade, quick.Section("Endereço"), rua, cidade).
		Submit("Salvar", nil)
	f.Model().Submit(nil) // revela os erros para a captura
	return juigo.NewVBox(f).Pad(16)
}

// todomvc monta o exemplo de tarefas com dados mistos.
func todomvc() juigo.Widget {
	repo := &tarefas.RepositorioJSON{Caminho: filepath.Join(os.TempDir(), "docs-todo.json")}
	lista := tarefas.NovaLista([]tarefas.Tarefa{
		{ID: 0, Titulo: "Comprar pão", Concluida: true},
		{ID: 1, Titulo: "Estudar Go"},
		{ID: 2, Titulo: "Publicar a JUIGo"},
	})
	return ui.New(lista, repo).Raiz
}

// agenda monta o exemplo mestre-detalhe com um contato selecionado.
func agenda() juigo.Widget {
	repo := &contatos.RepositorioJSON{Caminho: filepath.Join(os.TempDir(), "docs-contatos.json")}
	a := contatos.NovaAgenda(nil)
	ana := a.Adicionar("Ana Lima")
	a.Atualizar(contatos.Contato{
		ID: ana.ID, Nome: "Ana Lima", Email: "ana@plataforma.dev",
		Telefone: "(11) 91234-0000", Empresa: "Plataforma",
		Notas: "Prefere reuniões pela manhã.",
	})
	a.Adicionar("Bruno Reis")
	carla := a.Adicionar("Carla Dias")
	a.AlternarFavorito(carla.ID)
	v := contatosui.New(a, repo)
	v.Seleciona(ana.ID)
	return v.Raiz
}

// planilha monta o Cells do 7GUIs com fórmulas em cena.
func planilha() juigo.Widget {
	a := cells.New()
	a.M.Definir("A1", "10")
	a.M.Definir("B1", "7")
	a.M.Definir("C1", "=A1+B1")
	a.M.Definir("A2", "=A2")
	a.M.Definir("B3", "texto")
	return a.Raiz
}

func salva(nome string, raiz juigo.Widget, th *juigo.Theme, w, h int) {
	caminho := filepath.Join("docs", nome)
	if err := offscreen.SavePNG(caminho, offscreen.Render(raiz, th, w, h)); err != nil {
		log.Fatalf("gerar %s: %v", caminho, err)
	}
	log.Printf("gerado %s", caminho)
}

func main() {
	claro, err := juigo.DefaultTheme()
	if err != nil {
		log.Fatal(err)
	}
	escuro, err := juigo.DarkTheme()
	if err != nil {
		log.Fatal(err)
	}

	salva("vitrine-claro.png", vitrine(), claro, 440, 460)
	salva("vitrine-escuro.png", vitrine(), escuro, 440, 460)
	salva("quick-form.png", formulario(), claro, 440, 440)
	salva("todomvc.png", todomvc(), claro, 440, 400)
	salva("contatos.png", agenda(), claro, 720, 420)
	salva("cells.png", planilha(), claro, 620, 400)
}
