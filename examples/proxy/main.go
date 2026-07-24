// Proxy — um mini "Burp Suite" em JUIGo: um proxy de encaminhamento HTTP
// LOCAL que captura as trocas que passam por ele, filtra por método/URL,
// mostra requisição e resposta em visores (CodeEditor ReadOnly) e permite
// SIMULAR respostas de chamadas específicas (mocks). Três camadas, como os
// demais exemplos: proxy/ é o domínio puro, ui/ a Vista, este main compõe.
//
// Uso: rode o app, aponte um cliente HTTP para o proxy e veja as trocas:
//
//	go run ./examples/proxy
//	# noutro terminal:
//	HTTP_PROXY=http://localhost:8080 curl http://httpbin.org/get
//
// Ferramenta de DEPURAÇÃO da própria máquina. HTTPS passa por túnel (não é
// inspecionado): inspecionar/mockar HTTPS exigiria um CA de interceptação
// (MITM), fora do escopo deste exemplo.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/proxy/proxy"
	"github.com/JonathanSantos/JUIGo/examples/proxy/ui"
)

func main() {
	addr := flag.String("addr", ":8080", "endereço do proxy (host:porta)")
	flag.Parse()

	prox := proxy.New()

	app, err := juigo.New("Proxy — mini Burp em JUIGo", 1000, 620)
	if err != nil {
		log.Fatal(err)
	}
	vista := ui.New(prox, app.Post)
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
