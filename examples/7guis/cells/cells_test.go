package cells

import (
	"testing"

	"juigo"
	"juigo/uitest"
)

func TestPlanilhaComFormulas(t *testing.T) {
	a := New()
	h := uitest.New(t, a.Raiz, 700, 460)

	digita := func(ref, texto string) {
		h.ClickAt(a.Plan.CentroDe(ref))
		h.Click(uitest.Placeholder("valor ou =A1+B2"))
		h.Key(juigo.KeyA, juigo.ModControl)
		if texto == "" {
			h.Key(juigo.KeyBackspace)
			return
		}
		h.Type(texto)
	}

	// Valores e fórmula de soma.
	digita("A1", "5")
	digita("B1", "7")
	digita("C1", "=A1+B1")
	if v := a.M.Valor("C1"); v != "12" {
		t.Fatalf("C1 deveria calcular 12; got %q", v)
	}

	// Mudar uma dependência recalcula.
	digita("A1", "10")
	if v := a.M.Valor("C1"); v != "17" {
		t.Fatalf("C1 deveria recalcular para 17; got %q", v)
	}

	// Fórmula sobre fórmula.
	digita("D1", "=C1+3")
	if v := a.M.Valor("D1"); v != "20" {
		t.Fatalf("D1 deveria calcular 20; got %q", v)
	}

	// Texto vale 0 numa soma; sozinho, é exibido como texto.
	digita("E1", "oi")
	digita("F1", "=E1+A1")
	if v := a.M.Valor("E1"); v != "oi" {
		t.Fatalf("E1 deveria exibir o texto; got %q", v)
	}
	if v := a.M.Valor("F1"); v != "10" {
		t.Fatalf("texto deveria valer 0 na soma; got %q", v)
	}

	// Ciclo exibe #ERR.
	digita("A2", "=A2")
	if v := a.M.Valor("A2"); v != "#ERR" {
		t.Fatalf("ciclo deveria exibir #ERR; got %q", v)
	}
	digita("B2", "=C2")
	digita("C2", "=B2")
	if v := a.M.Valor("C2"); v != "#ERR" {
		t.Fatalf("ciclo indireto deveria exibir #ERR; got %q", v)
	}

	// Selecionar uma célula espelha o texto CRU na barra de fórmulas.
	h.ClickAt(a.Plan.CentroDe("C1"))
	if a.Barra.Text() != "=A1+B1" {
		t.Fatalf("a barra deveria exibir a fórmula crua; got %q", a.Barra.Text())
	}

	// Apagar limpa a célula.
	digita("B1", "")
	if v := a.M.Valor("B1"); v != "" {
		t.Fatalf("célula apagada deveria ficar vazia; got %q", v)
	}
	if v := a.M.Valor("C1"); v != "10" {
		t.Fatalf("C1 deveria recalcular sem B1; got %q", v)
	}
}
