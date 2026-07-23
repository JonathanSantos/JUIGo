package timer

import (
	"testing"
	"time"

	"juigo"
	"juigo/uitest"
)

func TestCronometro(t *testing.T) {
	// A UI agenda o tween JÁ NA CONSTRUÇÃO — por isso ela precisa nascer
	// depois de o harness ligar os hooks (senão o anim, sem scheduler,
	// salta ao alvo). Ver GAPS.md: candidato a uitest.New com builder lazy.
	h := uitest.New(t, juigo.NewVBox(), 460, 260)
	h.Session().SetRoot(UI())
	h.Layout() // a raiz trocada precisa de geometria antes dos cliques

	// O relógio virtual avança 5s: o decorrido acompanha (quadros de 16ms).
	h.Advance(5 * time.Second)
	if h.Find(uitest.Text("5.0s")) == nil {
		t.Fatal("após 5s, o decorrido deveria exibir 5.0s")
	}

	// O cronômetro PARA na duração (15s): avança além e satura.
	h.Advance(15 * time.Second)
	if h.Find(uitest.Text("15.0s")) == nil {
		t.Fatal("o decorrido deveria saturar na duração (15.0s)")
	}

	// Reset zera e volta a correr.
	h.Click(uitest.Text("Reset"))
	if h.Find(uitest.Text("0.0s")) == nil {
		t.Fatal("Reset deveria zerar o decorrido")
	}
	h.Advance(2 * time.Second)
	if h.Find(uitest.Text("2.0s")) == nil {
		t.Fatal("após o Reset, o cronômetro deveria voltar a correr")
	}
}
