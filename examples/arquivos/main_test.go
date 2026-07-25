package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// raizDeTeste monta uma pasta com subpastas e arquivos conhecidos.
func raizDeTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	for _, dir := range []string{"projetos", "musicas", filepath.Join("projetos", "juigo")} {
		if err := os.MkdirAll(filepath.Join(raiz, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, nome := range []string{
		filepath.Join("projetos", "demo.go"),
		filepath.Join("projetos", "leia.md"),
		"notas.txt",
	} {
		if err := os.WriteFile(filepath.Join(raiz, nome), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return raiz
}

// TestNavegadorDeArquivos percorre o fluxo: seleciona a pasta na árvore, o
// painel direito acompanha, e o diálogo Abrir… devolve o caminho escolhido.
func TestNavegadorDeArquivos(t *testing.T) {
	raiz := raizDeTeste(t)
	h := uitest.New(t, ui(raiz), 820, 560)

	// A raiz nasce expandida e selecionada; o painel direito lista os
	// arquivos dela.
	if h.Find(uitest.Text("notas.txt")) == nil {
		t.Fatal("o painel direito deveria listar notas.txt da raiz")
	}

	// Selecionar "projetos" na árvore troca a listagem.
	h.Click(uitest.Text("projetos"))
	h.Layout()
	if h.Find(uitest.Text("demo.go")) == nil {
		t.Fatal("após selecionar projetos, demo.go deveria aparecer à direita")
	}

	// Abrir… começa na pasta selecionada; escolher um arquivo ecoa no rodapé.
	h.Click(uitest.Text("Abrir…"))
	if h.Session().Overlay() == nil {
		t.Fatal("Abrir… deveria abrir o diálogo")
	}
	h.Click(uitest.Text("leia.md"))
	h.Click(uitest.Text("Open"))
	esperado := "Abriu: " + filepath.Join(raiz, "projetos", "leia.md")
	if h.Find(uitest.Text(esperado)) == nil {
		t.Fatalf("o rodapé deveria mostrar %q", esperado)
	}
}

// TestCapturaVisual salva docs/arquivos.png com o diálogo aberto sobre o
// split quando ARQUIVOS_CAPTURA aponta um caminho — só inspeção manual:
//
//	ARQUIVOS_CAPTURA=docs/arquivos.png go test ./examples/arquivos
func TestCapturaVisual(t *testing.T) {
	caminho := os.Getenv("ARQUIVOS_CAPTURA")
	if caminho == "" {
		t.Skip("defina ARQUIVOS_CAPTURA para salvar o frame")
	}
	// Raiz relativa via chdir: os caminhos da cena saem curtos e legíveis
	// ("Documentos/projetos"), não a pasta temporária absoluta. O caminho da
	// captura deve ser absoluto.
	base := t.TempDir()
	raiz := filepath.Join(base, "Documentos")
	for _, dir := range []string{"projetos/juigo", "musicas", "fotos"} {
		if err := os.MkdirAll(filepath.Join(raiz, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, nome := range []string{"projetos/demo.go", "projetos/leia.md", "notas.txt"} {
		if err := os.WriteFile(filepath.Join(raiz, nome), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(base)
	// Captura em ESCALA 2 (retina): o PNG sai nítido como o App real.
	th, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	if err := th.SetScale(2); err != nil {
		t.Fatal(err)
	}
	h := uitest.NewWithTheme(t, ui("Documentos"), th, 1640, 1120)
	h.Click(uitest.Text("projetos"))
	h.Click(uitest.Text("Abrir…"))
	h.Layout()
	if err := offscreen.SavePNG(caminho, h.Screenshot()); err != nil {
		t.Fatal(err)
	}
}
