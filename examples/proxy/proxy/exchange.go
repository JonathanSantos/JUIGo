// Package proxy é o DOMÍNIO do mini-proxy de depuração HTTP: um proxy de
// encaminhamento (forward proxy) que CAPTURA as trocas HTTP que passam por
// ele e pode responder chamadas específicas com respostas simuladas
// (mocks). É o núcleo testável — não importa juigo; a UI é uma projeção
// dele. Ferramenta de depuração LOCAL da própria máquina, no espírito de
// mitmproxy/Charles.
//
// Escopo honesto: HTTP em texto claro é capturado e pode ser mockado; o
// HTTPS chega como CONNECT e é TUNELADO (passa, mas não é inspecionado) —
// inspecionar/mockar HTTPS exigiria um CA de interceptação (MITM), fora do
// escopo deste exemplo de estudo.
package proxy

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Exchange é uma troca HTTP capturada: a requisição e a resposta (ou o
// erro), com metadados para a lista e os textos brutos para o visor.
type Exchange struct {
	ID       int
	Method   string
	URL      string
	Host     string
	Status   int // 0 = falhou antes da resposta
	ReqType  string
	RespType string
	Size     int // bytes do corpo da resposta
	Mocked   bool
	Tunnel   bool // CONNECT (HTTPS): só o host, sem corpo
	Err      string
	Started  time.Time
	Duration time.Duration

	RequestText  string
	ResponseText string
	// RespBody é só o corpo da resposta (sem a linha de status nem os
	// cabeçalhos) — a semente do "simular resposta".
	RespBody string
}

// StatusText devolve o status como texto para a lista ("—" sem resposta;
// "túnel" para CONNECT).
func (e *Exchange) StatusText() string {
	switch {
	case e.Tunnel:
		return "túnel"
	case e.Err != "":
		return "erro"
	case e.Status == 0:
		return "—"
	default:
		return fmt.Sprintf("%d", e.Status)
	}
}

// shortType reduz um Content-Type ao subtipo útil (application/json;
// charset=utf-8 → json).
func shortType(ct string) string {
	if ct == "" {
		return ""
	}
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.TrimSpace(ct)
	if i := strings.IndexByte(ct, '/'); i >= 0 {
		return ct[i+1:]
	}
	return ct
}

// formatRequest monta o texto bruto da requisição (linha, cabeçalhos e
// corpo) para o visor.
func formatRequest(r *http.Request, body []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\n", r.Method, requestTarget(r), r.Proto)
	fmt.Fprintf(&b, "Host: %s\n", r.Host)
	writeHeaders(&b, r.Header)
	writeBody(&b, body)
	return b.String()
}

// formatResponse monta o texto bruto da resposta para o visor.
func formatResponse(resp *http.Response, body []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", resp.Proto, resp.Status)
	writeHeaders(&b, resp.Header)
	writeBody(&b, body)
	return b.String()
}

// requestTarget devolve o alvo da linha de requisição (caminho + query).
func requestTarget(r *http.Request) string {
	if r.URL.RawQuery != "" {
		return r.URL.Path + "?" + r.URL.RawQuery
	}
	if r.URL.Path == "" {
		return "/"
	}
	return r.URL.Path
}

// writeHeaders escreve os cabeçalhos em ordem estável (nomes ordenados).
func writeHeaders(b *strings.Builder, h http.Header) {
	nomes := make([]string, 0, len(h))
	for k := range h {
		nomes = append(nomes, k)
	}
	sortStrings(nomes)
	for _, k := range nomes {
		for _, v := range h[k] {
			fmt.Fprintf(b, "%s: %s\n", k, v)
		}
	}
}

// writeBody anexa o corpo após uma linha em branco, se houver.
func writeBody(b *strings.Builder, body []byte) {
	if len(body) == 0 {
		return
	}
	b.WriteByte('\n')
	b.Write(body)
}

// sortStrings ordena in place (evita importar sort só para isto).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
