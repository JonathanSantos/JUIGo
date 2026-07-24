package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// bodyLimit é o teto do corpo GUARDADO para exibição (e, no proxy, também o
// que é repassado — corpos maiores são truncados: é uma ferramenta de
// depuração, não um proxy de produção).
const bodyLimit = 1 << 20 // 1 MiB

// hopHeaders são os cabeçalhos ponto-a-ponto, não repassados ao encaminhar.
var hopHeaders = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
	"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// Proxy é o handler do proxy de encaminhamento: consulta os mocks, encaminha
// o que não for interceptado e captura tudo no Store. É um http.Handler —
// sirva-o com http.Server escutando na porta do proxy. Com uma CA e a
// inspeção ligada (SetInspect), decifra também o HTTPS (MITM); senão, o
// HTTPS passa por túnel sem inspeção.
type Proxy struct {
	Store     *Store
	Mocks     *Mocks
	Transport http.RoundTripper
	// CA assina os certificados forjados da interceptação HTTPS (ver ca.go);
	// nil = sem inspeção de HTTPS.
	CA *CA

	inspect atomic.Bool
}

// New cria o proxy com store e mocks vazios e o transporte padrão.
func New() *Proxy {
	return &Proxy{
		Store:     NewStore(),
		Mocks:     NewMocks(),
		Transport: http.DefaultTransport,
	}
}

// SetInspect liga/desliga a INTERCEPTAÇÃO de HTTPS (só tem efeito com uma CA
// definida). Seguro para chamar de outra goroutine (a UI).
func (p *Proxy) SetInspect(on bool) {
	p.inspect.Store(on)
}

// Inspecting informa se a interceptação de HTTPS está ativa E possível.
func (p *Proxy) Inspecting() bool {
	return p.inspect.Load() && p.CA != nil
}

// capturedResp é a resposta pronta para ser enviada ao cliente — construída
// uma vez (mock ou encaminhamento) e escrita tanto no HTTP claro quanto na
// conexão TLS decifrada.
type capturedResp struct {
	status int
	header http.Header
	body   []byte
}

// ServeHTTP trata cada requisição que passa pelo proxy: CONNECT (HTTPS) é
// interceptado (com CA) ou tunelado; o resto é mockado ou encaminhado,
// sempre capturado.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if p.Inspecting() {
			p.mitm(w, r)
		} else {
			p.blindTunnel(w, r)
		}
		return
	}
	reqBody, _ := io.ReadAll(io.LimitReader(r.Body, bodyLimit))
	r.Body.Close()
	res, ex := p.handle(r, reqBody)
	writeResponse(w, res)
	p.Store.Add(ex)
}

// handle roda o ciclo mock-ou-encaminha para uma requisição já lida,
// devolvendo a resposta a enviar e a troca capturada. É o núcleo comum ao
// HTTP claro e ao HTTPS decifrado.
func (p *Proxy) handle(r *http.Request, reqBody []byte) (capturedResp, *Exchange) {
	inicio := time.Now()
	ex := &Exchange{
		Method:      r.Method,
		URL:         r.URL.String(),
		Host:        r.Host,
		Secure:      r.URL.Scheme == "https",
		ReqType:     shortType(r.Header.Get("Content-Type")),
		Started:     inicio,
		RequestText: formatRequest(r, reqBody),
	}
	var res capturedResp
	if rule, ok := p.Mocks.Match(r.Method, r.URL.String()); ok {
		res = p.mockResponse(ex, rule)
	} else {
		res = p.forwardResponse(r, reqBody, ex)
	}
	ex.Duration = time.Since(inicio)
	return res, ex
}

// mockResponse monta a resposta simulada e preenche a captura.
func (p *Proxy) mockResponse(ex *Exchange, rule *MockRule) capturedResp {
	status := rule.Status
	if status == 0 {
		status = http.StatusOK
	}
	ct := rule.ContentType
	if ct == "" {
		ct = "text/plain; charset=utf-8"
	}
	body := []byte(rule.Body)
	h := http.Header{}
	h.Set("Content-Type", ct)
	h.Set("X-Mocked-By", "juigo-proxy")

	ex.Status = status
	ex.RespType = shortType(ct)
	ex.Size = len(body)
	ex.Mocked = true
	ex.RespBody = rule.Body
	ex.ResponseText = "HTTP/1.1 " + statusLine(status) + "\n" +
		"Content-Type: " + ct + "\nX-Mocked-By: juigo-proxy\n\n" + rule.Body
	return capturedResp{status: status, header: h, body: body}
}

