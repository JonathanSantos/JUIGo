// Package cells é o 7GUIs nº 7: uma mini-planilha com fórmulas. Como o
// JUIGo não tem widget de tabela (ver ../GAPS.md), a planilha é UM widget
// custom que desenha tudo — cabeçalhos, grade, valores e a seleção — e
// resolve o clique por coordenada; a edição acontece numa barra de fórmulas
// (padrão Excel), sincronizada com a célula selecionada.
//
// Fórmulas: número, texto, ou "=A1+B2+…" (soma de referências e literais).
// Referência a texto vale 0; ciclo exibe #ERR. A cada edição TODAS as
// células são recalculadas — simplificação deliberada para a grade A–H ×
// 1–12 (o 7GUIs original usa propagação por dependência em 26×100).
package cells

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/render"
)

const (
	colunas = 8  // A–H
	linhas  = 12 // 1–12
)

// Modelo guarda o texto cru e os valores calculados de cada célula.
type Modelo struct {
	bruto   map[string]string
	valores map[string]string
}

// Definir grava o texto cru da célula e recalcula a planilha.
func (m *Modelo) Definir(ref, texto string) {
	if texto == "" {
		delete(m.bruto, ref)
	} else {
		m.bruto[ref] = texto
	}
	m.recomputa()
}

// Bruto devolve o texto cru da célula (o que a barra de fórmulas edita).
func (m *Modelo) Bruto(ref string) string {
	return m.bruto[ref]
}

// Valor devolve o texto CALCULADO exibido na célula.
func (m *Modelo) Valor(ref string) string {
	return m.valores[ref]
}

// recomputa recalcula todas as células (com memoização por rodada).
func (m *Modelo) recomputa() {
	m.valores = make(map[string]string, len(m.bruto))
	for ref := range m.bruto {
		m.aval(ref, map[string]bool{})
	}
}

// ehRef reconhece uma referência de célula (letra + número).
func ehRef(s string) bool {
	if len(s) < 2 || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	_, err := strconv.Atoi(s[1:])
	return err == nil
}

// aval calcula (e memoiza) o valor de exibição da célula.
func (m *Modelo) aval(ref string, visitando map[string]bool) string {
	if v, ok := m.valores[ref]; ok {
		return v
	}
	if visitando[ref] {
		return "#ERR" // ciclo
	}
	visitando[ref] = true
	defer delete(visitando, ref)

	bruto := m.bruto[ref]
	resultado := bruto
	if strings.HasPrefix(bruto, "=") {
		soma := 0.0
		erro := false
		for _, termo := range strings.Split(bruto[1:], "+") {
			termo = strings.TrimSpace(termo)
			switch {
			case ehRef(termo):
				v := m.aval(termo, visitando)
				if v == "#ERR" {
					erro = true
					break
				}
				// Texto (e célula vazia) vale 0 numa soma.
				if n, err := strconv.ParseFloat(v, 64); err == nil {
					soma += n
				}
			default:
				n, err := strconv.ParseFloat(termo, 64)
				if err != nil {
					erro = true
					break
				}
				soma += n
			}
			if erro {
				break
			}
		}
		if erro {
			resultado = "#ERR"
		} else {
			resultado = strconv.FormatFloat(soma, 'f', -1, 64)
		}
	}
	m.valores[ref] = resultado
	return resultado
}

// refDe monta a referência da célula (coluna 0 = A, linha 0 = 1).
func refDe(col, lin int) string {
	return fmt.Sprintf("%c%d", 'A'+col, lin+1)
}

// planilha é o widget custom que desenha a grade inteira e resolve cliques
// por coordenada.
type planilha struct {
	juigo.BaseWidget
	m            *Modelo
	sel          string
	aoSelecionar func(ref string)
	clip         image.RGBA
}

// medidas devolve as dimensões da grade na escala do tema.
func (p *planilha) medidas() (cabW, colW, linH int) {
	th := p.Theme()
	return th.Px(32), th.Px(64), th.LineHeight() + th.Px(6)
}

func (p *planilha) PreferredSize() image.Point {
	if p.Theme() == nil {
		return image.Point{}
	}
	cabW, colW, linH := p.medidas()
	return image.Point{X: cabW + colunas*colW, Y: (linhas + 1) * linH}
}

// CentroDe devolve o centro da célula na tela (para testes e cliques).
func (p *planilha) CentroDe(ref string) image.Point {
	cabW, colW, linH := p.medidas()
	col := int(ref[0] - 'A')
	lin, _ := strconv.Atoi(ref[1:])
	b := p.Bounds()
	return image.Pt(
		b.Min.X+cabW+col*colW+colW/2,
		b.Min.Y+linH*lin+linH/2,
	)
}

