package uitest_test

import (
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestLabelQuebraPorPalavras: o parágrafo quebra na largura do layout, a
// altura preferida acompanha, e uma janela mais estreita rende mais linhas.
func TestLabelQuebraPorPalavras(t *testing.T) {
	texto := strings.Repeat("palavra comprida ", 12)
	l := juigo.NewLabel(texto)
	h := uitest.New(t, juigo.NewVBox(l), 320, 400)
	h.Layout() // segundo passe: a preferência já reflete a largura concedida

	th := h.Session().Theme()
	lineH := th.LineHeight()
	linhasLargas := l.PreferredSize().Y / lineH
	if linhasLargas < 2 {
		t.Fatalf("o texto deveria quebrar em várias linhas; rendeu %d", linhasLargas)
	}
	if larg := l.PreferredSize().X; larg > 320 {
		t.Fatalf("nenhuma linha deveria exceder a largura; mediu %d", larg)
	}

	estreito := juigo.NewLabel(texto)
	h2 := uitest.New(t, juigo.NewVBox(estreito), 180, 500)
	h2.Layout()
	if linhasEstreitas := estreito.PreferredSize().Y / lineH; linhasEstreitas <= linhasLargas {
		t.Fatalf("mais estreito deveria render mais linhas: %d ≤ %d", linhasEstreitas, linhasLargas)
	}
}

// TestLabelQuebrasDurasEMaxWidth: \n quebra sempre; MaxWidth dá altura
// determinística sem depender do layout.
func TestLabelQuebrasDurasEMaxWidth(t *testing.T) {
	l := juigo.NewLabel("um\ndois\ntrês").MaxWidth(300)
	h := uitest.New(t, juigo.NewVBox(l), 400, 300)
	th := h.Session().Theme()
	if got := l.PreferredSize().Y; got != 3*th.LineHeight() {
		t.Fatalf("três quebras duras deveriam render 3 linhas; altura %d", got)
	}

	// MaxWidth: preferência correta ANTES de qualquer layout.
	solto := juigo.NewLabel(strings.Repeat("ai ui ", 30)).MaxWidth(120)
	juigo.Mount(solto, th)
	if linhas := solto.PreferredSize().Y / th.LineHeight(); linhas < 3 {
		t.Fatalf("com MaxWidth(120) deveriam sair várias linhas; saíram %d", linhas)
	}
}

// TestLabelPapeisEIncremental: papéis tipográficos valem no parágrafo e o
// render incremental bate com o completo durante rebind e resize de split.
func TestLabelPapeisEIncremental(t *testing.T) {
	texto := juigo.NewState(strings.Repeat("conteúdo reativo ", 8))
	corpo := juigo.NewLabel("").BindText(texto)
	legenda := juigo.NewLabel(strings.Repeat("nota pequena ", 8)).Caption()
	h := uitest.New(t, juigo.NewVBox(corpo, legenda).Pad(10), 300, 400)
	h.Layout()

	th := h.Session().Theme()
	if corpo.PreferredSize().Y%th.LineHeight() != 0 {
		t.Fatal("o corpo deveria medir em linhas do corpo")
	}
	if legenda.PreferredSize().Y%th.Caption().LineHeight() != 0 {
		t.Fatal("a legenda deveria medir em linhas de legenda")
	}

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		for i := range incremental.Pix {
			if incremental.Pix[i] != completo.Pix[i] {
				t.Fatalf("%s: incremental divergiu do completo", passo)
			}
		}
	}
	verifica("frame inicial")
	texto.Set("texto novo, bem mais curto")
	h.Layout()
	verifica("rebind com menos linhas")
}
