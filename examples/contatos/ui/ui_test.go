package ui

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/contatos/contatos"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// repoMemoria implementa o repositório em memória para os testes da UI.
type repoMemoria struct{ salvos int }

func (r *repoMemoria) Carregar() ([]contatos.Contato, error) { return nil, nil }
func (r *repoMemoria) Salvar([]contatos.Contato) error       { r.salvos++; return nil }

// vistaTeste monta a Vista com Ana (Plataforma), Bruno (Design) e Carla.
func vistaTeste(t *testing.T) (*Vista, *uitest.Harness, *repoMemoria) {
	a := contatos.NovaAgenda(nil)
	ana := a.Adicionar("Ana")
	a.Atualizar(contatos.Contato{ID: ana.ID, Nome: "Ana", Email: "ana@plataforma.dev", Empresa: "Plataforma"})
	a.Adicionar("Bruno")
	a.Adicionar("Carla")
	repo := &repoMemoria{}
	v := New(a, repo)
	h := uitest.New(t, v.Raiz, 760, 480)
	return v, h, repo
}

// centro devolve o centro do retângulo da linha i da tabela.
func (v *Vista) centroDaLinha(i int) (x, y int) {
	r := v.tabela.RowRect(i)
	return r.Min.X + r.Dx()/2, r.Min.Y + r.Dy()/2
}

// TestMestreDetalhe cobre o ciclo básico: sem seleção o detalhe fica
// desabilitado; selecionar carrega os campos; editar e Salvar atualiza o
// domínio, persiste, refresca a tabela e confirma com toast.
func TestMestreDetalhe(t *testing.T) {
	v, h, repo := vistaTeste(t)

	if !v.semSelecao.Get() {
		t.Fatal("sem seleção inicial, o detalhe deveria estar desabilitado")
	}

	// Seleciona Ana e confere a carga do detalhe.
	x, y := v.centroDaLinha(0)
	h.ClickAt(image.Pt(x, y))
	if v.nome.Value() != "Ana" || v.email.Value() != "ana@plataforma.dev" || v.semSelecao.Get() {
		t.Fatalf("selecionar deveria carregar o detalhe; nome=%q email=%q", v.nome.Value(), v.email.Value())
	}

	// Edita o nome e salva.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Ana Maria")
	salvosAntes := repo.salvos
	h.Click(uitest.Text("Salvar"))
	if got, _ := v.agenda.Buscar(v.selecionadoID); got.Nome != "Ana Maria" {
		t.Fatalf("Salvar deveria atualizar o domínio; veio %q", got.Nome)
	}
	if repo.salvos != salvosAntes+1 {
		t.Fatal("Salvar deveria persistir no repositório")
	}
	if v.celula(0, 0) != "Ana Maria" {
		t.Fatalf("a tabela deveria refletir o novo nome; veio %q", v.celula(0, 0))
	}
	if !h.Session().ToastVisible() {
		t.Fatal("Salvar deveria confirmar com toast")
	}
}

// TestBuscaPreservaSelecaoPorIdentidade: filtrar muda os índices, mas o
// contato selecionado continua o mesmo; filtrado para fora, a seleção
// limpa e o detalhe desabilita.
func TestBuscaPreservaSelecaoPorIdentidade(t *testing.T) {
	v, h, _ := vistaTeste(t)

	x, y := v.centroDaLinha(1) // Bruno
	h.ClickAt(image.Pt(x, y))
	id := v.selecionadoID
	if v.nome.Value() != "Bruno" {
		t.Fatalf("deveria carregar Bruno; veio %q", v.nome.Value())
	}

	v.busca.Set("bru")
	if v.selecionado.Get() != 0 || v.selecionadoID != id || v.nome.Value() != "Bruno" {
		t.Fatalf("filtro deveria preservar a seleção por identidade; idx=%d id=%d", v.selecionado.Get(), v.selecionadoID)
	}

	v.busca.Set("")
	if v.selecionado.Get() != 1 || v.selecionadoID != id {
		t.Fatalf("limpar o filtro deveria restaurar o índice; idx=%d", v.selecionado.Get())
	}

	v.busca.Set("carla")
	if v.selecionado.Get() != -1 || !v.semSelecao.Get() {
		t.Fatal("filtrado para fora, a seleção deveria limpar e o detalhe desabilitar")
	}
}

// TestAdicionarFocaONome: Adicionar cria o contato, o seleciona e abre o
// campo Nome focado; Enter no campo envia o formulário (Salvar).
func TestAdicionarFocaONome(t *testing.T) {
	v, h, _ := vistaTeste(t)

	h.Click(uitest.Text("Adicionar"))
	if v.nome.Value() != "Novo contato" {
		t.Fatalf("o novo contato deveria carregar no detalhe; veio %q", v.nome.Value())
	}
	if h.Focused() != v.nome.Control() {
		t.Fatal("o campo Nome deveria abrir focado")
	}

	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("Zeca")
	h.Key(juigo.KeyEnter) // Enter envia o formulário
	if got, _ := v.agenda.Buscar(v.selecionadoID); got.Nome != "Zeca" {
		t.Fatalf("Enter deveria salvar o novo nome; veio %q", got.Nome)
	}
}

// TestMenuDeContexto: botão direito seleciona a linha e abre o menu;
// Favoritar reordena com ♥; Excluir pede confirmação e remove com toast.
func TestMenuDeContexto(t *testing.T) {
	v, h, _ := vistaTeste(t)

	// Favorita Carla (linha 2): primeiro item do menu, Enter.
	x, y := v.centroDaLinha(2)
	h.RightClickAt(image.Pt(x, y))
	h.Key(juigo.KeyEnter)
	if v.visiveis[0].Nome != "Carla" || v.celula(0, 0) != "♥ Carla" {
		t.Fatalf("favoritar deveria reordenar com ♥; veio %v / %q", v.visiveis[0].Nome, v.celula(0, 0))
	}
	if v.selecionadoID == -1 || v.nome.Value() != "Carla" {
		t.Fatal("o botão direito deveria selecionar a linha (identidade segue a reordenação)")
	}

	// Exclui Carla (agora linha 0): segundo item, com confirmação.
	x, y = v.centroDaLinha(0)
	h.RightClickAt(image.Pt(x, y))
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEnter)
	if h.Find(uitest.Text("Excluir Carla?")) == nil {
		t.Fatal("o Confirm deveria estar aberto")
	}
	h.Click(uitest.Text("OK"))
	if _, ok := v.agenda.Buscar(v.selecionadoID); ok {
		t.Fatal("confirmar deveria excluir o contato")
	}
	if len(v.visiveis) != 2 {
		t.Fatalf("a tabela deveria ficar com 2 contatos; veio %d", len(v.visiveis))
	}
	if !h.Session().ToastVisible() {
		t.Fatal("a exclusão deveria confirmar com toast")
	}
}
