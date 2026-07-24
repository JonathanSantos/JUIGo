package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// clienteViaProxy monta um cliente que confia na CA do proxy (aceita os
// certificados forjados) e roteia por ele — o que um navegador/app faria
// depois de instalar a CA.
func clienteViaProxy(t *testing.T, ca *CA, proxyURL string) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert())
	pu, _ := url.Parse(proxyURL)
	return &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(pu),
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}
}

// proxyInspetor sobe um proxy com CA e inspeção ligadas, confiando no cert
// do upstream TLS dado (o que, na vida real, é uma CA pública).
func proxyInspetor(t *testing.T, upstream *httptest.Server) (*Proxy, *httptest.Server) {
	t.Helper()
	ca, err := NewCA()
	if err != nil {
		t.Fatal(err)
	}
	p := New()
	p.CA = ca
	p.SetInspect(true)
	upPool := x509.NewCertPool()
	upPool.AddCert(upstream.Certificate())
	p.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: upPool}}
	srv := httptest.NewServer(p)
	t.Cleanup(srv.Close)
	return p, srv
}

// TestMITMInspecionaHTTPS: com a CA instalada e a inspeção ligada, uma
// chamada HTTPS é DECIFRADA — capturada em texto claro nas duas pontas — e
// ainda chega intacta ao servidor real.
func TestMITMInspecionaHTTPS(t *testing.T) {
	recebido := ""
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		recebido = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	p, srv := proxyInspetor(t, upstream)
	client := clienteViaProxy(t, p.CA, srv.URL)

	resp, err := client.Post(upstream.URL+"/login", "text/plain", strings.NewReader("senha123"))
	if err != nil {
		t.Fatalf("requisição HTTPS via proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("resposta ao cliente: %q", body)
	}
	if recebido != "senha123" {
		t.Fatalf("o upstream real deveria receber o corpo intacto; veio %q", recebido)
	}

	// A troca HTTPS foi capturada EM TEXTO CLARO.
	if p.Store.Count() != 1 {
		t.Fatalf("deveria capturar 1 troca HTTPS; veio %d", p.Store.Count())
	}
	ex := p.Store.Snapshot()[0]
	if !ex.Secure || ex.Tunnel {
		t.Fatalf("a troca deveria ser HTTPS inspecionada (Secure); %+v", ex)
	}
	if !strings.HasPrefix(ex.URL, "https://") {
		t.Fatalf("URL capturada: %q", ex.URL)
	}
	if !strings.Contains(ex.RequestText, "senha123") {
		t.Fatalf("requisição capturada não tem o corpo: %q", ex.RequestText)
	}
	if !strings.Contains(ex.ResponseText, `"ok":true`) {
		t.Fatalf("resposta capturada: %q", ex.ResponseText)
	}
}

// TestMITMMockDeHTTPS: um mock intercepta a chamada HTTPS sem tocar no
// servidor real — o caso de mockar endpoints para testar um front-end.
func TestMITMMockDeHTTPS(t *testing.T) {
	tocou := false
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tocou = true
	}))
	defer upstream.Close()

	p, srv := proxyInspetor(t, upstream)
	p.Mocks.Add(&MockRule{
		Enabled: true, Method: "GET", URLContains: "/saldo",
		Status: 200, ContentType: "application/json", Body: `{"saldo":42}`,
	})
	client := clienteViaProxy(t, p.CA, srv.URL)

	resp, err := client.Get(upstream.URL + "/saldo")
	if err != nil {
		t.Fatalf("GET HTTPS via proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if tocou {
		t.Fatal("o mock NÃO deveria tocar no servidor HTTPS real")
	}
	if !strings.Contains(string(body), "42") {
		t.Fatalf("resposta simulada: %q", body)
	}
	if resp.Header.Get("X-Mocked-By") != "juigo-proxy" {
		t.Fatal("a resposta simulada deveria marcar X-Mocked-By")
	}
	ex := p.Store.Snapshot()[0]
	if !ex.Mocked || !ex.Secure {
		t.Fatalf("a troca deveria ser mock sobre HTTPS: %+v", ex)
	}
}

// TestSemInspecaoTunelaHTTPS: sem a inspeção ligada, o HTTPS passa por túnel
// (não é decifrado) — o comportamento seguro por padrão.
func TestSemInspecaoTunelaHTTPS(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	ca, _ := NewCA()
	p := New()
	p.CA = ca
	p.SetInspect(false) // desligada
	srv := httptest.NewServer(p)
	defer srv.Close()

	// O cliente confia no cert REAL do upstream (túnel = TLS fim a fim).
	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	pu, _ := url.Parse(srv.URL)
	client := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(pu),
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}

	resp, err := client.Get(upstream.URL + "/x")
	if err != nil {
		t.Fatalf("túnel HTTPS: %v", err)
	}
	resp.Body.Close()

	ex := p.Store.Snapshot()[0]
	if !ex.Tunnel || ex.Secure {
		t.Fatalf("sem inspeção, a troca deveria ser um túnel: %+v", ex)
	}
}
