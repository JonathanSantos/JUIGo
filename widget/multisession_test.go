package widget

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/theme"
)

// TestDanoPorSessao é o alicerce do multi-janela: depois do mount, o
// Invalidate de um widget vai DIRETO à sessão dona — mexer numa janela não
// suja as outras. Antes do mount, o dano cai no fallback global dos hooks.
func TestDanoPorSessao(t *testing.T) {
	tema, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}
	temaB, err := theme.Default()
	if err != nil {
		t.Fatal(err)
	}

	textoA := NewText("janela A")
	textoB := NewText("janela B")
	sa := NewSession(tema)
	sb := NewSession(temaB)
	sa.Resize(image.Pt(200, 100))
	sb.Resize(image.Pt(200, 100))
	sa.SetRoot(NewVBox(textoA))
	sb.SetRoot(NewVBox(textoB))

	bufA := image.NewRGBA(image.Rect(0, 0, 200, 100))
	bufB := image.NewRGBA(image.Rect(0, 0, 200, 100))
	sa.Render(bufA)
	sb.Render(bufB)

	// Setter num widget da sessão B: só B fica suja.
	textoB.SetText("mudou")
	if região, _ := sa.Render(bufA); !região.Empty() {
		t.Fatalf("a sessão A não deveria ter dano; repintou %v", região)
	}
	if região, _ := sb.Render(bufB); região.Empty() {
		t.Fatal("a sessão B deveria repintar o texto mudado")
	}

	// Widget AINDA não montado em sessão nenhuma: o dano cai no fallback
	// global (que o App reparte entre as janelas).
	var fallback int
	prev := hooks.Damage
	hooks.Damage = func(image.Rectangle) { fallback++ }
	defer func() { hooks.Damage = prev }()

	solto := NewText("fora de sessão")
	solto.SetText("outro")
	if fallback == 0 {
		t.Fatal("sem sessão anexada, o dano deveria cair no fallback dos hooks")
	}

	// Depois de entrar numa árvore e ser montado, passa a danificar a dona.
	fallback = 0
	raiz := sb.Root().(*VBox)
	raiz.Add(solto)
	sb.Render(bufB) // o mount do frame anexa a sessão
	solto.SetText("na janela B")
	if fallback != 0 {
		t.Fatalf("depois do mount o dano deveria ir direto à sessão; fallback=%d", fallback)
	}
	if região, _ := sb.Render(bufB); região.Empty() {
		t.Fatal("a sessão B deveria repintar o widget recém-adotado")
	}
}
