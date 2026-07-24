// Editor — o CodeEditor da fase 1 em ação: abra um arquivo passado como
// argumento (ou o exemplo embutido) e edite com gutter numerado, tabs
// literais, seleção multilinha, undo/redo coalescido (Cmd/Ctrl+Z; com
// Shift, refaz) e rolagem 2D. Ainda sem highlight nem salvar — highlight é
// a fase 2 do plano (docs/IME.md cobre a composição de texto, que este
// editor já desenha).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/syntax"
	"github.com/JonathanSantos/JUIGo/theme"
)

const amostra = `package main

import "fmt"

func main() {
	nome := "JUIGo"
	for i := 0; i < 3; i++ {
		fmt.Printf("olá, %s! (%d)\n", nome, i)
	}
}
`

func main() {
	titulo := "amostra.go"
	conteudo := amostra
	if len(os.Args) > 1 {
		dados, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatalf("editor: %v", err)
		}
		conteudo = string(dados)
		titulo = filepath.Base(os.Args[1])
	}

	app, err := juigo.New("Editor — CodeEditor em JUIGo", 720, 480)
	if err != nil {
		log.Fatal(err)
	}

	editor := juigo.NewCodeEditor()
	// Highlighter pela extensão: Go por padrão (a amostra é Go).
	switch strings.ToLower(filepath.Ext(titulo)) {
	case ".json":
		editor.Highlight(syntax.JSON())
	default:
		editor.Highlight(syntax.Go())
	}
	editor.SetText(conteudo)

	status := juigo.NewState("")
	atualiza := func() {
		status.Set(fmt.Sprintf("%s — %d linhas", titulo, editor.LineCount()))
	}
	editor.OnChange(atualiza)
	atualiza()

	// Ir à linha por composição: quick.Prompt + SetCursor.
	irALinha := juigo.NewButton("Ir à linha…", func() {
		quick.Prompt("Ir à linha", "número", func(v string) {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				editor.SetCursor(n-1, 0)
			}
			juigo.Focus(editor)
		})
	})

	// Tamanho da fonte deste editor (0 = o do tema).
	tamanho := 0.0
	ajusta := func(delta float64) {
		if tamanho == 0 {
			tamanho = 16
		}
		tamanho += delta
		if tamanho < 8 {
			tamanho = 8
		}
		if tamanho > 32 {
			tamanho = 32
		}
		editor.FontSize(tamanho)
	}
	menor := juigo.NewButton("A−", func() { ajusta(-2) }).Pad(4)
	maior := juigo.NewButton("A+", func() { ajusta(+2) }).Pad(4)

	// Quebra visual de linhas (sem rolagem horizontal).
	quebra := juigo.NewCheckbox("Quebrar linhas").OnChange(func(v bool) {
		editor.WrapLines(v)
	})

	// Fonte mono do tema: Go Mono ↔ Fira Code (embutidas).
	fonteSel := juigo.NewState("Go Mono")
	fonteSel.Watch(func(nome string) {
		fnt, err := theme.GoMono()
		if nome == "Fira Code" {
			fnt, err = theme.FiraCode()
		}
		if err != nil {
			log.Println("editor:", err)
			return
		}
		if err := app.Theme().UseMonoFont(fnt); err != nil {
			log.Println("editor:", err)
			return
		}
		app.Invalidate()
	})
	fonte := juigo.NewDropdown("Go Mono", "Fira Code").BindValue(fonteSel)

	raiz := juigo.NewVBox(
		juigo.Grow(editor, 1),
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("").BindText(status)),
			juigo.NewSpacer(),
			juigo.Centered(quebra),
			fonte,
			menor,
			maior,
			irALinha,
		).Gap(8),
	).Pad(8).Gap(6)

	app.SetRoot(raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
