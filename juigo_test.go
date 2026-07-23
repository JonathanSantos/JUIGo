package juigo_test

import (
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/offscreen"
)

// TestFachada garante que a API reexportada pela raiz monta, mede e desenha
// uma interface completa sem tocar nos subpacotes (exceto render, para o
// fundo) — o contrato de "import único" da DX.
func TestFachada(t *testing.T) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}

	valor := juigo.NewState("")
	campo := juigo.NewInput("Digite…").BindValue(valor)
	eco := juigo.Map(valor, func(s string) string { return "eco: " + s })

	ui := juigo.NewVBox(
		juigo.NewText("Título").Center(),
		juigo.NewHBox(
			juigo.Grow(campo, 1),
			juigo.NewButton("OK", nil),
		),
		juigo.NewText("").BindText(eco),
		juigo.NewCheckbox("Ativo"),
		juigo.Centered(juigo.NewSlider(0, 1)),
	).Pad(16)

	juigo.Mount(ui, th)

	// Interação pela fachada: eventos e estado reexportados.
	campo.HandleEvent(juigo.FocusEvent{Gained: true})
	for _, r := range "çã" {
		campo.HandleEvent(juigo.CharEvent{Rune: r})
	}
	campo.HandleEvent(juigo.KeyEvent{Key: juigo.KeyLeft, Mods: juigo.ModShift})
	if valor.Get() != "çã" {
		t.Fatalf("binding pela fachada: State = %q, esperado %q", valor.Get(), "çã")
	}

	// Renderiza pelo pacote offscreen e verifica que algo foi pintado.
	buf := offscreen.Render(ui, th, 480, 320)
	bg := th.Background
	painted := false
	for i := 0; i < len(buf.Pix); i += 4 {
		if buf.Pix[i] != bg.R || buf.Pix[i+1] != bg.G || buf.Pix[i+2] != bg.B {
			painted = true
			break
		}
	}
	if !painted {
		t.Fatal("Render pela fachada não pintou nenhum pixel")
	}

	// Com Grow, o campo ocupa a largura que o botão não usa.
	if campo.Bounds().Dx() <= th.InputMinWidthPx() {
		t.Fatalf("Grow deveria expandir o campo além da largura mínima; Dx = %d", campo.Bounds().Dx())
	}
}
