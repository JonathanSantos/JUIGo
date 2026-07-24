package proxy

// Interceptação HTTPS (MITM) — geração da CA e dos certificados por host.
//
// SEGURANÇA: a CA (certificado-raiz + chave privada) nasce e vive na SUA
// máquina. Instalá-la na trust store permite que ESTE proxy se passe por
// qualquer site HTTPS para a máquina que confia nela — por isso a chave é
// gravada com permissão 0600, nunca deve ser compartilhada, e a inspeção só
// se justifica para depurar o SEU próprio tráfego, no SEU próprio aparelho.
// É a mesma técnica do mitmproxy/Charles/Burp. Cert pinning continua
// rejeitando o proxy (de propósito).

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CA é a autoridade certificadora LOCAL do proxy: a raiz que assina um
// certificado forjado por host durante a interceptação HTTPS.
type CA struct {
	cert    *x509.Certificate
	certDER []byte
	key     *ecdsa.PrivateKey

	mu     sync.Mutex
	leaves map[string]*tls.Certificate
}

// serial devolve um número de série aleatório de 128 bits (a trust store
// reclama de séries repetidas).
func serial() *big.Int {
	limite := new(big.Int).Lsh(big.NewInt(1), 128)
	n, _ := rand.Int(rand.Reader, limite)
	if n.Sign() == 0 {
		n = big.NewInt(1)
	}
	return n
}

// NewCA gera uma CA nova EM MEMÓRIA (não persiste). Use LoadOrCreateCA para
// reaproveitar a mesma CA entre execuções.
func NewCA() (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("proxy: gerar chave da CA: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject: pkix.Name{
			CommonName:   "JUIGo Proxy CA (local)",
			Organization: []string{"JUIGo Proxy — CA local de desenvolvimento"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(5, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("proxy: criar certificado da CA: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &CA{cert: cert, certDER: der, key: key, leaves: map[string]*tls.Certificate{}}, nil
}

// LoadOrCreateCA carrega a CA persistida em dir (juigo-proxy-ca.pem +
// juigo-proxy-ca-key.pem) ou gera e grava uma nova. A chave vai com
// permissão 0600.
func LoadOrCreateCA(dir string) (*CA, error) {
	certPath := filepath.Join(dir, "juigo-proxy-ca.pem")
	keyPath := filepath.Join(dir, "juigo-proxy-ca-key.pem")

	certPEM, err1 := os.ReadFile(certPath)
	keyPEM, err2 := os.ReadFile(keyPath)
	if err1 == nil && err2 == nil {
		if ca, err := parseCA(certPEM, keyPEM); err == nil {
			return ca, nil
		}
		// Arquivos corrompidos: regenera.
	}

	ca, err := NewCA()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(certPath, ca.CertPEM(), 0o644); err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(ca.key)
	if err != nil {
		return nil, err
	}
	keyPEMOut := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEMOut, 0o600); err != nil {
		return nil, err
	}
	return ca, nil
}

// parseCA reconstrói a CA a partir dos PEMs do certificado e da chave.
func parseCA(certPEM, keyPEM []byte) (*CA, error) {
	cb, _ := pem.Decode(certPEM)
	kb, _ := pem.Decode(keyPEM)
	if cb == nil || kb == nil {
		return nil, fmt.Errorf("proxy: PEM da CA inválido")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParseECPrivateKey(kb.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{cert: cert, certDER: cb.Bytes, key: key, leaves: map[string]*tls.Certificate{}}, nil
}

// Cert devolve o certificado-raiz da CA (para montar uma trust store em
// testes).
func (ca *CA) Cert() *x509.Certificate {
	return ca.cert
}

// CertPEM devolve o certificado-raiz em PEM — o que se instala na trust
// store do sistema/navegador para a inspeção funcionar.
func (ca *CA) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.certDER})
}

// ExportPEM grava o certificado-raiz (PEM) no caminho dado, para instalar.
func (ca *CA) ExportPEM(path string) error {
	return os.WriteFile(path, ca.CertPEM(), 0o644)
}

// LeafFor devolve (gerando e cacheando na primeira vez) um certificado de
// servidor para host, assinado por esta CA — o que o proxy apresenta ao
// cliente ao se passar pelo site real. host pode vir com porta (é
// removida) e ser nome ou IP.
func (ca *CA) LeafFor(host string) (*tls.Certificate, error) {
	host = stripPort(host)
	ca.mu.Lock()
	defer ca.mu.Unlock()
	if c, ok := ca.leaves[host]; ok {
		return c, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{der, ca.certDER}, // folha + CA (cadeia completa)
		PrivateKey:  key,
		Leaf:        leaf,
	}
	ca.leaves[host] = tlsCert
	return tlsCert, nil
}

// stripPort remove ":porta" de um host, se houver.
func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
