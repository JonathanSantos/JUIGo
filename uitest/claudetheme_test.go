package uitest_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestTemaClaudePorPixels confere os tokens centrais do design system
// "papel e tinta" na tela: papel no fundo, terracota no botão, e a troca
// claro↔escuro em runtime mantendo a terracota de ação.
func TestTemaClaudePorPixels(t *testing.T) {
	th, err := juigo.ClaudeTheme()
	if err != nil {
		t.Fatal(err)
	}
	ui := juigo.NewVBox(juigo.NewButton("Ação", nil)).Pad(20)
	h := uitest.NewWithTheme(t, ui, th, 300, 200)

	papel := color.RGBA{R: 0xFA, G: 0xF9, B: 0xF5, A: 0xFF}
	terracota := color.RGBA{R: 0xD9, G: 0x77, B: 0x57, A: 0xFF}
	grafite := color.RGBA{R: 0x26, G: 0x26, B: 0x24, A: 0xFF}

	img := h.Screenshot()
	if got := img.RGBAAt(2, 2); got != papel {
		t.Fatalf("o fundo deveria ser papel %v; veio %v", papel, got)
	}
	// Ponto no miolo do botão, longe do rótulo e da borda arredondada.
	btn := h.Find(uitest.Text("Ação")).Bounds()
	dentro := image.Pt(btn.Min.X+8, btn.Min.Y+btn.Dy()/2)
	if got := img.RGBAAt(dentro.X, dentro.Y); got != terracota {
		t.Fatalf("o botão deveria ser terracota %v; veio %v", terracota, got)
	}

	// Troca em runtime: grafite no fundo, a MESMA terracota na ação.
	escuro, err := juigo.ClaudeDarkTheme()
	if err != nil {
		t.Fatal(err)
	}
	h.Session().SetTheme(escuro)
	img = h.Screenshot()
	if got := img.RGBAAt(2, 2); got != grafite {
		t.Fatalf("no escuro o fundo deveria ser grafite %v; veio %v", grafite, got)
	}
	if got := img.RGBAAt(dentro.X, dentro.Y); got != terracota {
		t.Fatalf("no escuro o botão deveria seguir terracota %v; veio %v", terracota, got)
	}
}
