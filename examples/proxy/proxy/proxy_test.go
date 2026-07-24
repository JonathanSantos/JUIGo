package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// proxRequest monta uma requisição no estilo que um proxy recebe (URL
// absoluta) e a serve pelo handler, devolvendo o gravador.
func proxRequest(p *Proxy, metodo, url, corpo string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, url, strings.NewReader(corpo))
	r.RequestURI = url
	w := httptest.NewRecorder()
	p.ServeHTTP(w, r)
	return w
}

func TestEncaminhaECaptura(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"echo":"` + string(body) + `"}`))
	}))
	defer upstream.Close()

	p := New()
	w := proxRequest(p, "POST", upstream.URL+"/users", "ana")

	if w.Code != http.StatusCreated {
		t.Fatalf("status repassado = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"echo":"ana"`) {
		t.Fatalf("corpo repassado = %q", w.Body.String())
	}

	// A troca foi capturada com os textos brutos.
	if p.Store.Count() != 1 {
		t.Fatalf("deveria capturar 1 troca; veio %d", p.Store.Count())
	}
	ex := p.Store.Snapshot()[0]
	if ex.Method != "POST" || ex.Status != 201 || ex.RespType != "json" {
		t.Fatalf("metadados: %+v", ex)
	}
	if !strings.Contains(ex.RequestText, "POST") || !strings.Contains(ex.RequestText, "ana") {
		t.Fatalf("texto da requisição: %q", ex.RequestText)
	}
	if !strings.Contains(ex.ResponseText, "201") || !strings.Contains(ex.ResponseText, `"echo":"ana"`) {
		t.Fatalf("texto da resposta: %q", ex.ResponseText)
	}
	if ex.Mocked {
		t.Fatal("uma troca encaminhada não é mock")
	}
}

func TestMockInterceptaSemTocarNoServidor(t *testing.T) {
	tocou := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tocou = true
	}))
	defer upstream.Close()

	p := New()
	p.Mocks.Add(&MockRule{
		Enabled: true, Method: "GET", URLContains: "/saldo",
		Status: 200, ContentType: "application/json", Body: `{"saldo":9999}`,
	})

	w := proxRequest(p, "GET", upstream.URL+"/saldo", "")
	if tocou {
		t.Fatal("o mock NÃO deveria tocar no servidor real")
	}
	if w.Code != 200 || !strings.Contains(w.Body.String(), "9999") {
		t.Fatalf("resposta simulada: %d %q", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Mocked-By") != "juigo-proxy" {
		t.Fatal("a resposta simulada deveria marcar X-Mocked-By")
	}
	ex := p.Store.Snapshot()[0]
	if !ex.Mocked || ex.Status != 200 {
		t.Fatalf("a troca deveria constar como mock: %+v", ex)
	}

	// Método que não casa passa direto para o servidor.
	proxRequest(p, "POST", upstream.URL+"/saldo", "")
	if !tocou {
		t.Fatal("POST não casava a regra (GET) e deveria ter encaminhado")
	}
}

func TestMatchDeRegras(t *testing.T) {
	casos := []struct {
		rule        MockRule
		metodo, url string
		esperaCasar bool
	}{
		{MockRule{Enabled: true, URLContains: "api"}, "GET", "http://x/api/v1", true},
		{MockRule{Enabled: true, Method: "POST", URLContains: "api"}, "GET", "http://x/api", false},
		{MockRule{Enabled: true, Method: "*"}, "DELETE", "http://x/qualquer", true},
		{MockRule{Enabled: false, URLContains: "api"}, "GET", "http://x/api", false},
		{MockRule{Enabled: true, Method: "get"}, "GET", "http://x/", true}, // case-insensitive
	}
	for i, c := range casos {
		if got := c.rule.Matches(c.metodo, c.url); got != c.esperaCasar {
			t.Errorf("caso %d: Matches=%v, esperado %v", i, got, c.esperaCasar)
		}
	}

	m := NewMocks()
	m.Add(&MockRule{Enabled: true, URLContains: "a"})
	r2 := m.Add(&MockRule{Enabled: true, URLContains: "b"})
	if _, ok := m.Match("GET", "http://x/b"); !ok {
		t.Fatal("deveria casar a segunda regra")
	}
	m.Toggle(r2.ID)
	if _, ok := m.Match("GET", "http://x/b-only"); ok {
		t.Fatal("regra desligada não deveria casar")
	}
	m.Remove(r2.ID)
	if m.Count() != 1 {
		t.Fatalf("após remover: %d regras", m.Count())
	}
}

func TestStoreNotificaEIsola(t *testing.T) {
	s := NewStore()
	n := 0
	s.OnChange(func() { n++ })
	id1 := s.Add(&Exchange{Method: "GET"})
	s.Add(&Exchange{Method: "POST"})
	if n != 2 || id1 != 1 {
		t.Fatalf("notificações=%d id1=%d", n, id1)
	}

	// O snapshot é uma cópia: mexer nele não afeta o store.
	snap := s.Snapshot()
	snap[0] = nil
	if e, _ := s.Get(1); e == nil || e.Method != "GET" {
		t.Fatal("o snapshot deveria ser uma cópia isolada")
	}
	s.Clear()
	if s.Count() != 0 || n != 3 {
		t.Fatalf("após Clear: count=%d notificações=%d", s.Count(), n)
	}
}

func TestErroDeEncaminhamento(t *testing.T) {
	p := New()
	// Porta sem servidor: RoundTrip falha.
	w := proxRequest(p, "GET", "http://127.0.0.1:1/nada", "")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("erro deveria virar 502; veio %d", w.Code)
	}
	ex := p.Store.Snapshot()[0]
	if ex.Err == "" {
		t.Fatal("a troca deveria registrar o erro")
	}
}