// celulaEm converte um ponto na referência da célula ("" fora da grade).
func (p *planilha) celulaEm(pos image.Point) string {
	cabW, colW, linH := p.medidas()
	b := p.Bounds()
	col := (pos.X - b.Min.X - cabW) / colW
	lin := (pos.Y-b.Min.Y)/linH - 1
	if pos.X < b.Min.X+cabW || pos.Y < b.Min.Y+linH || col < 0 || col >= colunas || lin < 0 || lin >= linhas {
		return ""
	}
	return refDe(col, lin)
}

func (p *planilha) HandleEvent(ev juigo.Event) bool {
	if e, ok := ev.(juigo.MouseEvent); ok && e.Kind == juigo.MouseDown && e.Button == juigo.MouseButtonLeft {
		if ref := p.celulaEm(e.Pos); ref != "" && ref != p.sel {
			p.sel = ref
			if p.aoSelecionar != nil {
				p.aoSelecionar(ref)
			}
			p.Invalidate()
		}
		return true
	}
	return false
}

func (p *planilha) Draw(dst *image.RGBA) {
	th := p.Theme()
	if th == nil {
		return
	}
	cabW, colW, linH := p.medidas()
	b := p.Bounds()
	larg := cabW + colunas*colW
	alt := (linhas + 1) * linH

	render.FillRect(dst, image.Rect(b.Min.X, b.Min.Y, b.Min.X+larg, b.Min.Y+alt), th.InputBackground)

	// Cabeçalhos de coluna (A–H) e de linha (1–12).
	for c := 0; c < colunas; c++ {
		x := b.Min.X + cabW + c*colW
		th.DrawText(dst, string(rune('A'+c)), image.Pt(x+colW/2-th.Px(3), b.Min.Y+th.Ascent()+th.Px(3)), th.Placeholder)
	}
	for l := 0; l < linhas; l++ {
		y := b.Min.Y + (l+1)*linH
		th.DrawText(dst, strconv.Itoa(l+1), image.Pt(b.Min.X+th.Px(6), y+th.Ascent()+th.Px(3)), th.Placeholder)
	}

	// Linhas da grade.
	for c := 0; c <= colunas; c++ {
		x := b.Min.X + cabW + c*colW
		render.FillRect(dst, image.Rect(x, b.Min.Y, x+1, b.Min.Y+alt), th.InputBorder)
	}
	for l := 0; l <= linhas+1; l++ {
		y := b.Min.Y + l*linH
		render.FillRect(dst, image.Rect(b.Min.X, y, b.Min.X+larg, y+1), th.InputBorder)
	}

	// Valores, recortados à própria célula.
	for c := 0; c < colunas; c++ {
		for l := 0; l < linhas; l++ {
			v := p.m.Valor(refDe(c, l))
			if v == "" {
				continue
			}
			x := b.Min.X + cabW + c*colW
			y := b.Min.Y + (l+1)*linH
			celula := image.Rect(x+1, y+1, x+colW, y+linH)
			view := render.Clip(dst, celula, &p.clip)
			th.DrawText(view, v, image.Pt(x+th.Px(4), y+th.Ascent()+th.Px(3)), th.Text)
		}
	}

	// Seleção por cima da grade.
	col := int(p.sel[0] - 'A')
	lin, _ := strconv.Atoi(p.sel[1:])
	x := b.Min.X + cabW + col*colW
	y := b.Min.Y + lin*linH
	render.StrokeRect(dst, image.Rect(x, y, x+colW+1, y+linH+1), 2*th.BorderPx(), th.Accent)
}

// App expõe as peças da planilha (o launcher usa Raiz; os testes, o resto).
type App struct {
	M     *Modelo
	Plan  *planilha
	Barra *juigo.Input
	Raiz  juigo.Widget
}

// New monta a planilha com a barra de fórmulas sincronizada com a seleção.
func New() *App {
	m := &Modelo{bruto: map[string]string{}, valores: map[string]string{}}
	p := &planilha{m: m, sel: "A1"}
	selecao := juigo.NewState("A1")

	barra := juigo.NewInput("valor ou =A1+B2").OnChange(func(s string) {
		m.Definir(p.sel, s)
		p.Invalidate()
	})
	p.aoSelecionar = func(ref string) {
		selecao.Set(ref)
		barra.SetText(m.Bruto(ref)) // SetText não dispara OnChange
	}

	raiz := juigo.NewVBox(
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("").BindText(juigo.Map(selecao, func(r string) string { return r + ":" }))),
			juigo.Grow(barra, 1),
		),
		juigo.Grow(juigo.NewScroll(p).Horizontal(), 1),
	).Pad(16)
	return &App{M: m, Plan: p, Barra: barra, Raiz: raiz}
}

// UI monta a tela (conveniência para o launcher).
func UI() juigo.Widget {
	return New().Raiz
}
