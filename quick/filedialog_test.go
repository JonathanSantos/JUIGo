package quick_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// raizDeTeste monta uma árvore de arquivos controlada:
//
//	raiz/
//	  docs/nota.txt
//	  leia.md
//	  .oculto  (não deve aparecer)
func raizDeTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{filepath.Join("docs", "nota.txt"), "leia.md", ".oculto"} {
		if err := os.WriteFile(filepath.Join(raiz, nome), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return raiz
}

// TestOpenFileNavegaEEscolhe percorre o fluxo feliz: entra na pasta pelo
// clique, sobe pelo "↑", seleciona o arquivo e confirma.
func TestOpenFileNavegaEEscolhe(t *testing.T) {
	raiz := raizDeTeste(t)
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")), 640, 560)

	var gotPath string
	gotOK := false
	chamadas := 0
	quick.OpenFileIn(raiz, "Abrir arquivo", func(path string, ok bool) {
		gotPath, gotOK = path, ok
		chamadas++
	})

	if h.Session().Overlay() == nil {
		t.Fatal("o diálogo deveria estar aberto")
	}
	if w := h.Find(uitest.Text(".oculto")); w != nil {
		t.Fatal("arquivos ocultos não deveriam aparecer")
	}

	// Pastas vêm primeiro, com sufixo "/"; clicar entra.
	h.Click(uitest.Text("docs/"))
	h.Layout()
	if h.Find(uitest.Text("nota.txt")) == nil {
		t.Fatal("após entrar em docs/, nota.txt deveria estar listado")
	}

	// "↑" volta à raiz.
	h.Click(uitest.Text("↑"))
	h.Layout()
	if h.Find(uitest.Text("leia.md")) == nil {
		t.Fatal("após subir, leia.md deveria estar listado")
	}

	// Seleciona o arquivo e confirma.
	h.Click(uitest.Text("leia.md"))
	h.Click(uitest.Text("Open"))
	if chamadas != 1 {
		t.Fatalf("onResult deveria rodar exatamente uma vez; rodou %d", chamadas)
	}
	if !gotOK || gotPath != filepath.Join(raiz, "leia.md") {
		t.Fatalf("esperava (%q, true); veio (%q, %v)", filepath.Join(raiz, "leia.md"), gotPath, gotOK)
	}
	if h.Session().Overlay() != nil {
		t.Fatal("o diálogo deveria ter fechado")
	}
}

// TestOpenFileSemSelecaoNaoConfirma: Open começa desabilitado; Escape
// cancela com ("", false) exatamente uma vez.
func TestOpenFileSemSelecaoNaoConfirma(t *testing.T) {
	raiz := raizDeTeste(t)
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")), 640, 560)

	chamadas := 0
	gotOK := true
	quick.OpenFileIn(raiz, "Abrir", func(path string, ok bool) {
		gotOK = ok
		chamadas++
	})

	// Sem arquivo selecionado, o clique em Open é engolido pelo disabled.
	h.Click(uitest.Text("Open"))
	if h.Session().Overlay() == nil {
		t.Fatal("Open desabilitado não deveria fechar o diálogo")
	}
	h.Key(juigo.KeyEscape)
	if chamadas != 1 || gotOK {
		t.Fatalf("Escape deveria cancelar uma vez com ok=false; chamadas=%d ok=%v", chamadas, gotOK)
	}
}

// TestSaveFileNomeEClique: o nome inicial preenche o campo, clicar num
// arquivo existente o substitui, e Save devolve o caminho completo.
func TestSaveFileNomeEClique(t *testing.T) {
	raiz := raizDeTeste(t)
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")), 640, 560)

	var gotPath string
	gotOK := false
	quick.SaveFileIn(raiz, "Salvar como", "novo.txt", func(path string, ok bool) {
		gotPath, gotOK = path, ok
	})

	// Clicar num arquivo existente copia o nome para o campo.
	h.Click(uitest.Text("leia.md"))
	campo := h.Find(uitest.Placeholder("nome do arquivo…")).(*juigo.Input)
	if campo.Text() != "leia.md" {
		t.Fatalf("o campo deveria ter adotado o nome clicado; tem %q", campo.Text())
	}

	h.Click(uitest.Text("Save"))
	if !gotOK || gotPath != filepath.Join(raiz, "leia.md") {
		t.Fatalf("esperava (%q, true); veio (%q, %v)", filepath.Join(raiz, "leia.md"), gotPath, gotOK)
	}
}

// TestSaveFileEnterNoCampo: Enter no campo confirma com o nome digitado,
// mesmo navegando antes.
func TestSaveFileEnterNoCampo(t *testing.T) {
	raiz := raizDeTeste(t)
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("fundo")), 640, 560)

	var gotPath string
	gotOK := false
	quick.SaveFileIn(raiz, "Salvar", "", func(path string, ok bool) {
		gotPath, gotOK = path, ok
	})

	h.Click(uitest.Text("docs/")) // salva dentro de docs/
	h.Click(uitest.Placeholder("nome do arquivo…"))
	h.Type("relatório.md")
	h.Key(juigo.KeyEnter)
	if !gotOK || gotPath != filepath.Join(raiz, "docs", "relatório.md") {
		t.Fatalf("esperava salvar em docs/relatório.md; veio (%q, %v)", gotPath, gotOK)
	}
}
