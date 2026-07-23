package circles

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
	"github.com/JonathanSantos/JUIGo/widget"
)

func TestDesenhoComUndoRedo(t *testing.T) {
	m, ui := New()
	h := uitest.New(t, ui, 480, 360)

	tela := h.Find(uitest.OfType[*tela]()).(*tela)
	dentro := func(dx, dy int) image.Point { return tela.Bounds().Min.Add(image.Pt(dx, dy)) }

	// Sem histórico, os botões ficam presos.
	if !widget.DisabledOf(h.Find(uitest.Text("Desfazer"))) {
		t.Fatal("sem histórico, Desfazer deveria estar desabilitado")
	}

	// Cliques em área vazia criam círculos.
	h.ClickAt(dentro(60, 60))
	h.ClickAt(dentro(200, 120))
	if len(m.Circulos()) != 2 {
		t.Fatalf("dois cliques deveriam criar 2 círculos; got %d", len(m.Circulos()))
	}

	// Clique DENTRO de um círculo existente não cria outro.
	h.ClickAt(dentro(60, 60))
	if len(m.Circulos()) != 2 {
		t.Fatal("clique sobre um círculo não deveria criar outro")
	}

	// Undo/redo caminham pelos snapshots.
	h.Click(uitest.Text("Desfazer"))
	if len(m.Circulos()) != 1 {
		t.Fatalf("Desfazer deveria remover o 2º círculo; got %d", len(m.Circulos()))
	}
	h.Click(uitest.Text("Refazer"))
	if len(m.Circulos()) != 2 {
		t.Fatalf("Refazer deveria restaurá-lo; got %d", len(m.Circulos()))
	}

	// Hover realça o círculo sob o cursor.
	h.MoveTo(dentro(60, 60))
	if tela.hover < 0 {
		t.Fatal("o círculo sob o cursor deveria estar realçado")
	}
	h.MoveTo(dentro(300, 200))
	if tela.hover != -1 {
		t.Fatal("longe dos círculos, nada deveria estar realçado")
	}

	// Clique direito no círculo abre o POPUP ancorado; arrastar o slider
	// muda o diâmetro AO VIVO; fechar registra UMA entrada de undo.
	h.RightClickAt(dentro(60, 60))
	popup, ok := h.Session().Overlay().(*juigo.Popup)
	if !ok {
		t.Fatalf("clique direito deveria abrir um Popup; got %T", h.Session().Overlay())
	}
	if !popup.Shown() {
		t.Fatal("o popup deveria estar aberto")
	}
	sl := h.Find(uitest.OfType[*juigo.Slider]()).(*juigo.Slider)
	meio := sl.Bounds().Min.Add(sl.Bounds().Size().Div(2))
	h.Drag(meio, meio.Add(image.Pt(200, 0))) // arrasta ao máximo (diâmetro 200)
	if r := m.Circulos()[0].r; r != 100 {
		t.Fatalf("o slider no máximo deveria deixar o raio em 100; got %d", r)
	}
	h.Key(juigo.KeyEscape) // fecha o popup → 1 entrada de undo
	if h.Session().Overlay() != nil {
		t.Fatal("Escape deveria fechar o popup")
	}
	h.Click(uitest.Text("Desfazer"))
	if r := m.Circulos()[0].r; r == 100 {
		t.Fatal("Desfazer deveria reverter o ajuste de diâmetro")
	}
}
