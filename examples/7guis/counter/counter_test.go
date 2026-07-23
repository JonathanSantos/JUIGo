package counter

import (
	"testing"

	"juigo/uitest"
)

func TestContador(t *testing.T) {
	h := uitest.New(t, UI(), 320, 120)
	if h.Find(uitest.Text("0")) == nil {
		t.Fatal("o contador deveria começar em 0")
	}
	for i := 0; i < 3; i++ {
		h.Click(uitest.Text("Contar"))
	}
	if h.Find(uitest.Text("3")) == nil {
		t.Fatal("três cliques deveriam exibir 3")
	}
}
