package tempconv

import (
	"testing"

	"juigo"
	"juigo/uitest"
)

func TestConversaoDuasVias(t *testing.T) {
	h := uitest.New(t, UI(), 520, 120)

	celsius := h.Find(uitest.Placeholder("Celsius")).(*juigo.Input)
	fahrenheit := h.Find(uitest.Placeholder("Fahrenheit")).(*juigo.Input)

	// C → F.
	h.Click(uitest.Placeholder("Celsius"))
	h.Type("100")
	if fahrenheit.Text() != "212" {
		t.Fatalf("100°C deveria virar 212°F; got %q", fahrenheit.Text())
	}

	// F → C (limpa e digita do outro lado).
	h.Click(uitest.Placeholder("Fahrenheit"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("32")
	if celsius.Text() != "0" {
		t.Fatalf("32°F deveria virar 0°C; got %q", celsius.Text())
	}

	// Entrada inválida não propaga.
	h.Click(uitest.Placeholder("Celsius"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("abc")
	if fahrenheit.Text() != "32" {
		t.Fatalf("entrada inválida não deveria mexer no outro campo; got %q", fahrenheit.Text())
	}
}
