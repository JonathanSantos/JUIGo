package theme

import (
	"image/color"
	"math"
	"reflect"
	"testing"
)

// paletasDeTeste cobre a identidade e sementes variadas (com croma, sem
// croma, claras, escuras, só a ação) nos dois modos.
func paletasDeTeste() map[string]Palette {
	base := map[string]Palette{
		"nanquim":  Nanquim(),
		"vazia":    {},
		"cobalto":  {Primary: rgb(0x2B59D9)},
		"musgo":    {Primary: rgb(0x2D6A4F), Neutral: rgb(0x6B7A60)},
		"amarelo":  {Primary: rgb(0xFFD400), Signal: rgb(0x00A3FF)},
		"branca":   {Primary: rgb(0xFFFFFF)},
		"rosa":     {Primary: rgb(0xE5487A), Neutral: rgb(0xA07080), Danger: rgb(0xFF0000)},
		"ciano":    {Primary: rgb(0x00ADD8), Success: rgb(0x00FF00), Warning: rgb(0xFFFF00)},
		"clarinha": {Primary: rgb(0xF6F0E6)},
	}
	out := map[string]Palette{}
	for nome, p := range base {
		out[nome+"/claro"] = p
		p.Dark = true
		out[nome+"/escuro"] = p
	}
	return out
}

// TestPaletaPassaNaAuditoria: qualquer semente produz um tema sem pares
// abaixo da régua WCAG — o construtor GARANTE o contraste.
func TestPaletaPassaNaAuditoria(t *testing.T) {
	for nome, p := range paletasDeTeste() {
		th, err := New(p)
		if err != nil {
			t.Fatal(err)
		}
		if issues := th.Audit(); len(issues) > 0 {
			t.Errorf("%s: %v", nome, issues)
		}
		if deltaE(th.Syntax.Keyword, th.Text) < 0.08 {
			t.Errorf("%s: a palavra-chave deveria se distinguir do texto comum", nome)
		}
	}
}

// TestPaletaPreencheTodasAsCores percorre por reflexão todo campo de cor do
// Theme: um token novo esquecido pela derivação aparece aqui.
func TestPaletaPreencheTodasAsCores(t *testing.T) {
	for nome, p := range paletasDeTeste() {
		th, err := New(p)
		if err != nil {
			t.Fatal(err)
		}
		verificaCores(t, nome, reflect.ValueOf(*th), "")
	}
}

func verificaCores(t *testing.T, nome string, v reflect.Value, prefixo string) {
	t.Helper()
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		switch c := v.Field(i).Interface().(type) {
		case color.RGBA:
			if c.A == 0 {
				t.Errorf("%s: %s%s ficou sem cor", nome, prefixo, f.Name)
			}
		case SyntaxPalette:
			verificaCores(t, nome, v.Field(i), f.Name+".")
		}
	}
}

// TestPaletaDeterministica: mesmas sementes, mesmas cores.
func TestPaletaDeterministica(t *testing.T) {
	a, _ := New(Palette{Primary: rgb(0x2B59D9), Dark: true})
	b, _ := New(Palette{Primary: rgb(0x2B59D9), Dark: true})
	a.Face, b.Face, a.MonoFace, b.MonoFace = nil, nil, nil, nil
	if a.Background != b.Background || a.ButtonHover != b.ButtonHover || a.Syntax != b.Syntax || a.Selection != b.Selection {
		t.Fatal("as mesmas sementes deveriam produzir as mesmas cores")
	}
}

// TestTintaSeInverteNoEscuro: ação sem croma vira a cor do texto no modo
// oposto; ação com croma mantém o matiz.
func TestTintaSeInverteNoEscuro(t *testing.T) {
	escuro, _ := Dark()
	if escuro.ButtonNormal != escuro.Text {
		t.Fatalf("tinta no escuro deveria ser o texto %v; veio %v", escuro.Text, escuro.ButtonNormal)
	}
	claro, _ := Default()
	if claro.ButtonNormal != rgb(0x111111) {
		t.Fatalf("tinta no claro deveria ficar #111111; veio %v", claro.ButtonNormal)
	}
	azul, _ := New(Palette{Primary: rgb(0x2B59D9), Dark: true})
	h := toOKLCh(azul.ButtonNormal).H - toOKLCh(rgb(0x2B59D9)).H
	if math.Abs(h) > 0.05 {
		t.Fatalf("o matiz da ação deveria se manter no escuro (Δ %.3f rad)", h)
	}
}

// TestParMesmasMetricas: claro e escuro compartilham métricas e fontes.
func TestParMesmasMetricas(t *testing.T) {
	c, _ := Default()
	e, _ := Dark()
	if c.Radius != e.Radius || c.Padding != e.Padding || c.Spacing != e.Spacing ||
		c.LineHeight() != e.LineHeight() || c.Title().LineHeight() != e.Title().LineHeight() {
		t.Fatal("o par claro/escuro deveria ter as mesmas métricas")
	}
}

