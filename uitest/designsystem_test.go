package uitest_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestPapeisTipograficos confere a escala: Título > Subtítulo > corpo >
// legenda em altura de linha, nos temas Default e Claude (a serif Lora do
// Claude mede diferente da Go Bold, mas a ORDEM é invariante do sistema).
func TestPapeisTipograficos(t *testing.T) {
	titulo := juigo.NewText("Relatório").Title()
	sub := juigo.NewText("Relatório").Subtitle()
	corpo := juigo.NewText("Relatório")
	legenda := juigo.NewText("Relatório").Caption()
	ui := juigo.NewVBox(titulo, sub, corpo, legenda)

	verifica := func(nome string) {
		t.Helper()
		ht, hs, hc, hl := titulo.PreferredSize().Y, sub.PreferredSize().Y,
			corpo.PreferredSize().Y, legenda.PreferredSize().Y
		if !(ht > hs && hs > hc && hc > hl) {
			t.Fatalf("%s: a escala deveria decrescer Título>Subtítulo>corpo>legenda; veio %d/%d/%d/%d",
				nome, ht, hs, hc, hl)
		}
	}

	h := uitest.New(t, ui, 400, 300)
	verifica("Default")

	claude, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	h.Session().SetTheme(claude)
	h.Layout()
	verifica("Claude")
}

// TestBotoesHierarquia confere as três variantes por pixels no tema Claude:
// primário terracota, secundário na superfície com fio, ghost invisível em
// repouso e com fundo de hover sob o ponteiro.
func TestBotoesHierarquia(t *testing.T) {
	claude, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	prim := juigo.NewButton("Pagar", nil)
	sec := juigo.NewButton("Extrato", nil).Secondary()
	ghost := juigo.NewButton("Cancelar", nil).Ghost()
	ui := juigo.NewVBox(prim, sec, ghost).Pad(20).Gap(12)
	h := uitest.NewWithTheme(t, ui, claude, 300, 260)

	dentro := func(w juigo.Widget) image.Point {
		b := w.Bounds()
		return image.Pt(b.Min.X+8, b.Min.Y+b.Dy()/2)
	}
	img := h.Screenshot()
	if got, quer := img.RGBAAt(dentro(prim).X, dentro(prim).Y), claude.ButtonNormal; got != quer {
		t.Fatalf("primário deveria ser terracota %v; veio %v", quer, got)
	}
	if got, quer := img.RGBAAt(dentro(sec).X, dentro(sec).Y), claude.Surface; got != quer {
		t.Fatalf("secundário deveria ter fundo de superfície %v; veio %v", quer, got)
	}
	if got, quer := img.RGBAAt(dentro(ghost).X, dentro(ghost).Y), claude.Background; got != quer {
		t.Fatalf("ghost em repouso deveria mostrar o papel %v; veio %v", quer, got)
	}

	h.Hover(uitest.Text("Cancelar"))
	img = h.Screenshot()
	if got, quer := img.RGBAAt(dentro(ghost).X, dentro(ghost).Y), claude.HoverBackground; got != quer {
		t.Fatalf("ghost sob o ponteiro deveria ganhar hover %v; veio %v", quer, got)
	}
}

// TestCardEDivider confere superfície, fio e o respiro padrão do cartão.
func TestCardEDivider(t *testing.T) {
	claude, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	miolo := juigo.NewVBox(juigo.NewText("Conteúdo"), juigo.NewDivider(), juigo.NewText("Resto"))
	card := juigo.NewCard(miolo)
	ui := juigo.NewVBox(card).Pad(16)
	h := uitest.NewWithTheme(t, ui, claude, 320, 240)
	img := h.Screenshot()

	b := card.Bounds()
	centroX := b.Min.X + b.Dx()/2
	if got := img.RGBAAt(centroX, b.Min.Y+4); got != claude.Surface {
		t.Fatalf("o miolo do cartão deveria ser a superfície %v; veio %v", claude.Surface, got)
	}
	if got := img.RGBAAt(centroX, b.Min.Y); got != claude.SurfaceBorder {
		t.Fatalf("a beirada do cartão deveria ser o fio %v; veio %v", claude.SurfaceBorder, got)
	}
	// Respiro padrão: o conteúdo começa 2×Padding para dentro.
	pad := claude.Px(2 * claude.Padding)
	if got := miolo.Bounds().Min.X - b.Min.X; got != pad {
		t.Fatalf("o respiro interno deveria ser %dpx; veio %d", pad, got)
	}
}

// TestIncrementalIdentidade é a rede de segurança das dirty regions para os
// componentes novos: títulos serif, Card, Divider e variantes de botão.
func TestIncrementalIdentidade(t *testing.T) {
	claude, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	rotulo := juigo.NewText("Fatura").Title()
	ui := juigo.NewVBox(
		rotulo,
		juigo.NewCard(juigo.NewVBox(
			juigo.NewText("Cobrança").Subtitle(),
			juigo.NewDivider(),
			juigo.NewHBox(
				juigo.NewButton("Cancelar", nil).Ghost(),
				juigo.NewButton("Extrato", nil).Secondary(),
				juigo.NewButton("Pagar", nil),
			).Gap(8),
		).Gap(8)),
	).Pad(16).Gap(10)
	h := uitest.NewWithTheme(t, ui, claude, 420, 260)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			t.Fatalf("%s: render incremental divergiu do completo", passo)
		}
	}

	verifica("frame inicial")
	h.Hover(uitest.Text("Cancelar"))
	verifica("hover no ghost")
	h.Hover(uitest.Text("Pagar"))
	verifica("hover no primário")
	rotulo.SetText("Fatura de julho")
	verifica("título serif maior")
}
