package widget

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
)

// scrollFixture monta um Scroll de 100px de viewport com conteúdo de 10
// botões (~360px na escala 1).
func scrollFixture(t *testing.T) (*Scroll, *VBox, []*Button) {
	t.Helper()
	th := newTestTheme(t)
	var btns []*Button
	lista := NewVBox().Gap(0).Pad(0)
	for i := 0; i < 10; i++ {
		b := NewButton("item", nil)
		btns = append(btns, b)
		lista.Add(b)
	}
	s := NewScroll(lista)
	s.SetTheme(th)
	s.Layout(image.Rect(0, 0, 200, 100))
	return s, lista, btns
}

func TestScrollLayoutEClamp(t *testing.T) {
	s, lista, _ := scrollFixture(t)

	if lista.Bounds().Min.Y != 0 {
		t.Fatalf("sem rolagem, conteúdo deveria começar no topo; Min.Y = %d", lista.Bounds().Min.Y)
	}
	content := lista.Bounds().Dy()
	if content <= 100 {
		t.Fatalf("pré-condição: conteúdo (%d) deveria exceder a viewport", content)
	}

	// ScrollTo além do fim limita ao máximo válido.
	s.ScrollTo(99999)
	s.Layout(image.Rect(0, 0, 200, 100))
	if want := content - 100; s.Offset() != want {
		t.Fatalf("Offset após ScrollTo(∞) = %d, esperado %d", s.Offset(), want)
	}
	if lista.Bounds().Max.Y != 100 {
		t.Fatalf("no fim, a base do conteúdo deveria coincidir com a viewport; Max.Y = %d", lista.Bounds().Max.Y)
	}
}

func TestScrollEventoConsumoEBorbulha(t *testing.T) {
	s, lista, _ := scrollFixture(t)
	th := s.Theme()
	pos := image.Pt(50, 50)

	// Um passo de roda para baixo (DY negativo) rola o conteúdo.
	if !s.HandleEvent(event.ScrollEvent{Pos: pos, DY: -1}) {
		t.Fatal("rolagem com espaço disponível deveria ser consumida")
	}
	if want := th.Px(th.ScrollStep); s.Offset() != want {
		t.Fatalf("um passo deveria rolar %d px; rolou %d", want, s.Offset())
	}

	// No topo, rolar para cima não é consumido (borbulha para o ancestral).
	s.ScrollTo(0)
	s.Layout(image.Rect(0, 0, 200, 100))
	if s.HandleEvent(event.ScrollEvent{Pos: pos, DY: 1}) {
		t.Fatal("no topo, rolar para cima deveria propagar (false)")
	}

	// Deltas fracionários de trackpad acumulam até virar pixel.
	s.accum = 0
	passo := float64(th.Px(th.ScrollStep))
	fino := -1.0 / (passo * 2) // meio pixel por evento
	s.HandleEvent(event.ScrollEvent{Pos: pos, DY: fino})
	if s.Offset() != 0 {
		t.Fatalf("meio pixel não deveria mover; Offset = %d", s.Offset())
	}
	s.HandleEvent(event.ScrollEvent{Pos: pos, DY: fino})
	if s.Offset() != 1 {
		t.Fatalf("dois meios pixels deveriam mover 1px; Offset = %d", s.Offset())
	}

	// Conteúdo que cabe: nunca consome.
	menor := NewScroll(NewButton("só um", nil))
	menor.SetTheme(th)
	menor.Layout(image.Rect(0, 0, 200, 300))
	if menor.HandleEvent(event.ScrollEvent{Pos: pos, DY: -1}) {
		t.Fatal("conteúdo que cabe na viewport não deveria consumir rolagem")
	}

	_ = lista
}

func TestScrollHitTestNaoAtravessaRecorte(t *testing.T) {
	s, _, btns := scrollFixture(t)
	root := NewContainer(s)
	root.Layout(image.Rect(0, 0, 400, 400))

	// Clique dentro da viewport atinge o item visível.
	down := event.MouseEvent{Kind: event.MouseDown, Pos: image.Pt(50, 10), Button: event.MouseButtonLeft}
	if got := DispatchMouse(root, down); got != Widget(btns[0]) {
		t.Fatalf("clique na viewport deveria atingir o primeiro item; atingiu %T", got)
	}

	// Abaixo da viewport, os bounds estendidos do conteúdo oculto NÃO podem
	// receber eventos: o Scroll (100px) não contém o ponto.
	fora := event.MouseEvent{Kind: event.MouseDown, Pos: image.Pt(50, 200), Button: event.MouseButtonLeft}
	if got := DispatchMouse(root, fora); got != nil {
		t.Fatalf("clique abaixo da viewport não deveria atingir conteúdo oculto; atingiu %T", got)
	}
	if FocusableAt(root, image.Pt(50, 200)) != nil {
		t.Fatal("foco por clique também não deveria alcançar conteúdo oculto")
	}
}

func TestScrollDesenhoRecortado(t *testing.T) {
	s, _, _ := scrollFixture(t)
	th := s.Theme()

	buf := image.NewRGBA(image.Rect(0, 0, 240, 200))
	bg := th.Background
	render.FillRect(buf, buf.Bounds(), bg)
	s.Draw(buf)

	// Nada pode ser pintado abaixo da viewport (y >= 100), apesar de o
	// conteúdo laid out se estender até ~360px.
	for y := 100; y < 200; y++ {
		for x := 0; x < 240; x++ {
			i := buf.PixOffset(x, y)
			if buf.Pix[i] != bg.R || buf.Pix[i+1] != bg.G || buf.Pix[i+2] != bg.B {
				t.Fatalf("pixel pintado fora da viewport em (%d,%d)", x, y)
			}
		}
	}

	// E dentro da viewport há conteúdo pintado.
	painted := false
	for y := 0; y < 100 && !painted; y++ {
		for x := 0; x < 200; x++ {
			i := buf.PixOffset(x, y)
			if buf.Pix[i] != bg.R || buf.Pix[i+1] != bg.G || buf.Pix[i+2] != bg.B {
				painted = true
				break
			}
		}
	}
	if !painted {
		t.Fatal("a viewport deveria ter conteúdo desenhado")
	}
}
