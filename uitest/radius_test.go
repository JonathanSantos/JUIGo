package uitest_test

import (
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestVisualArredondadoEClassico fixa o contrato do Theme.Radius: por padrão
// os controles têm cantos arredondados (o arco poupa o pixel extremo do
// canto, que mostra o fundo da janela) e o botão ganha borda; com o tema
// clássico (Radius zero, botão sem borda) o desenho volta aos retângulos
// retos de antes — os dois visuais ficam à escolha do dev, em runtime.
func TestVisualArredondadoEClassico(t *testing.T) {
	btn := juigo.NewButton("OK", nil)
	campo := juigo.NewInput("Digite…")
	h := uitest.New(t, juigo.NewVBox(campo, btn).Pad(8), 240, 140)

	th := h.Session().Theme()
	img := h.Screenshot()
	b := btn.Bounds()
	if got := img.RGBAAt(b.Min.X, b.Min.Y); got != th.Background {
		t.Fatalf("canto do botão deveria mostrar o fundo %v; veio %v", th.Background, got)
	}
	if got := img.RGBAAt(b.Min.X+b.Dx()/2, b.Min.Y); got != th.ButtonBorder {
		t.Fatalf("beirada do botão deveria ser a borda %v; veio %v", th.ButtonBorder, got)
	}
	cb := campo.Bounds()
	if got := img.RGBAAt(cb.Min.X, cb.Min.Y); got != th.Background {
		t.Fatalf("canto do campo deveria mostrar o fundo %v; veio %v", th.Background, got)
	}

	// Troca para o visual clássico em runtime: cantos retos, sem borda.
	classico, err := juigo.ClassicTheme()
	if err != nil {
		t.Fatalf("ClassicTheme: %v", err)
	}
	h.Session().SetTheme(classico)
	img = h.Screenshot()
	b = btn.Bounds()
	if got := img.RGBAAt(b.Min.X, b.Min.Y); got != classico.ButtonNormal {
		t.Fatalf("no clássico o canto do botão deveria ser %v; veio %v", classico.ButtonNormal, got)
	}
	cb = campo.Bounds()
	if got := img.RGBAAt(cb.Min.X, cb.Min.Y); got != classico.InputBorder {
		t.Fatalf("no clássico o canto do campo deveria ser a borda %v; veio %v", classico.InputBorder, got)
	}
}
