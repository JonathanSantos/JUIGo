// Package syntax traz highlighters prontos para o widget.CodeEditor —
// lexers LÉXICOS escritos à mão (keywords, strings, números, comentários),
// uma linha por vez com estado entre linhas, sem dependências:
//
//	editor := juigo.NewCodeEditor().Highlight(syntax.Go())
//
// O contrato (widget.Highlighter) é aberto: implemente o seu para outras
// linguagens — os spans devem cobrir a linha inteira, em bytes.
package syntax

import "github.com/JonathanSantos/JUIGo/widget"

// spanBuilder acumula spans coalescendo trechos consecutivos da mesma
// classe.
type spanBuilder struct {
	spans []widget.HighlightSpan
}

// add anexa n bytes com a classe dada (coalescendo com o span anterior).
func (b *spanBuilder) add(n int, st widget.SyntaxStyle) {
	if n <= 0 {
		return
	}
	if k := len(b.spans); k > 0 && b.spans[k-1].Style == st {
		b.spans[k-1].Len += n
		return
	}
	b.spans = append(b.spans, widget.HighlightSpan{Len: n, Style: st})
}

// isIdentStart aceita o início de identificador (letras Unicode entram pela
// heurística de byte alto — bom o suficiente para highlight léxico).
func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

// isIdentPart aceita a continuação de identificador.
func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// isNumPart aceita a continuação de literal numérico (hex, expoentes,
// separadores — aproximação léxica).
func isNumPart(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '.' || c == '_'
}

// scanString consome um literal delimitado por quote a partir de i (que
// aponta para a abertura), respeitando escapes com '\'. Devolve o índice
// logo após o fechamento (ou o fim da linha, se não fechar).
func scanString(line string, i int, quote byte) int {
	j := i + 1
	for j < len(line) {
		switch line[j] {
		case '\\':
			j += 2
		case quote:
			return j + 1
		default:
			j++
		}
	}
	if j > len(line) {
		j = len(line)
	}
	return j
}
