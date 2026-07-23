package uitest_test

import (
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestTrocaDeTemaEmRuntime cobre o claro ↔ escuro ao vivo: o fundo e os
// widgets re-renderizam com a nova paleta, e um widget com tema EXPLÍCITO
// mantém o próprio.
func TestTrocaDeTemaEmRuntime(t *testing.T) {
	claro, err := juigo.DefaultTheme()
	if err != nil {
		t.Fatalf("DefaultTheme: %v", err)
	}
	escuro, err := juigo.DarkTheme()
	if err != nil {
		t.Fatalf("DarkTheme: %v", err)
	}

	campo := juigo.NewInput("Digite…")
	fixo := juigo.NewText("sempre claro")
	fixo.SetTheme(claro) // tema explícito: sobrevive à troca global
	ui := juigo.NewVBox(campo, juigo.NewButton("OK", nil), fixo).Pad(8)

	h := uitest.NewWithTheme(t, ui, claro, 400, 200)

	// Fundo claro no início.
	img := h.Screenshot()
	if img.RGBAAt(2, 190) != claro.Background {
		t.Fatalf("fundo inicial = %v, esperado claro %v", img.RGBAAt(2, 190), claro.Background)
	}

	// Troca para o escuro: fundo e campo mudam de paleta.
	h.Session().SetTheme(escuro)
	img = h.Screenshot()
	if img.RGBAAt(2, 190) != escuro.Background {
		t.Fatalf("fundo pós-troca = %v, esperado escuro %v", img.RGBAAt(2, 190), escuro.Background)
	}
	dentro := campo.Bounds().Min.Add(campo.Bounds().Size().Div(2))
	if img.RGBAAt(dentro.X, dentro.Y) != escuro.InputBackground {
		t.Fatalf("campo pós-troca = %v, esperado %v", img.RGBAAt(dentro.X, dentro.Y), escuro.InputBackground)
	}

	// O widget com tema explícito continua com o claro.
	if fixo.Theme() != claro {
		t.Fatal("SetTheme explícito deveria sobreviver à troca global")
	}

	// A interface continua interativa após a troca.
	h.Click(uitest.Placeholder("Digite…"))
	h.Type("çã")
	if campo.Text() != "çã" {
		t.Fatalf("digitação pós-troca: %q", campo.Text())
	}

	// E a volta ao claro funciona.
	h.Session().SetTheme(claro)
	img = h.Screenshot()
	if img.RGBAAt(2, 190) != claro.Background {
		t.Fatal("voltar ao claro deveria restaurar o fundo")
	}
}
