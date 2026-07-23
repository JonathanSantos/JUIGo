package uitest_test

import (
	"image"
	"testing"

	"juigo"
	"juigo/uitest"
)

// TestInspectorDeDepuracao cobre o ciclo do inspector: atalho Ctrl/Cmd+I,
// realce do widget sob o ponteiro, crachá de informações e desligamento.
func TestInspectorDeDepuracao(t *testing.T) {
	btn := juigo.NewButton("Enviar", nil)
	campo := juigo.NewInput("Digite…")
	h := uitest.New(t, juigo.NewVBox(campo, btn).Pad(8), 400, 240)

	// Liga pelo atalho.
	h.Key(juigo.KeyI, juigo.ModControl)
	if !h.Session().InspectorEnabled() {
		t.Fatal("Ctrl+I deveria ligar o inspector")
	}

	// Sob o botão: realce rosa na borda dele e crachá presente.
	h.Hover(uitest.Text("Enviar"))
	img := h.Screenshot()
	borda := img.RGBAAt(btn.Bounds().Min.X, btn.Bounds().Min.Y)
	if borda.R < 0xC0 || borda.G > 0x80 {
		t.Fatalf("borda do alvo deveria ter o realce rosa; got %v", borda)
	}
	if !temCorProxima(img, 0x14, 0x16, 0x1A) {
		t.Fatal("o crachá escuro deveria estar na imagem")
	}

	// A interface continua interativa com o inspector ligado.
	h.Click(uitest.Placeholder("Digite…"))
	h.Type("çã")
	if campo.Text() != "çã" {
		t.Fatalf("digitação com inspector ligado: %q", campo.Text())
	}
	// Consome o dano pendente antes do desligamento, como o app real (um
	// frame por evento) — senão o InvalidateAll do hover com inspector
	// ligado vaza para o frame seguinte e mascara fantasmas do toggle.
	h.Screenshot()

	// Desliga: a imagem volta a ser a interface pura.
	h.Key(juigo.KeyI, juigo.ModSuper) // Cmd também funciona
	if h.Session().InspectorEnabled() {
		t.Fatal("Cmd+I deveria desligar o inspector")
	}
	img = h.Screenshot()
	th := h.Session().Theme()
	if got := img.RGBAAt(btn.Bounds().Min.X, btn.Bounds().Min.Y); got != th.ButtonNormal {
		t.Fatalf("sem inspector, a borda do botão deveria voltar ao normal; got %v", got)
	}

	// I sem modificador não liga (é tecla comum).
	h.Key(juigo.KeyI)
	if h.Session().InspectorEnabled() {
		t.Fatal("I sem Ctrl/Cmd não deveria ligar o inspector")
	}
}

// temCorProxima procura um pixel com a cor aproximada (blending do crachá).
func temCorProxima(img *image.RGBA, r, g, b uint8) bool {
	perto := func(a, c uint8) bool {
		d := int(a) - int(c)
		return d > -24 && d < 24
	}
	for i := 0; i < len(img.Pix); i += 4 {
		if perto(img.Pix[i], r) && perto(img.Pix[i+1], g) && perto(img.Pix[i+2], b) {
			return true
		}
	}
	return false
}
