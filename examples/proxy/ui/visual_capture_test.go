package ui

import (
	"os"
	"testing"

	"github.com/JonathanSantos/JUIGo/examples/proxy/proxy"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestCapturaVisual salva a cena da documentação (docs/proxy.png) quando a
// variável PROXY_CAPTURA aponta um caminho — só para inspeção manual:
//
//	PROXY_CAPTURA=docs/proxy.png go test ./examples/proxy/ui -run TestCapturaVisual
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("PROXY_CAPTURA")
	if caminho == "" {
		t.Skip("defina PROXY_CAPTURA para salvar o frame")
	}
	corpo := "{\n  \"id\": 42,\n  \"nome\": \"Ana Lima\",\n  \"plano\": \"pro\"\n}"
	p := proxy.New()
	// A Vista nasce ANTES das trocas: ela projeta pelo OnChange do Store
	// (aqui com post síncrono), então os Add abaixo já entram na tabela.
	v := New(p, func(fn func()) { fn() })
	p.Store.Add(&proxy.Exchange{
		Method: "GET", URL: "https://api.exemplo.dev/usuarios/42", Host: "api.exemplo.dev",
		Status: 200, RespType: "application/json", Size: len(corpo), Secure: true,
		RequestText:  "GET /usuarios/42 HTTP/1.1\nHost: api.exemplo.dev\nAccept: application/json",
		ResponseText: "HTTP/1.1 200 OK\nContent-Type: application/json\n\n" + corpo,
		RespBody:     corpo,
	})
	p.Store.Add(&proxy.Exchange{
		Method: "POST", URL: "http://httpbin.org/post", Host: "httpbin.org",
		Status: 201, RespType: "application/json", Size: 12,
		ResponseText: "HTTP/1.1 201 Created\n\n{\"ok\": true}", RespBody: "{\"ok\": true}",
	})
	p.Store.Add(&proxy.Exchange{
		Method: "GET", URL: "https://api.exemplo.dev/planos", Host: "api.exemplo.dev",
		Status: 418, RespType: "application/json", Size: 14, Mocked: true,
		ResponseText: "HTTP/1.1 418 I'm a teapot\n\n{\"mock\": true}", RespBody: "{\"mock\": true}",
	})
	// Captura em ESCALA 2 (retina): o PNG sai nítido como o App real.
	th, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}
	if err := th.SetScale(2); err != nil {
		t.Fatal(err)
	}
	h := uitest.NewWithTheme(t, v.Raiz, th, 1800, 1040)
	v.SelectFirst()
	v.ShowResponse()
	h.Layout()
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
