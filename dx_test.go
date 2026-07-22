package juigo

import (
	"image"
	"testing"
)

// TestTemaAmbienteMount cobre a injeção de tema no mount: herança pela
// árvore, override explícito por SetTheme e tematização de subárvore.
func TestTemaAmbienteMount(t *testing.T) {
	th := newTestTheme(t)

	txt := NewText("a")
	btn := NewButton("b", nil)
	root := NewVBox(txt, btn)

	if txt.Theme() != nil {
		t.Fatal("widget recém-criado não deveria ter tema antes do mount")
	}
	propagateTheme(root, th)
	if txt.Theme() != th || btn.Theme() != th || root.Theme() != th {
		t.Fatal("mount deveria injetar o tema do App em toda a árvore")
	}

	// Override explícito sobrevive a novas propagações.
	custom := newTestTheme(t)
	btn.SetTheme(custom)
	propagateTheme(root, th)
	if btn.Theme() != custom {
		t.Fatal("SetTheme explícito não deveria ser sobrescrito pelo mount")
	}
	if txt.Theme() != th {
		t.Fatal("irmão sem override deveria continuar com o tema do App")
	}

	// Tema explícito em um container tematiza a subárvore inteira — e
	// propaga imediatamente, sem depender do mount do App.
	child := NewText("c")
	sub := NewVBox(child)
	sub.SetTheme(custom)
	if child.Theme() != custom {
		t.Fatal("SetTheme em container deveria propagar imediatamente aos filhos")
	}
	outer := NewVBox(sub)
	propagateTheme(outer, th)
	if outer.Theme() != th {
		t.Fatal("container externo deveria herdar o tema do App")
	}
	if child.Theme() != custom {
		t.Fatal("descendente deveria herdar o tema explícito do ancestral")
	}
}

// TestContainerMetricsPadrao garante que VBox sem Gap/Pad usa o Spacing do
// tema e padding zero, e que Gap/Pad são unidades lógicas escaladas.
func TestContainerMetricsPadrao(t *testing.T) {
	th := newTestTheme(t)

	a := NewButton("A", nil)
	b := NewButton("B", nil)
	v := NewVBox(a, b)
	propagateTheme(v, th)
	v.Layout(image.Rect(0, 0, 200, 300))

	gap := b.Bounds().Min.Y - a.Bounds().Max.Y
	if gap != th.SpacingPx() {
		t.Fatalf("espaçamento padrão = %d, esperado Theme.SpacingPx() = %d", gap, th.SpacingPx())
	}
	if a.Bounds().Min != image.Pt(0, 0) {
		t.Fatalf("sem Pad, primeiro filho deveria começar em (0,0); começou em %v", a.Bounds().Min)
	}

	// Gap/Pad são lógicos: na escala 2, viram o dobro de pixels.
	if err := th.SetScale(2); err != nil {
		t.Fatalf("SetScale(2): %v", err)
	}
	v.Gap(10).Pad(4)
	v.Layout(image.Rect(0, 0, 400, 600))
	if a.Bounds().Min != image.Pt(8, 8) {
		t.Fatalf("Pad(4) na escala 2 deveria posicionar o filho em (8,8); ficou %v", a.Bounds().Min)
	}
	gap = b.Bounds().Min.Y - a.Bounds().Max.Y
	if gap != 20 {
		t.Fatalf("Gap(10) na escala 2 = %d px, esperado 20", gap)
	}
}

// TestPreMountSemPanico garante que widgets sem tema (antes do mount) são
// inertes e seguros: tamanho zero, desenho nulo e eventos sem pânico.
func TestPreMountSemPanico(t *testing.T) {
	in := NewInput("placeholder")
	btn := NewButton("OK", nil)
	txt := NewText("olá")

	if in.PreferredSize() != (image.Point{}) || btn.PreferredSize() != (image.Point{}) || txt.PreferredSize() != (image.Point{}) {
		t.Fatal("PreferredSize antes do mount deveria ser zero")
	}

	buf := image.NewRGBA(image.Rect(0, 0, 10, 10))
	in.Draw(buf)
	btn.Draw(buf)
	txt.Draw(buf)

	typeString(in, "aç")
	if in.Text() != "aç" {
		t.Fatalf("edição antes do mount deveria funcionar; Text() = %q", in.Text())
	}

	// Após o mount, o Draw ressincroniza o cursor com o tema real.
	in.SetTheme(newTestTheme(t))
	in.Layout(image.Rect(0, 0, 200, 30))
	in.HandleEvent(FocusEvent{Gained: true})
	in.Draw(buf)
	if in.cursorX == 0 {
		t.Fatal("cursorX deveria ter sido recalculado no primeiro Draw pós-mount")
	}
}
