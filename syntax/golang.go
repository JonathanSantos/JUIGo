package syntax

import (
	"strings"

	"github.com/JonathanSantos/JUIGo/widget"
)

// Estados carregados entre linhas pelo lexer de Go.
const (
	goNormal     widget.HighlightState = 0
	goComentario widget.HighlightState = 1
	goRaw        widget.HighlightState = 2
)

// goKeywords são as 25 palavras-chave da linguagem.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true,
	"continue": true, "default": true, "defer": true, "else": true,
	"fallthrough": true, "for": true, "func": true, "go": true,
	"goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true,
	"var": true,
}

// goBuiltins são tipos, constantes e funções embutidas.
var goBuiltins = map[string]bool{
	"any": true, "bool": true, "byte": true, "comparable": true,
	"complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true,
	"uint64": true, "uintptr": true,
	"true": true, "false": true, "iota": true, "nil": true,
	"append": true, "cap": true, "clear": true, "close": true,
	"complex": true, "copy": true, "delete": true, "imag": true,
	"len": true, "make": true, "max": true, "min": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true,
	"recover": true,
}

// Go devolve o highlighter léxico de Go: keywords, tipos/funções embutidas,
// strings (com raw strings atravessando linhas), runas, números e
// comentários (// e /* */ atravessando linhas).
func Go() widget.Highlighter {
	return goLexer{}
}

type goLexer struct{}

func (goLexer) HighlightLine(line string, state widget.HighlightState) ([]widget.HighlightSpan, widget.HighlightState) {
	var b spanBuilder
	i := 0

	// Estado herdado da linha anterior.
	switch state {
	case goComentario:
		if idx := strings.Index(line, "*/"); idx >= 0 {
			b.add(idx+2, widget.SyntaxComment)
			i = idx + 2
		} else {
			b.add(len(line), widget.SyntaxComment)
			return b.spans, goComentario
		}
	case goRaw:
		if idx := strings.IndexByte(line, '`'); idx >= 0 {
			b.add(idx+1, widget.SyntaxString)
			i = idx + 1
		} else {
			b.add(len(line), widget.SyntaxString)
			return b.spans, goRaw
		}
	}

	for i < len(line) {
		c := line[i]
		switch {
		case c == '/' && i+1 < len(line) && line[i+1] == '/':
			b.add(len(line)-i, widget.SyntaxComment)
			i = len(line)
		case c == '/' && i+1 < len(line) && line[i+1] == '*':
			if idx := strings.Index(line[i+2:], "*/"); idx >= 0 {
				b.add(idx+4, widget.SyntaxComment)
				i += idx + 4
			} else {
				b.add(len(line)-i, widget.SyntaxComment)
				return b.spans, goComentario
			}
		case c == '`':
			if idx := strings.IndexByte(line[i+1:], '`'); idx >= 0 {
				b.add(idx+2, widget.SyntaxString)
				i += idx + 2
			} else {
				b.add(len(line)-i, widget.SyntaxString)
				return b.spans, goRaw
			}
		case c == '"' || c == '\'':
			j := scanString(line, i, c)
			b.add(j-i, widget.SyntaxString)
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(line) && isNumPart(line[j]) {
				j++
			}
			b.add(j-i, widget.SyntaxNumber)
			i = j
		case isIdentStart(c):
			j := i + 1
			for j < len(line) && isIdentPart(line[j]) {
				j++
			}
			word := line[i:j]
			st := widget.SyntaxText
			switch {
			case goKeywords[word]:
				st = widget.SyntaxKeyword
			case goBuiltins[word]:
				st = widget.SyntaxBuiltin
			}
			b.add(j-i, st)
			i = j
		default:
			b.add(1, widget.SyntaxText)
			i++
		}
	}
	return b.spans, goNormal
}
