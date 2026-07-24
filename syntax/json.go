package syntax

import "github.com/JonathanSantos/JUIGo/widget"

// jsonLiterals são os três literais nomeados do JSON.
var jsonLiterals = map[string]bool{"true": true, "false": true, "null": true}

// JSON devolve o highlighter de JSON: strings, números e os literais
// true/false/null. JSON não tem construções multilinha — o estado é sempre
// o inicial.
func JSON() widget.Highlighter {
	return jsonLexer{}
}

type jsonLexer struct{}

func (jsonLexer) HighlightLine(line string, _ widget.HighlightState) ([]widget.HighlightSpan, widget.HighlightState) {
	var b spanBuilder
	i := 0
	for i < len(line) {
		c := line[i]
		switch {
		case c == '"':
			j := scanString(line, i, '"')
			b.add(j-i, widget.SyntaxString)
			i = j
		case c == '-' && i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9',
			c >= '0' && c <= '9':
			j := i + 1
			for j < len(line) && (isNumPart(line[j]) || line[j] == '-' || line[j] == '+') {
				j++
			}
			b.add(j-i, widget.SyntaxNumber)
			i = j
		case isIdentStart(c):
			j := i + 1
			for j < len(line) && isIdentPart(line[j]) {
				j++
			}
			st := widget.SyntaxText
			if jsonLiterals[line[i:j]] {
				st = widget.SyntaxBuiltin
			}
			b.add(j-i, st)
			i = j
		default:
			b.add(1, widget.SyntaxText)
			i++
		}
	}
	return b.spans, 0
}
