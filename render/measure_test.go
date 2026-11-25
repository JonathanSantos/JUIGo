package render

import (
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

func testFace(t testing.TB, size float64) font.Face {
	t.Helper()
	fnt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		t.Fatal(err)
	}
	return face
}

// A tabela de prefixos precisa bater, posição a posição, com a medição
// oficial de cada prefixo — é ela que posiciona cursor e seleção.
func TestPrefixWidthsParidade(t *testing.T) {
	face := testFace(t, 16)
	for _, s := range []string{"", "a", "AVATAR To Wa", "ação, coração — fim", "tofu 中文 ok", "  espaços  "} {
		runes := []rune(s)
		got := PrefixWidths(face, runes, nil)
		if len(got) != len(runes)+1 {
			t.Fatalf("%q: len = %d, esperado %d", s, len(got), len(runes)+1)
		}
		for i := range got {
			if want := MeasureText(face, string(runes[:i])); got[i] != want {
				t.Fatalf("%q[:%d]: prefixo = %d, MeasureText = %d", s, i, got[i], want)
			}
		}
	}
}

func TestPrefixWidthsReusaBuffer(t *testing.T) {
	face := testFace(t, 16)
	runes := []rune("texto de tamanho razoável para o teste")
	buf := PrefixWidths(face, runes, nil)
	allocs := testing.AllocsPerRun(50, func() { buf = PrefixWidths(face, runes, buf) })
	if allocs != 0 {
		t.Fatalf("PrefixWidths alocou %v vezes com buffer suficiente", allocs)
	}
}

// IndexAtX devolve a fronteira mais próxima; empate fica com a da esquerda.
func TestIndexAtX(t *testing.T) {
	widths := []int{0, 10, 20, 40}
	cases := []struct{ x, want int }{
		{-5, 0}, {0, 0}, {4, 0}, {5, 0}, {6, 1}, {10, 1}, {15, 1}, {16, 2}, {29, 2}, {30, 2}, {31, 3}, {40, 3}, {999, 3},
	}
	for _, c := range cases {
		if got := IndexAtX(widths, c.x); got != c.want {
			t.Errorf("IndexAtX(%d) = %d, esperado %d", c.x, got, c.want)
		}
	}
	if IndexAtX(nil, 10) != 0 || IndexAtX([]int{0}, 10) != 0 {
		t.Error("tabelas vazias devem devolver 0")
	}
}

func TestMeasurerParidadeECache(t *testing.T) {
	face := testFace(t, 16)
	var m Measurer
	for _, s := range []string{"", "Enviar", "Olá, mundo", "Enviar"} {
		if got, want := m.Measure(face, s), MeasureText(face, s); got != want {
			t.Fatalf("%q: Measure = %d, MeasureText = %d", s, got, want)
		}
	}
	if allocs := testing.AllocsPerRun(100, func() { m.Measure(face, "Olá, mundo") }); allocs != 0 {
		t.Fatalf("consulta com cache aquecido alocou %v vezes", allocs)
	}

	// Trocar de face descarta as medidas antigas.
	big := testFace(t, 32)
	if got, want := m.Measure(big, "Enviar"), MeasureText(big, "Enviar"); got != want {
		t.Fatalf("após trocar de face: %d, esperado %d", got, want)
	}

	// O cache não cresce sem limite.
	buf := make([]byte, 0, 8)
	for i := range measureCacheLimit * 3 {
		buf = append(buf[:0], byte('a'+i%26), byte('a'+i/26%26), byte('a'+i/676%26))
		m.Measure(big, string(buf))
	}
	if len(m.widths) > measureCacheLimit {
		t.Fatalf("cache com %d entradas, teto %d", len(m.widths), measureCacheLimit)
	}
}
