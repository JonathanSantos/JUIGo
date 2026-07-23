package crud

import (
	"image"
	"testing"

	"juigo"
	"juigo/uitest"
	"juigo/widget"
)

func centro(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }

func TestCadastroCompleto(t *testing.T) {
	m, ui := New()
	h := uitest.New(t, ui, 560, 360)

	// Três pessoas pré-carregadas; sem seleção, Atualizar/Excluir presos.
	if m.tabela.Count() != 3 {
		t.Fatalf("a tabela deveria ter 3 linhas; got %d", m.tabela.Count())
	}
	if !widget.DisabledOf(h.Find(uitest.Text("Atualizar"))) {
		t.Fatal("sem seleção, Atualizar deveria estar desabilitado")
	}

	// Clique numa linha da Table: seleciona e espelha os campos.
	h.ClickAt(centro(m.tabela.RowRect(1))) // Mustermann
	if m.selecionada.Get() != 1 {
		t.Fatalf("clique deveria selecionar a linha 1; got %d", m.selecionada.Get())
	}
	if m.nome.Get() != "Max" || m.sobrenome.Get() != "Mustermann" {
		t.Fatalf("campos deveriam espelhar a seleção; got %q %q", m.nome.Get(), m.sobrenome.Get())
	}
	if widget.DisabledOf(h.Find(uitest.Text("Excluir"))) {
		t.Fatal("com seleção, Excluir deveria habilitar")
	}

	// Clique no CABEÇALHO fixo não seleciona nada.
	h.ClickAt(centro(image.Rect(
		m.tabela.Bounds().Min.X, m.tabela.Bounds().Min.Y,
		m.tabela.Bounds().Max.X, m.tabela.RowRect(0).Min.Y,
	)))
	if m.selecionada.Get() != 1 {
		t.Fatal("clique no cabeçalho não deveria mudar a seleção")
	}

	// Atualizar troca o nome no cadastro.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Maximilian")
	h.Click(uitest.Text("Atualizar"))
	if p := m.Pessoas()[1]; p.Nome != "Maximilian" {
		t.Fatalf("Atualizar deveria aplicar o nome; got %q", p.Nome)
	}

	// Filtro por prefixo estreita a tabela preservando a seleção por
	// identidade (Mustermann segue selecionado, agora na linha 0).
	h.Click(uitest.Placeholder("prefixo…"))
	h.Type("Mus")
	if m.tabela.Count() != 1 {
		t.Fatalf("filtro 'Mus' deveria deixar 1 linha; got %d", m.tabela.Count())
	}
	if m.selecionada.Get() != 0 {
		t.Fatalf("a seleção deveria seguir a pessoa para a linha 0; got %d", m.selecionada.Get())
	}
	// Limpar o filtro traz todo mundo de volta.
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyBackspace)
	if m.tabela.Count() != 3 {
		t.Fatalf("sem filtro deveriam voltar 3 linhas; got %d", m.tabela.Count())
	}

	// Criar insere com os campos atuais.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Ana")
	h.Click(uitest.Placeholder("sobrenome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Silva")
	h.Click(uitest.Text("Criar"))
	if len(m.Pessoas()) != 4 {
		t.Fatalf("Criar deveria inserir a 4ª pessoa; got %d", len(m.Pessoas()))
	}

	// Excluir remove a selecionada (Mustermann segue selecionado).
	h.Click(uitest.Text("Excluir"))
	if len(m.Pessoas()) != 3 {
		t.Fatalf("Excluir deveria remover; got %d pessoas", len(m.Pessoas()))
	}
	for _, p := range m.Pessoas() {
		if p.Sobrenome == "Mustermann" {
			t.Fatal("Mustermann deveria ter sido excluído")
		}
	}
	if m.selecionada.Get() != -1 {
		t.Fatal("após excluir, a seleção deveria limpar")
	}
}
