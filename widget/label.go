package widget

import (
	"image"
	"image/color"
	"strings"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Label é o parágrafo do JUIGo: texto só-leitura com QUEBRA DE LINHA —
// quebras duras (\n) e soft wrap por palavras na largura disponível. Aceita
// os papéis tipográficos do tema (Title/Subtitle/Caption) e as cores de
// Text (Color/Danger).
//
//	juigo.NewLabel("Um parágrafo inteiro, que quebra sozinho na largura " +
//	    "do layout — sem empilhar Texts na mão.")
//
// A altura preferida acompanha a largura: com MaxWidth ela é determinística
// desde o primeiro frame; sem, o Label quebra na largura do último layout e
// converge no frame seguinte a uma mudança de largura (ele agenda o frame
// sozinho). Palavras mais largas que a linha não são hifenizadas — ficam
// sozinhas na linha e recortam à direita (limitação documentada).
type Label struct {
	BaseWidget
	text string
	role textRole

	col    color.RGBA
	danger bool
	// maxW limita a largura do wrap em unidades lógicas (0 = a largura do
	// layout).
	maxW int

	// Cache do wrap: fatias [início,fim) em bytes por linha VISUAL, válido
	// para (largura, versão do texto, face do papel — a chave muda em troca
	// de tema ou de escala).
	lines   [][2]int
	wrapW   int
	wrapVer int
	ver     int
	wrapKey any
	clip    image.RGBA
}

// NewLabel cria um parágrafo com o conteúdo dado. O tema é herdado no mount.
func NewLabel(s string) *Label {
	return &Label{text: s}
}

// Title dá ao parágrafo o papel de TÍTULO (ver Text.Title). Encadeável.
func (l *Label) Title() *Label { l.role = roleTitle; l.bump(); return l }

// Subtitle dá o papel de SUBTÍTULO. Encadeável.
func (l *Label) Subtitle() *Label { l.role = roleSubtitle; l.bump(); return l }

// Caption dá o papel de LEGENDA. Encadeável.
func (l *Label) Caption() *Label { l.role = roleCaption; l.bump(); return l }

// Color sobrescreve a cor do texto (ver Text.Color). Encadeável.
func (l *Label) Color(c color.RGBA) *Label {
	l.col = c
	l.danger = false
	l.Invalidate()
	return l
}

// Danger usa a cor de erro do tema (ver Text.Danger). Encadeável.
func (l *Label) Danger() *Label {
	l.danger = true
	l.Invalidate()
	return l
}

// MaxWidth limita a largura do wrap em unidades lógicas — a altura
// preferida fica determinística sem depender do layout (zero volta ao
// automático). Encadeável.
func (l *Label) MaxWidth(lu int) *Label {
	l.maxW = lu
	l.bump()
	return l
}

// BindText vincula o conteúdo ao State (ver Text.BindText). Encadeável.
func (l *Label) BindText(s *state.State[string]) *Label {
	l.SetText(s.Get())
	s.Watch(func(v string) { l.SetText(v) })
	return l
}

// Text devolve o conteúdo atual.
func (l *Label) Text() string { return l.text }

// SetText troca o conteúdo e agenda um redesenho se ele mudou.
func (l *Label) SetText(s string) {
	if l.text == s {
		return
	}
	l.text = s
	l.bump()
}

// bump invalida o cache do wrap e agenda um redesenho.
func (l *Label) bump() {
	l.ver++
	l.Invalidate()
}

// measure mede s pelo papel tipográfico atual (método, não method value —
// nada de closure alocando no caminho quente).
func (l *Label) measure(s string) int {
	if f := l.roleFont(); f != nil {
		return f.Measure(s)
	}
	return l.theme.MeasureString(s)
}

// metrics devolve altura de linha, ascent e a CHAVE de invalidação do papel
// atual (a face — muda em troca de tema ou escala).
func (l *Label) metrics() (lineH, ascent int, key any) {
	if f := l.roleFont(); f != nil {
		return f.LineHeight(), f.Ascent(), f.Face
	}
	th := l.theme
	return th.LineHeight(), th.Ascent(), th.Face
}

// roleFont devolve a fonte do papel atual, ou nil para o corpo.
func (l *Label) roleFont() *theme.TextFont {
	if l.theme == nil {
		return nil
	}
	switch l.role {
	case roleTitle:
		return l.theme.Title()
	case roleSubtitle:
		return l.theme.Subtitle()
	case roleCaption:
		return l.theme.Caption()
	}
	return nil
}

// wrapWidth devolve a largura de quebra em pixels: MaxWidth quando
// definida, senão a largura do último layout (0 = sem quebra suave).
func (l *Label) wrapWidth() int {
	if l.theme == nil {
		return 0
	}
	if l.maxW > 0 {
		return l.theme.Px(l.maxW)
	}
	return l.Bounds().Dx()
}

// ensureWrap recalcula as linhas visuais para a largura dada, se preciso.
func (l *Label) ensureWrap(width int) {
	_, _, key := l.metrics()
	if l.wrapW == width && l.wrapVer == l.ver && l.wrapKey == key {
		return
	}
	l.wrapW, l.wrapVer, l.wrapKey = width, l.ver, key
	l.lines = l.lines[:0]

	spaceW := l.measure(" ")
	start := 0
	for start <= len(l.text) {
		// Linha DURA corrente: até o próximo \n (ou o fim).
		end := strings.IndexByte(l.text[start:], '\n')
		hardEnd := len(l.text)
		if end >= 0 {
			hardEnd = start + end
		}
		l.wrapHardLine(start, hardEnd, width, spaceW)
		if hardEnd == len(l.text) {
			break
		}
		start = hardEnd + 1
	}
	if len(l.lines) == 0 {
		l.lines = append(l.lines, [2]int{0, 0})
	}
}

// wrapHardLine quebra o trecho [start,end) por palavras dentro de width
// (width <= 0 não quebra). A largura acumula palavra a palavra (mais o
// espaço) — kerning atravessando espaços é desprezível.
func (l *Label) wrapHardLine(start, end, width, spaceW int) {
	if start >= end {
		l.lines = append(l.lines, [2]int{start, start})
		return
	}
	lineStart, lineW := -1, 0
	i := start
	for i < end {
		// Pula espaços entre palavras.
		for i < end && l.text[i] == ' ' {
			i++
		}
		if i >= end {
			break
		}
		wordStart := i
		for i < end && l.text[i] != ' ' {
			i++
		}
		wordW := l.measure(l.text[wordStart:i])
		switch {
		case lineStart < 0:
			lineStart, lineW = wordStart, wordW
		case width > 0 && lineW+spaceW+wordW > width:
			l.lines = append(l.lines, [2]int{lineStart, prevWordEnd(l.text, wordStart)})
			lineStart, lineW = wordStart, wordW
		default:
			lineW += spaceW + wordW
		}
	}
	if lineStart >= 0 {
		l.lines = append(l.lines, [2]int{lineStart, end})
	} else {
		l.lines = append(l.lines, [2]int{start, start})
	}
}

// prevWordEnd devolve o fim da palavra anterior a i (sem os espaços).
func prevWordEnd(s string, i int) int {
	for i > 0 && s[i-1] == ' ' {
		i--
	}
	return i
}

// PreferredSize devolve a largura da linha mais longa e a altura das linhas
// quebradas na largura efetiva (MaxWidth ou o último layout).
func (l *Label) PreferredSize() image.Point {
	if l.theme == nil {
		return image.Point{}
	}
	lineH, _, _ := l.metrics()
	l.ensureWrap(l.wrapWidth())
	var w int
	for _, ln := range l.lines {
		if lw := l.measure(l.text[ln[0]:ln[1]]); lw > w {
			w = lw
		}
	}
	return image.Pt(w, len(l.lines)*lineH)
}

// Layout guarda os bounds e reconcilia o wrap: se a altura preferida na
// largura nova difere da concedida, agenda um frame — o layout seguinte
// (que roda a cada frame) usa a preferência atualizada e converge.
func (l *Label) Layout(bounds image.Rectangle) {
	l.BaseWidget.Layout(bounds)
	if l.theme == nil {
		return
	}
	lineH, _, _ := l.metrics()
	l.ensureWrap(l.wrapWidth())
	if len(l.lines)*lineH != bounds.Dy() {
		l.Invalidate()
	}
}

// Draw desenha as linhas visuais a partir do topo, recortadas aos bounds.
func (l *Label) Draw(dst *image.RGBA) {
	if l.theme == nil {
		return
	}
	b := l.Bounds()
	c := l.col
	if l.danger {
		c = l.theme.Danger
	} else if c == (color.RGBA{}) {
		c = l.theme.Text
	}
	lineH, ascent, _ := l.metrics()
	l.ensureWrap(l.wrapWidth())
	view := render.Clip(dst, b, &l.clip)
	f := l.roleFont()
	y := b.Min.Y + ascent
	for _, ln := range l.lines {
		if y-ascent >= b.Max.Y {
			break
		}
		s := l.text[ln[0]:ln[1]]
		if f != nil {
			f.Draw(view, s, image.Pt(b.Min.X, y), c)
		} else {
			l.theme.DrawText(view, s, image.Pt(b.Min.X, y), c)
		}
		y += lineH
	}
}
