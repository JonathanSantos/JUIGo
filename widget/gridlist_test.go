package widget

import (
	"fmt"
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo/event"
)

func TestGridColunasELinhas(t *testing.T) {
	th := newTestTheme(t)

	l1 := NewText("Nome:")
	c1 := Grow(NewInput("nome"), 1)
	l2 := NewText("E-mail bem comprido:")
	c2 := Grow(NewInput("mail"), 1)
	g := NewGrid(2, l1, c1, l2, c2).Gap(8).Pad(0)
	Mount(g, th)
	g.Layout(image.Rect(0, 0, 400, 200))

	// Coluna 0 tem a largura do rótulo mais largo; as duas linhas alinham.
	if l1.Bounds().Dx() != l2.Bounds().Dx() {
		t.Fatalf("rótulos deveriam ter a mesma largura de coluna: %d vs %d", l1.Bounds().Dx(), l2.Bounds().Dx())
	}
	if c1.Bounds().Min.X != c2.Bounds().Min.X {
		t.Fatal("campos deveriam começar na mesma coluna")
	}
	// A coluna com Grow ocupa a sobra até a borda.
	if c1.Bounds().Max.X != 400 {
		t.Fatalf("campo deveria crescer até a borda; Max.X = %d", c1.Bounds().Max.X)
	}
	// Linhas empilhadas: campo 2 abaixo do campo 1.
	if c2.Bounds().Min.Y <= c1.Bounds().Max.Y {
		t.Fatal("a segunda linha deveria ficar abaixo da primeira")
	}
	// Altura da linha = filho mais alto (input > texto).
	if l1.Bounds().Dy() != c1.Bounds().Dy() {
		t.Fatalf("célula do rótulo deveria esticar à altura da linha: %d vs %d", l1.Bounds().Dy(), c1.Bounds().Dy())
	}

	// PreferredSize coerente: duas linhas + gap.
	pref := g.PreferredSize()
	want := c1.PreferredSize().Y + c2.PreferredSize().Y + 8
	if pref.Y != want {
		t.Fatalf("PreferredSize().Y = %d, esperado %d", pref.Y, want)
	}
}

func TestListVirtualizacao(t *testing.T) {
	th := newTestTheme(t)
	itens := make([]string, 10000)
	for i := range itens {
		itens[i] = fmt.Sprintf("item %04d", i)
	}
	criadas := 0
	lista := NewList(len(itens),
		func() *Text { criadas++; return NewText("") },
		func(row *Text, i int) { row.SetText(itens[i]) },
	)
	sc := NewScroll(lista)
	sc.SetTheme(th)
	sc.Layout(image.Rect(0, 0, 300, 100))

	// O Scroll enxerga o conteúdo inteiro…
	rowH := lista.PreferredSize().Y / len(itens)
	if rowH <= 0 {
		t.Fatal("altura de linha deveria ser positiva")
	}
	if lista.Bounds().Dy() != len(itens)*rowH {
		t.Fatalf("conteúdo lógico = %d, esperado %d", lista.Bounds().Dy(), len(itens)*rowH)
	}

	// …mas só as linhas visíveis viraram widgets.
	visiveis := len(lista.Children())
	esperado := 100/rowH + 2
	if visiveis == 0 || visiveis > esperado {
		t.Fatalf("linhas no pool = %d (esperado ≤ %d)", visiveis, esperado)
	}
	if criadas > esperado {
		t.Fatalf("widgets criados = %d — a virtualização deveria criar só os visíveis", criadas)
	}
	if lista.Children()[0].(*Text).Text() != "item 0000" {
		t.Fatalf("primeira linha vinculada = %q", lista.Children()[0].(*Text).Text())
	}

	// Rola para o fundo: o MESMO pool é revinculado aos últimos índices.
	sc.ScrollTo(1 << 30)
	sc.Layout(image.Rect(0, 0, 300, 100))
	antes := criadas
	buf := image.NewRGBA(image.Rect(0, 0, 300, 100))
	sc.Draw(buf)
	if criadas != antes {
		t.Fatalf("rolar não deveria criar widgets novos (criadas %d → %d)", antes, criadas)
	}
	ultima := lista.Children()[len(lista.Children())-1].(*Text)
	if ultima.Text() != "item 9999" {
		t.Fatalf("no fundo, a última linha deveria ser o item 9999; got %q", ultima.Text())
	}

	// Refresh revincula com dados novos.
	itens[9999] = "MUDOU"
	lista.Refresh()
	sc.Layout(image.Rect(0, 0, 300, 100))
	if ultima.Text() != "MUDOU" {
		t.Fatalf("Refresh deveria revincular; got %q", ultima.Text())
	}
}

func TestTextAreaSoftWrap(t *testing.T) {
	th := newTestTheme(t)
	ta := NewTextArea("")
	ta.SetTheme(th)
	ta.Layout(image.Rect(0, 0, 160, 200)) // estreito de propósito
	ta.HandleEvent(event.FocusEvent{Gained: true})

	for _, r := range "palavra outra mais uma linha comprida" {
		ta.HandleEvent(event.CharEvent{Rune: r})
	}
	ta.ensureWrap()

	// Uma linha real, várias visuais.
	if len(ta.lines) != 1 {
		t.Fatalf("linhas reais = %d, esperado 1", len(ta.lines))
	}
	if len(ta.vlines) < 2 {
		t.Fatalf("texto estreito deveria quebrar em ≥2 linhas visuais; got %d", len(ta.vlines))
	}
	// Cada linha visual cabe na largura útil e as quebras preferem espaços.
	inner := ta.innerWidth()
	for i, v := range ta.vlines {
		if w := th.MeasureString(ta.segment(v)); w > inner {
			t.Fatalf("vline %d não cabe: %dpx > %dpx", i, w, inner)
		}
		if i > 0 {
			anterior := ta.vlines[i-1]
			ultimaRune := ta.runes[ta.startIdx(v)-1]
			_ = anterior
			if ultimaRune != ' ' {
				t.Fatalf("quebra da vline %d deveria ser após espaço; rune anterior = %q", i, ultimaRune)
			}
		}
	}

	// Cima/Baixo navegam entre linhas VISUAIS preservando a coluna.
	ta.HandleEvent(event.KeyEvent{Key: event.KeyHome}) // início da linha REAL = 1ª visual
	if ta.visualOf(ta.Cursor()) != 0 {
		t.Fatal("Home deveria levar à primeira linha visual")
	}
	ta.HandleEvent(event.KeyEvent{Key: event.KeyDown})
	if got := ta.visualOf(ta.Cursor()); got != 1 {
		t.Fatalf("Down deveria ir à 2ª linha visual; got %d", got)
	}
	ta.HandleEvent(event.KeyEvent{Key: event.KeyUp})
	if got := ta.visualOf(ta.Cursor()); got != 0 {
		t.Fatalf("Up deveria voltar à 1ª linha visual; got %d", got)
	}

	// Palavra única maior que o campo quebra no meio (sem laço infinito).
	ta.SetText("supercalifragilisticoespialidoso_sem_espacos_para_quebrar")
	ta.ensureWrap()
	if len(ta.vlines) < 2 {
		t.Fatal("palavra gigante deveria quebrar no meio")
	}

	// Alargar o campo remove as quebras.
	ta.Layout(image.Rect(0, 0, 2000, 200))
	ta.ensureWrap()
	if len(ta.vlines) != 1 {
		t.Fatalf("campo largo não deveria quebrar; vlines = %d", len(ta.vlines))
	}
}