// forwardResponse encaminha a requisição ao servidor real e preenche a
// captura com o par.
func (p *Proxy) forwardResponse(r *http.Request, reqBody []byte, ex *Exchange) capturedResp {
	out := r.Clone(r.Context())
	out.RequestURI = ""
	out.Body = io.NopCloser(bytes.NewReader(reqBody))
	out.ContentLength = int64(len(reqBody))
	stripHop(out.Header)

	resp, err := p.Transport.RoundTrip(out)
	if err != nil {
		ex.Err = err.Error()
		ex.Status = http.StatusBadGateway
		ex.ResponseText = "juigo-proxy: falha ao encaminhar\n\n" + err.Error()
		return capturedResp{
			status: http.StatusBadGateway,
			header: http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
			body:   []byte("juigo-proxy: " + err.Error()),
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	ex.Status = resp.StatusCode
	ex.RespType = shortType(resp.Header.Get("Content-Type"))
	ex.Size = len(body)
	ex.RespBody = string(body)
	ex.ResponseText = formatResponse(resp, body)

	h := resp.Header.Clone()
	stripHop(h)
	return capturedResp{status: resp.StatusCode, header: h, body: body}
}

// writeResponse escreve a resposta capturada num http.ResponseWriter (HTTP
// claro).
func writeResponse(w http.ResponseWriter, res capturedResp) {
	dst := w.Header()
	for k, vs := range res.header {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	w.WriteHeader(res.status)
	w.Write(res.body)
}

// mitm INTERCEPTA uma conexão CONNECT (HTTPS): apresenta ao cliente um
// certificado forjado (assinado pela CA local), decifra o TLS e trata cada
// requisição como no HTTP claro — capturando e podendo mockar em texto
// claro. Requer a CA do cliente instalada na trust store (ver ca.go).
func (p *Proxy) mitm(w http.ResponseWriter, r *http.Request) {
	leaf, err := p.CA.LeafFor(r.Host)
	if err != nil {
		http.Error(w, "juigo-proxy: "+err.Error(), http.StatusInternalServerError)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "juigo-proxy: sem suporte a hijack", http.StatusInternalServerError)
		return
	}
	cliente, _, err := hj.Hijack()
	if err != nil {
		return
	}
	defer cliente.Close()
	cliente.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	tlsConn := tls.Server(cliente, &tls.Config{Certificates: []tls.Certificate{*leaf}})
	if err := tlsConn.Handshake(); err != nil {
		return // cert pinning ou CA não confiada: o cliente recusou
	}
	defer tlsConn.Close()

	autoridade := r.Host // host:porta do CONNECT
	leitor := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(leitor)
		if err != nil {
			return // EOF ou conexão fechada
		}
		reqBody, _ := io.ReadAll(io.LimitReader(req.Body, bodyLimit))
		req.Body.Close()
		// A requisição decifrada vem em forma de origem ("/caminho"):
		// completa o esquema e o host para o encaminhamento e a captura.
		req.URL.Scheme = "https"
		req.URL.Host = autoridade

		res, ex := p.handle(req, reqBody)
		p.Store.Add(ex)
		if err := writeConnResponse(tlsConn, req, res); err != nil {
			return
		}
		if req.Close {
			return
		}
	}
}

// writeConnResponse serializa a resposta capturada de volta na conexão TLS
// decifrada.
func writeConnResponse(conn net.Conn, req *http.Request, res capturedResp) error {
	h := res.header
	if h == nil {
		h = http.Header{}
	}
	h.Del("Content-Length") // será derivado de ContentLength
	h.Del("Transfer-Encoding")
	resp := &http.Response{
		StatusCode:    res.status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        h,
		Body:          io.NopCloser(bytes.NewReader(res.body)),
		ContentLength: int64(len(res.body)),
		Request:       req,
	}
	return resp.Write(conn)
}

// blindTunnel repassa uma conexão CONNECT (HTTPS) byte a byte, SEM
// inspecionar — o caminho quando não há CA ou a inspeção está desligada.
func (p *Proxy) blindTunnel(w http.ResponseWriter, r *http.Request) {
	ex := &Exchange{
		Method:       http.MethodConnect,
		URL:          r.Host,
		Host:         r.Host,
		Tunnel:       true,
		Started:      time.Now(),
		RequestText:  "CONNECT " + r.Host + " (túnel HTTPS — não inspecionado)",
		ResponseText: "O tráfego HTTPS passa por um túnel e não é inspecionado.\n\nLigue “Inspecionar HTTPS” (com a CA instalada) para vê-lo em texto claro.",
	}
	defer p.Store.Add(ex)

	destino, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		ex.Err = err.Error()
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		ex.Err = "sem suporte a hijack"
		http.Error(w, "juigo-proxy: sem suporte a hijack", http.StatusInternalServerError)
		destino.Close()
		return
	}
	cliente, _, err := hj.Hijack()
	if err != nil {
		ex.Err = err.Error()
		destino.Close()
		return
	}
	cliente.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go copiaFecha(destino, cliente)
	go copiaFecha(cliente, destino)
}

// copiaFecha copia de src para dst e fecha ambos ao terminar.
func copiaFecha(dst io.WriteCloser, src io.ReadCloser) {
	io.Copy(dst, src)
	dst.Close()
	src.Close()
}

// stripHop remove os cabeçalhos ponto-a-ponto.
func stripHop(h http.Header) {
	for _, k := range hopHeaders {
		h.Del(k)
	}
}

// statusLine devolve "<code> <texto>" para a linha de status simulada.
func statusLine(code int) string {
	txt := http.StatusText(code)
	if txt == "" {
		txt = "Status"
	}
	return strconv.Itoa(code) + " " + txt
}