// TestApplyPalettePreservaMetricas: re-colorir não mexe em forma e fonte.
func TestApplyPalettePreservaMetricas(t *testing.T) {
	th, _ := Default()
	th.Radius, th.Padding = 12, 14
	th.ApplyPalette(Palette{Primary: rgb(0x2D6A4F)})
	if th.Radius != 12 || th.Padding != 14 {
		t.Fatal("ApplyPalette não deveria tocar nas métricas")
	}
	if th.ButtonNormal == rgb(0x111111) {
		t.Fatal("ApplyPalette deveria trocar a ação")
	}
}

// TestContraste confere a régua WCAG nos extremos e a simetria.
func TestContraste(t *testing.T) {
	preto, branco := rgb(0x000000), rgb(0xFFFFFF)
	if r := Contrast(preto, branco); math.Abs(r-21) > 0.01 {
		t.Fatalf("preto/branco deveria ser 21:1; veio %.2f", r)
	}
	if Contrast(branco, preto) != Contrast(preto, branco) {
		t.Fatal("contraste deveria ser simétrico")
	}
	if r := Contrast(branco, branco); r != 1 {
		t.Fatalf("cor igual deveria ser 1:1; veio %.2f", r)
	}
}

// TestOKLChIdaEVolta: converter e voltar preserva a cor (±1 por canal).
func TestOKLChIdaEVolta(t *testing.T) {
	for _, c := range []color.RGBA{rgb(0x111111), rgb(0xF0A500), rgb(0x2B59D9), rgb(0xFFFFFF), rgb(0x000000), rgb(0xD92D20)} {
		got := toOKLCh(c).rgb()
		d := func(a, b uint8) int { return int(math.Abs(float64(a) - float64(b))) }
		if d(got.R, c.R) > 1 || d(got.G, c.G) > 1 || d(got.B, c.B) > 1 {
			t.Errorf("%v voltou como %v", c, got)
		}
	}
}

// TestAuditApontaProblema: um tema estragado à mão aparece na auditoria.
func TestAuditApontaProblema(t *testing.T) {
	th, _ := Default()
	th.Placeholder = rgb(0xEEEEEE)
	issues := th.Audit()
	if len(issues) != 1 || issues[0].Pair != "Placeholder/InputBackground" {
		t.Fatalf("esperava só o placeholder na auditoria; veio %v", issues)
	}
}

// TestClassicMantemPaleta: o Classic segue com o azul e os cantos retos.
func TestClassicMantemPaleta(t *testing.T) {
	c, err := Classic()
	if err != nil {
		t.Fatal(err)
	}
	if c.ButtonNormal != rgb(0x3B82F6) || c.Radius != 0 || c.ButtonBorder.A != 0 || c.Padding != 8 {
		t.Fatal("Classic deveria manter o azul, cantos retos, sem borda e Padding 8")
	}
	d, err := ClassicDark()
	if err != nil {
		t.Fatal(err)
	}
	if d.Background != rgb(0x1E2126) || d.Radius != 0 {
		t.Fatal("ClassicDark deveria manter o grafite azulado e cantos retos")
	}
}

// deltaE é a distância perceptual (OKLab) entre duas cores: acima de ~0.08
// a diferença salta aos olhos mesmo com luminância parecida.
func deltaE(a, b color.RGBA) float64 {
	oa, ob := toOKLCh(a), toOKLCh(b)
	ax, ay := oa.C*math.Cos(oa.H), oa.C*math.Sin(oa.H)
	bx, by := ob.C*math.Cos(ob.H), ob.C*math.Sin(ob.H)
	return math.Sqrt((oa.L-ob.L)*(oa.L-ob.L) + (ax-bx)*(ax-bx) + (ay-by)*(ay-by))
}

// TestVarreduraDeSementes cobre o cubo RGB (passo 34) como ação, nos dois
// modos, e uma grade de sinais: a auditoria tem de sair vazia para QUALQUER
// semente — a garantia que a documentação promete.
func TestVarreduraDeSementes(t *testing.T) {
	for r := 0; r < 256; r += 34 {
		for g := 0; g < 256; g += 34 {
			for b := 0; b < 256; b += 34 {
				c := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
				for _, escuro := range []bool{false, true} {
					for _, p := range []Palette{
						{Primary: c, Dark: escuro},
						{Signal: c, Neutral: c, Dark: escuro},
					} {
						th, err := New(p)
						if err != nil {
							t.Fatal(err)
						}
						if issues := th.Audit(); len(issues) > 0 {
							t.Fatalf("%+v: %v", p, issues)
						}
					}
				}
			}
		}
	}
}
