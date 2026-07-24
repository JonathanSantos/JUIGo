package proxy

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// bodyLimit é o teto do corpo GUARDADO para exibição (a resposta ao cliente
// não é truncada; só a cópia capturada).
const bodyLimit = 1 << 20 // 1 MiB

// hopHeaders são os cabeçalhos ponto-a-ponto, não repassados ao encaminhar.
var hopHeaders = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
	"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// Proxy é o handler do proxy de encaminhamento: consulta os mocks, encaminha
// o que não for interceptado e captura tudo no Store. É um http.Handler —
// sirva-o com http.Server escutando na porta do proxy.
type Proxy struct {
	Store     *Store
	Mocks     *Mocks
	Transport http.RoundTripper
}

// New cria o proxy com store e mocks vazios e o transporte padrão.
func New() *Proxy {
	return &Proxy{
		Store:     NewStore(),
		Mocks:     NewMocks(),
		Transport: http.DefaultTransport,
	}
}

// ServeHTTP trata cada requisição que passa pelo proxy: CONNECT (HTTPS) é
// tunelado; o resto é mockado ou encaminhado, sempre capturado.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.tunnel(w, r)
		return
	}
	inicio := time.Now()
	reqBody, _ := io.ReadAll(io.LimitReader(r.Body, bodyLimit))
	r.Body.Close()

	ex := &Exchange{
		Method:      r.Method,
		URL:         r.URL.String(),
		Host:        r.Host,
		ReqType:     shortType(r.Header.Get("Content-Type")),
		Started:     inicio,
		RequestText: formatRequest(r, reqBody),
	}

	if rule, ok := p.Mocks.Match(r.Method, r.URL.String()); ok {
		p.serveMock(w, ex, rule)
		ex.Duration = time.Since(inicio)
		p.Store.Add(ex)
		return
	}

	p.forward(w, r, reqBody, ex)
	ex.Duration = time.Since(inicio)
	p.Store.Add(ex)
}

// serveMock responde com a regra simulada e registra a troca.
func (p *Proxy) serveMock(w http.ResponseWriter, ex *Exchange, rule *MockRule) {
	status := rule.Status
	if status == 0 {
		status = http.StatusOK
	}
	ct := rule.ContentType
	if ct == "" {
		ct = "text/plain; charset=utf-8"
	}
	body := []byte(rule.Body)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Mocked-By", "juigo-proxy")
	w.WriteHeader(status)
	w.Write(body)

	ex.Status = status
	ex.RespType = shortType(ct)
	ex.Size = len(body)
	ex.Mocked = true
	ex.RespBody = rule.Body
	ex.ResponseText = "HTTP/1.1 " + statusLine(status) + "\n" +
		"Content-Type: " + ct + "\nX-Mocked-By: juigo-proxy\n\n" + rule.Body
}

// forward encaminha a requisição ao servidor real, copia a resposta ao
// cliente e captura o par.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, reqBody []byte, ex *Exchange) {
	out := r.Clone(r.Context())
	out.RequestURI = ""
	out.Body = io.NopCloser(bytes.NewReader(reqBody))
	out.ContentLength = int64(len(reqBody))
	stripHop(out.Header)

	resp, err := p.Transport.RoundTrip(out)
	if err != nil {
		ex.Err = err.Error()
		http.Error(w, "juigo-proxy: "+err.Error(), http.StatusBadGateway)
		ex.Status = http.StatusBadGateway
		ex.ResponseText = "juigo-proxy: falha ao encaminhar\n\n" + err.Error()
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	// Repassa cabeçalhos, status e o corpo capturado ao cliente.
	dst := w.Header()
	for k, vs := range resp.Header {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	stripHop(dst)
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
	// Se o corpo excedeu o teto de captura, drena o resto para o cliente.
	if int64(len(respBody)) == bodyLimit {
		io.Copy(w, resp.Body)
	}

	ex.Status = resp.StatusCode
	ex.RespType = shortType(resp.Header.Get("Content-Type"))
	ex.Size = len(respBody)
	ex.RespBody = string(respBody)
	ex.ResponseText = formatResponse(resp, respBody)
}

// tunnel repassa uma conexão CONNECT (HTTPS) byte a byte, sem inspecionar,
// e registra o host tunelado.
func (p *Proxy) tunnel(w http.ResponseWriter, r *http.Request) {
	ex := &Exchange{
		Method:       http.MethodConnect,
		URL:          r.Host,
		Host:         r.Host,
		Tunnel:       true,
		Started:      time.Now(),
		RequestText:  "CONNECT " + r.Host + " (túnel HTTPS — não inspecionado)",
		ResponseText: "O tráfego HTTPS passa por um túnel e não é inspecionado.\n\nInspecionar/mockar HTTPS exigiria um CA de interceptação (MITM),\nfora do escopo deste exemplo.",
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
