package flightbooker

import (
	"image"
	"testing"

	"juigo"
	"juigo/uitest"
	"juigo/widget"
)

func centro(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }

func TestReservaDeVoo(t *testing.T) {
	h := uitest.New(t, UI(), 460, 320)

	inputs := []*juigo.Input{}
	for _, w := range h.FindAll(uitest.OfType[*juigo.Input]()) {
		inputs = append(inputs, w.(*juigo.Input))
	}
	if len(inputs) != 2 {
		t.Fatalf("deveria haver 2 campos de data; got %d", len(inputs))
	}
	campoIda, campoVolta := inputs[0], inputs[1]

	// Só ida: o campo de volta está desabilitado e a reserva, válida.
	if !widget.DisabledOf(campoVolta) {
		t.Fatal("com só ida, o campo de volta deveria estar desabilitado")
	}
	reservar := h.Find(uitest.Text("Reservar"))
	if widget.DisabledOf(reservar) {
		t.Fatal("com datas iniciais válidas, Reservar deveria estar habilitado")
	}

	// Data de ida inválida desabilita a reserva; o filtro barra letras.
	h.ClickAt(centro(campoIda.Bounds()))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("32/13/2026")
	if !widget.DisabledOf(reservar) {
		t.Fatal("data inválida deveria desabilitar Reservar")
	}
	h.Type("abc") // letras não entram no campo de data
	if campoIda.Text() != "32/13/2026" {
		t.Fatalf("o filtro deveria barrar letras; got %q", campoIda.Text())
	}
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("10/12/2026")

	// Ida e volta com volta ANTES da ida: inválido; corrigir habilita.
	h.Click(uitest.Text("só ida")) // abre o popup do dropdown
	h.Key(juigo.KeyDown)
	h.Key(juigo.KeyEnter) // seleciona "ida e volta"
	if widget.DisabledOf(campoVolta) {
		t.Fatal("com ida e volta, o campo de volta deveria habilitar")
	}
	h.ClickAt(centro(campoVolta.Bounds()))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("01/12/2026")
	if !widget.DisabledOf(reservar) {
		t.Fatal("volta antes da ida deveria desabilitar Reservar")
	}
	h.Key(juigo.KeyTab) // blur → touched: o erro passa a ser exibido
	if h.Find(uitest.Text("A volta deve ser depois da ida")) == nil {
		t.Fatal("a regra multi-fonte deveria exibir o erro no campo de volta")
	}
	h.ClickAt(centro(campoVolta.Bounds()))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("20/12/2026")
	if widget.DisabledOf(reservar) {
		t.Fatal("datas coerentes deveriam habilitar Reservar")
	}

	// Reservar abre o diálogo de confirmação com as datas.
	h.Click(uitest.Text("Reservar"))
	if h.Find(uitest.Text("Reservado: ida 10/12/2026, volta 20/12/2026")) == nil {
		t.Fatal("o diálogo deveria confirmar a reserva")
	}
}
