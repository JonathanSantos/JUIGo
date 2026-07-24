package syntax

import (
	"testing"

	"github.com/JonathanSantos/JUIGo/widget"
)

// estilos materializa a classe de CADA byte da linha e confere o invariante
// de que os spans a cobrem por inteiro.
func estilos(t *testing.T, h widget.Highlighter, line string, state widget.HighlightState) ([]widget.SyntaxStyle, widget.HighlightState) {
	t.Helper()
	spans, next := h.HighlightLine(line, state)
	out := make([]widget.SyntaxStyle, 0, len(line))
	for _, sp := range spans {
		for k := 0; k < sp.Len; k++ {
			out = append(out, sp.Style)
		}
	}
	if len(out) != len(line) {
		t.Fatalf("spans cobrem %d de %d bytes em %q", len(out), len(line), line)
	}
	return out, next
}

// classeEm devolve a classe do byte na posição i.
func classeEm(st []widget.SyntaxStyle, i int) widget.SyntaxStyle {
	return st[i]
}

func TestGoTokensBasicos(t *testing.T) {
	h := Go()
	line := `func soma(a int) int { return a + 42 } // fim`
	st, next := estilos(t, h, line, 0)
	if next != goNormal {
		t.Fatalf("estado final: %d", next)
	}
	if classeEm(st, 0) != widget.SyntaxKeyword { // func
		t.Fatal("'func' deveria ser keyword")
	}
	if classeEm(st, 12) != widget.SyntaxBuiltin { // int
		t.Fatal("'int' deveria ser builtin")
	}
	if classeEm(st, 5) != widget.SyntaxText { // soma
		t.Fatal("'soma' deveria ser texto comum")
	}
	if classeEm(st, 35) != widget.SyntaxNumber { // 42
		t.Fatal("'42' deveria ser número")
	}
	if classeEm(st, len(line)-1) != widget.SyntaxComment { // // fim
		t.Fatal("o comentário de linha deveria ir até o fim")
	}
}

func TestGoStringsEEscapes(t *testing.T) {
	h := Go()
	line := `x := "a\"b" + 'c' + "aberta`
	st, _ := estilos(t, h, line, 0)
	if classeEm(st, 5) != widget.SyntaxString || classeEm(st, 10) != widget.SyntaxString {
		t.Fatal("a string com aspas escapadas deveria ser uma só")
	}
	if classeEm(st, 12) != widget.SyntaxText { // o '+' entre literais
		t.Fatal("o operador entre strings é texto comum")
	}
	if classeEm(st, len(line)-1) != widget.SyntaxString {
		t.Fatal("string sem fechamento vai até o fim da linha")
	}
}

func TestGoComentarioDeBlocoEntreLinhas(t *testing.T) {
	h := Go()
	st1, s1 := estilos(t, h, `antes /* abre`, 0)
	if s1 != goComentario || classeEm(st1, 8) != widget.SyntaxComment {
		t.Fatalf("abrir /* deveria carregar o estado; %d", s1)
	}
	if classeEm(st1, 0) != widget.SyntaxText {
		t.Fatal("o que vem antes do /* segue comum")
	}
	st2, s2 := estilos(t, h, `func dentro`, s1)
	if s2 != goComentario || classeEm(st2, 0) != widget.SyntaxComment {
		t.Fatal("dentro do bloco, até 'func' é comentário")
	}
	st3, s3 := estilos(t, h, `fecha */ func f()`, s2)
	if s3 != goNormal || classeEm(st3, 0) != widget.SyntaxComment {
		t.Fatal("o prefixo até */ é comentário")
	}
	if classeEm(st3, 9) != widget.SyntaxKeyword {
		t.Fatal("depois do */ o lexer volta ao normal")
	}
}

func TestGoRawStringEntreLinhas(t *testing.T) {
	h := Go()
	_, s1 := estilos(t, h, "q := `raw", 0)
	if s1 != goRaw {
		t.Fatalf("crase aberta deveria carregar o estado; %d", s1)
	}
	st2, s2 := estilos(t, h, "meio", s1)
	if s2 != goRaw || classeEm(st2, 0) != widget.SyntaxString {
		t.Fatal("dentro da raw string tudo é string")
	}
	st3, s3 := estilos(t, h, "fim` + 1", s2)
	if s3 != goNormal || classeEm(st3, 3) != widget.SyntaxString || classeEm(st3, 7) != widget.SyntaxNumber {
		t.Fatal("a crase fecha e o resto volta ao normal")
	}
}

func TestJSON(t *testing.T) {
	h := JSON()
	line := `{"nome": "ana", "idade": -12.5e2, "ativa": true}`
	st, next := estilos(t, h, line, 0)
	if next != 0 {
		t.Fatal("JSON não carrega estado entre linhas")
	}
	if classeEm(st, 1) != widget.SyntaxString || classeEm(st, 9) != widget.SyntaxString {
		t.Fatal("chaves e valores string")
	}
	if classeEm(st, 25) != widget.SyntaxNumber {
		t.Fatal("número com sinal e expoente")
	}
	if classeEm(st, 43) != widget.SyntaxBuiltin {
		t.Fatal("'true' é literal nomeado")
	}
	if classeEm(st, 0) != widget.SyntaxText {
		t.Fatal("pontuação é texto comum")
	}
}
