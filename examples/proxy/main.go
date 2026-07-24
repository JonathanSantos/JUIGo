// Proxy — um mini "Burp Suite" em JUIGo: um proxy de encaminhamento HTTP
// LOCAL que captura as trocas que passam por ele, filtra por método/URL,
// mostra requisição e resposta em visores (CodeEditor ReadOnly) e permite
// SIMULAR respostas de chamadas específicas (mocks). Três camadas, como os
// demais exemplos: proxy/ é o domínio puro, ui/ a Vista, este main compõe.
//
// HTTP é inspecionado direto. Para inspecionar/mockar HTTPS, ligue
// "Inspecionar HTTPS" e instale a CA local (botão "Exportar CA…") na trust
// store do sistema/navegador — é a mesma técnica do mitmproxy/Charles. A CA
// vive só na sua máquina; use apenas para depurar o SEU próprio tráfego.
//
//	go run ./examples/proxy
//	# noutro terminal:
//	HTTP_PROXY=http://localhost:8080 curl http://httpbin.org/get
//	# com HTTPS (após instalar a CA):
//	HTTPS_PROXY=http://localhost:8080 curl https://httpbin.org/get
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/proxy/proxy"
	"github.com/JonathanSantos/JUIGo/examples/proxy/ui"
	"github.com/JonathanSantos/JUIGo/quick"
)

func main() {
	addr := flag.String("addr", ":8080", "endereço do proxy (host:porta)")
	flag.Parse()

	prox := proxy.New()

	// CA local para interceptar HTTPS (persistida no diretório de config).
	dir := "juigo-proxy"
	if base, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(base, "juigo-proxy")
	}
	caPath := filepath.Join(dir, "juigo-proxy-ca.pem")
	if ca, err := proxy.LoadOrCreateCA(dir); err != nil {
		log.Printf("proxy: sem CA (HTTPS só por túnel): %v", err)
	} else {
		prox.CA = ca
	}

	app, err := juigo.New("Proxy — mini Burp em JUIGo", 1000, 620)
	if err != nil {
		log.Fatal(err)
	}
	vista := ui.New(prox, app.Post)
	vista.OnExportCA(func() {
		if prox.CA == nil {
			quick.Alert("Nenhuma CA disponível.")
			return
		}
		// A CA já está persistida em caPath por LoadOrCreateCA; avisa onde.
		quick.Alert("CA em:\n" + caPath + "\n\nInstale-a na trust store do sistema/navegador para inspecionar HTTPS. Use só para depurar o seu próprio tráfego.")
	})
	app.SetRoot(vista.Raiz)

	// Sobe o servidor do proxy em background; a UI segue na main thread.
	srv := &http.Server{Addr: *addr, Handler: prox}
	go func() {
		vista.SetStatus("ouvindo em " + *addr)
		app.Post(app.Invalidate)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("proxy: %v", err)
			app.Post(func() { vista.SetStatus("erro: " + err.Error()) })
		}
	}()
	defer srv.Close()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
