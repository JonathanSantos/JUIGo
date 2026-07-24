package proxy

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestCAGeraCadeiaValida(t *testing.T) {
	ca, err := NewCA()
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert())

	// Folha por nome: a cadeia valida contra a CA para aquele DNS.
	leaf, err := ca.LeafFor("api.exemplo.dev:443")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := leaf.Leaf.Verify(x509.VerifyOptions{DNSName: "api.exemplo.dev", Roots: pool}); err != nil {
		t.Fatalf("cadeia por nome inválida: %v", err)
	}
	if _, err := leaf.Leaf.Verify(x509.VerifyOptions{DNSName: "outro.dev", Roots: pool}); err == nil {
		t.Fatal("a folha não deveria valer para outro host")
	}

	// Cache: o mesmo host (com ou sem porta) devolve o mesmo certificado.
	leaf2, _ := ca.LeafFor("api.exemplo.dev")
	if leaf2 != leaf {
		t.Fatal("LeafFor deveria cachear por host")
	}

	// Folha por IP.
	ipLeaf, err := ca.LeafFor("127.0.0.1:8443")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ipLeaf.Leaf.Verify(x509.VerifyOptions{DNSName: "127.0.0.1", Roots: pool}); err != nil {
		t.Fatalf("cadeia por IP inválida: %v", err)
	}
}

func TestLoadOrCreateCAPersiste(t *testing.T) {
	dir := t.TempDir()
	ca1, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	// A segunda carga reaproveita a MESMA CA (mesmo certificado).
	ca2, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !ca1.Cert().Equal(ca2.Cert()) {
		t.Fatal("LoadOrCreateCA deveria reusar a CA persistida")
	}

	// A chave é gravada com permissão restritiva.
	info, err := os.Stat(filepath.Join(dir, "juigo-proxy-ca-key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("a chave da CA deveria ser 0600; veio %v", info.Mode().Perm())
	}

	// ExportPEM grava o certificado (o que se instala).
	saida := filepath.Join(dir, "exportada.pem")
	if err := ca1.ExportPEM(saida); err != nil {
		t.Fatal(err)
	}
	dados, _ := os.ReadFile(saida)
	if len(dados) == 0 || string(dados[:27]) != "-----BEGIN CERTIFICATE-----" {
		t.Fatalf("PEM exportado inesperado: %q", dados)
	}
}
