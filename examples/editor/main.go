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
	"strings"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/syntax"
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

	raiz := juigo.NewVBox(
		juigo.Grow(editor, 1),
		juigo.NewText("").BindText(status),
	).Pad(8).Gap(6)

	app, err := juigo.New("Editor — CodeEditor em JUIGo", 720, 480)
	if err != nil {
		log.Fatal(err)
	}
	app.SetRoot(raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
