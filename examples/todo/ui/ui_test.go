package ui

import (
	"path/filepath"
	"testing"
	"time"

	"juigo"
	"juigo/examples/todo/tarefas"
	"juigo/uitest"
	"juigo/widget"
)

// monta cria uma Vista limpa com repositório JSON num diretório temporário.
func monta(t *testing.T) (*Vista, *tarefas.RepositorioJSON, *uitest.Harness) {
	t.Helper()
	repo := &tarefas.RepositorioJSON{Caminho: filepath.Join(t.TempDir(), "todo.json")}
	v := New(tarefas.NovaLista(nil), repo)
	h := uitest.New(t, v.Raiz, 480, 480)
	return v, repo, h
}

// adiciona digita o título no campo principal e envia com Enter.
func adiciona(h *uitest.Harness, titulo string) {
	h.Click(uitest.Placeholder("O que precisa ser feito?"))
	h.Type(titulo)
	h.Key(juigo.KeyEnter)
}

// titulos devolve os textos das linhas visíveis, em ordem.
func titulos(h *uitest.Harness) []string {
	var out []string
	for _, w := range h.FindAll(uitest.OfType[*titulo]()) {
		out = append(out, w.(*titulo).texto)
	}
	return out
}

func TestAdicionarConcluirFiltrarPersistir(t *testing.T) {
	_, repo, h := monta(t)

	adiciona(h, "Comprar pão")
	adiciona(h, "Estudar Go")
	if h.Find(uitest.Text("2 itens restantes")) == nil {
		t.Fatal("o resumo deveria contar 2 itens")
	}
	if got := titulos(h); len(got) != 2 || got[0] != "Comprar pão" {
		t.Fatalf("linhas erradas: %v", got)
	}

	// Conclui a primeira (checkboxes: [0] é o marcar-todas do topo).
	caixas := h.FindAll(uitest.OfType[*juigo.Checkbox]())
	if len(caixas) != 3 {
		t.Fatalf("deveriam ser 3 checkboxes (todas + 2 linhas); got %d", len(caixas))
	}
	h.ClickAt(caixas[1].Bounds().Min.Add(caixas[1].Bounds().Size().Div(2)))
	if h.Find(uitest.Text("1 item restante")) == nil {
		t.Fatal("concluir uma deveria deixar 1 item restante")
	}

	// Filtros projetam o domínio.
	h.Click(uitest.Text("Concluídas"))
	if got := titulos(h); len(got) != 1 || got[0] != "Comprar pão" {
		t.Fatalf("filtro Concluídas deveria mostrar só o pão; got %v", got)
	}
	h.Click(uitest.Text("Ativas"))
	if got := titulos(h); len(got) != 1 || got[0] != "Estudar Go" {
		t.Fatalf("filtro Ativas deveria mostrar só o estudo; got %v", got)
	}
	h.Click(uitest.Text("Todas"))

	// Tudo persistiu no JSON a cada mutação.
	itens, err := repo.Carregar()
	if err != nil || len(itens) != 2 {
		t.Fatalf("o repositório deveria ter 2 tarefas; got %v %v", itens, err)
	}
	if !itens[0].Concluida || itens[1].Concluida {
		t.Fatalf("só a primeira deveria estar concluída; got %+v", itens)
	}

	// Recarregar do disco reconstrói o mesmo estado (ciclo completo).
	recarregada := tarefas.NovaLista(itens)
	if recarregada.Restantes() != 1 {
		t.Fatal("a lista recarregada deveria ter 1 restante")
	}
}

func TestEditarComDuploClique(t *testing.T) {
	_, _, h := monta(t)
	adiciona(h, "Erro de digitassão")

	// Clique ÚNICO não edita (a janela de duplo clique expira no relógio
	// virtual).
	rotulo := h.Find(uitest.OfType[*titulo]())
	centro := rotulo.Bounds().Min.Add(rotulo.Bounds().Size().Div(2))
	h.ClickAt(centro)
	h.Advance(500 * time.Millisecond)
	if n := len(h.FindAll(uitest.OfType[*juigo.Input]())); n != 1 {
		t.Fatalf("clique único não deveria abrir edição; inputs = %d", n)
	}

	// Duplo clique abre a edição JÁ FOCADA.
	h.ClickAt(centro)
	h.ClickAt(centro)
	inputs := h.FindAll(uitest.OfType[*juigo.Input]())
	if len(inputs) != 2 {
		t.Fatalf("duplo clique deveria abrir o campo de edição; inputs = %d", len(inputs))
	}
	campo := inputs[1].(*juigo.Input)
	if h.Focused() != widget.Widget(campo) {
		t.Fatal("o campo de edição deveria abrir focado")
	}
	if campo.Text() != "Erro de digitassão" {
		t.Fatalf("o campo deveria abrir com o título atual; got %q", campo.Text())
	}

	// Enter confirma a edição.
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Erro de digitação")
	h.Key(juigo.KeyEnter)
	if got := titulos(h); len(got) != 1 || got[0] != "Erro de digitação" {
		t.Fatalf("Enter deveria confirmar o novo título; got %v", got)
	}
	if n := len(h.FindAll(uitest.OfType[*juigo.Input]())); n != 1 {
		t.Fatal("a edição deveria fechar após o Enter")
	}

	// Editar para VAZIO remove a tarefa (semântica TodoMVC).
	rotulo = h.Find(uitest.OfType[*titulo]())
	centro = rotulo.Bounds().Min.Add(rotulo.Bounds().Size().Div(2))
	h.ClickAt(centro)
	h.ClickAt(centro)
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyBackspace)
	h.Key(juigo.KeyEnter)
	if got := titulos(h); len(got) != 0 {
		t.Fatalf("editar para vazio deveria remover; got %v", got)
	}
	if h.Find(uitest.Text("0 itens restantes")) == nil {
		t.Fatal("o resumo deveria zerar")
	}
}

func TestMarcarTodasLimparEExcluir(t *testing.T) {
	_, _, h := monta(t)
	adiciona(h, "a")
	adiciona(h, "b")
	adiciona(h, "c")

	// Excluir pelo × da linha.
	h.Click(uitest.Text("×"))
	if got := titulos(h); len(got) != 2 || got[0] != "b" {
		t.Fatalf("o × deveria excluir a primeira; got %v", got)
	}

	// Limpar concluídas começa preso; marcar todas o libera.
	limpar := h.Find(uitest.Text("Limpar concluídas"))
	if !widget.DisabledOf(limpar) {
		t.Fatal("sem concluídas, Limpar deveria estar desabilitado")
	}
	caixas := h.FindAll(uitest.OfType[*juigo.Checkbox]())
	h.ClickAt(caixas[0].Bounds().Min.Add(caixas[0].Bounds().Size().Div(2))) // marcar todas
	if h.Find(uitest.Text("0 itens restantes")) == nil {
		t.Fatal("marcar todas deveria zerar os restantes")
	}
	if widget.DisabledOf(limpar) {
		t.Fatal("com concluídas, Limpar deveria habilitar")
	}
	h.Click(uitest.Text("Limpar concluídas"))
	if got := titulos(h); len(got) != 0 {
		t.Fatalf("Limpar deveria esvaziar a lista; got %v", got)
	}
}
