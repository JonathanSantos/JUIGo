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
	linhas := h.FindAll(uitest.OfType[*linha]())
	if len(linhas) != 3 {
		t.Fatalf("deveria haver 3 linhas visíveis; got %d", len(linhas))
	}
	if !widget.DisabledOf(h.Find(uitest.Text("Atualizar"))) {
		t.Fatal("sem seleção, Atualizar deveria estar desabilitado")
	}

	// Clique numa linha: seleciona e espelha os campos.
	var alvo *linha
	for _, w := range linhas {
		if w.(*linha).pessoa.Sobrenome == "Mustermann" {
			alvo = w.(*linha)
		}
	}
	h.ClickAt(centro(alvo.Bounds()))
	if m.selecionado.Get() != 1 {
		t.Fatalf("clique deveria selecionar Mustermann (ID 1); got %d", m.selecionado.Get())
	}
	if m.nome.Get() != "Max" || m.sobrenome.Get() != "Mustermann" {
		t.Fatalf("campos deveriam espelhar a seleção; got %q %q", m.nome.Get(), m.sobrenome.Get())
	}
	if widget.DisabledOf(h.Find(uitest.Text("Excluir"))) {
		t.Fatal("com seleção, Excluir deveria habilitar")
	}

	// Atualizar troca o nome no cadastro.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Maximilian")
	h.Click(uitest.Text("Atualizar"))
	if p := m.Pessoas()[1]; p.Nome != "Maximilian" {
		t.Fatalf("Atualizar deveria aplicar o nome; got %q", p.Nome)
	}

	// Filtro por prefixo do sobrenome estreita a lista…
	h.Click(uitest.Placeholder("prefixo…"))
	h.Type("Mus")
	h.Layout() // rebinda o pool da lista após o SetCount
	if n := len(h.FindAll(uitest.OfType[*linha]())); n != 1 {
		t.Fatalf("filtro 'Mus' deveria deixar 1 linha; got %d", n)
	}
	// …e limpar o filtro traz todo mundo de volta.
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyBackspace)
	h.Layout()
	if n := len(h.FindAll(uitest.OfType[*linha]())); n != 3 {
		t.Fatalf("sem filtro deveriam voltar 3 linhas; got %d", n)
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

	// Excluir remove a selecionada (Mustermann segue selecionada).
	h.Click(uitest.Text("Excluir"))
	if len(m.Pessoas()) != 3 {
		t.Fatalf("Excluir deveria remover; got %d pessoas", len(m.Pessoas()))
	}
	for _, p := range m.Pessoas() {
		if p.Sobrenome == "Mustermann" {
			t.Fatal("Mustermann deveria ter sido excluído")
		}
	}
	if m.selecionado.Get() != -1 {
		t.Fatal("após excluir, a seleção deveria limpar")
	}
}
