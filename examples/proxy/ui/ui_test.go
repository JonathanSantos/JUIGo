package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/proxy/proxy"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// vistaTeste monta a Vista com um post SÍNCRONO (o relógio de testes não
// tem goroutines) e um punhado de trocas capturadas.
func vistaTeste(t *testing.T) (*Vista, *proxy.Proxy, *uitest.Harness) {
	t.Helper()
	prox := proxy.New()
	v := New(prox, func(fn func()) { fn() })
	h := uitest.New(t, v.Raiz, 1000, 620)

	prox.Store.Add(&proxy.Exchange{
		Method: "GET", URL: "http://api.x/users", Status: 200, RespType: "json",
		RequestText: "GET /users HTTP/1.1\nHost: api.x", RespBody: `{"n":2}`,
		ResponseText: "HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\"n\":2}",
	})
	prox.Store.Add(&proxy.Exchange{
		Method: "POST", URL: "http://api.x/login", Status: 401,
		RequestText: "POST /login HTTP/1.1", ResponseText: "HTTP/1.1 401 Unauthorized",
	})
	return v, prox, h
}

func centro(w juigo.Widget) image.Point {
	return w.Bounds().Min.Add(w.Bounds().Size().Div(2))
}

// TestCapturaFiltroEDetalhe: as trocas aparecem na tabela; selecionar carrega
// os visores ReadOnly; o filtro por método restringe preservando a seleção
// por identidade.
func TestCapturaFiltroEDetalhe(t *testing.T) {
	v, _, h := vistaTeste(t)

	if len(v.visiveis) != 2 {
		t.Fatalf("deveria capturar 2 trocas; veio %d", len(v.visiveis))
	}

	// Seleciona a 1ª linha (clique na tabela) e confere os visores.
	r := v.tabela.RowRect(0)
	h.ClickAt(image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2))
	if v.reqView.Text() == "" || v.respView.Text() == "" {
		t.Fatal("selecionar deveria carregar requisição e resposta")
	}
	sel := v.visiveis[v.sel.Get()]
	if sel.Method != "GET" {
		t.Fatalf("selecionou %s", sel.Method)
	}

	// Filtra por POST: some a GET, e a seleção (GET) limpa.
	v.metodo.Set("POST")
	if len(v.visiveis) != 1 || v.visiveis[0].Method != "POST" {
		t.Fatalf("filtro por método: %d visíveis", len(v.visiveis))
	}

	// Filtro por texto na URL.
	v.metodo.Set("Todos")
	v.filtro.Set("login")
	if len(v.visiveis) != 1 || v.visiveis[0].URL != "http://api.x/login" {
		t.Fatalf("filtro por texto: %v", v.visiveis)
	}
}

// TestVisorResponseEhReadOnly: digitar no visor de resposta não altera o
// conteúdo (é o CodeEditor em modo ReadOnly).
func TestVisorResponseEhReadOnly(t *testing.T) {
	v, _, h := vistaTeste(t)
	v.sel.Set(0)

	antes := v.respView.Text()
	h.ClickAt(centro(v.respView))
	h.Type("lixo")
	h.Key(juigo.KeyBackspace)
	if v.respView.Text() != antes {
		t.Fatalf("o visor deveria ser somente leitura; virou %q", v.respView.Text())
	}
	if !v.respView.IsReadOnly() {
		t.Fatal("respView deveria estar em ReadOnly")
	}
}

// TestSimularRespostaDaSelecao: criar um mock da troca selecionada faz o
// proxy interceptar a próxima chamada ao mesmo caminho.
func TestSimularRespostaDaSelecao(t *testing.T) {
	v, prox, _ := vistaTeste(t)
	v.sel.Set(0) // a GET /users, resposta JSON

	v.criaMockDaSelecao()
	if prox.Mocks.Count() != 1 {
		t.Fatalf("deveria criar 1 mock; veio %d", prox.Mocks.Count())
	}
	rule, ok := prox.Mocks.Match("GET", "http://outro-host.y/users?x=1")
	if !ok {
		t.Fatal("o mock deveria casar o mesmo caminho em qualquer host")
	}
	if rule.Status != 200 || rule.Body != `{"n":2}` {
		t.Fatalf("o mock deveria herdar status e corpo da resposta: %+v", rule)
	}

	// A lista de mocks foi reconstruída com a regra visível.
	if !mockNaLista(v, "/users") {
		t.Fatal("a regra deveria aparecer na lista de mocks")
	}
}

// mockNaLista informa se a lista de mocks contém um texto com o alvo.
func mockNaLista(v *Vista, alvo string) bool {
	for _, w := range v.mocks.Children() {
		if temTexto(w, alvo) {
			return true
		}
	}
	return false
}

// temTexto desce a árvore procurando um juigo.Text cujo conteúdo contenha o
// alvo.
func temTexto(w juigo.Widget, alvo string) bool {
	if txt, ok := w.(*juigo.Text); ok && strings.Contains(txt.Text(), alvo) {
		return true
	}
	if p, ok := w.(juigo.ParentWidget); ok {
		for _, ch := range p.Children() {
			if temTexto(ch, alvo) {
				return true
			}
		}
	}
	return false
}
