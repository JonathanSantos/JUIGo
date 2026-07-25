package juigo_test

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/chart"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/widget"
)

// corTinta é a cor fixa dos benchmarks de primitivas.
var corTinta = color.RGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xFF}

// BenchmarkFrameClaude1x é o BenchmarkFrame1x com o design system "papel e
// tinta": mede o custo do raio 10 com antialiasing e da tipografia por
// papéis sobre o mesmo conteúdo.
func BenchmarkFrameClaude1x(b *testing.B) {
	th, err := juigo.ClaudeTheme()
	if err != nil {
		b.Fatalf("ClaudeTheme: %v", err)
	}
	campo := juigo.NewInput("Digite aqui…")
	campo.SetText("Olá, ação! Texto de exemplo çãé")
	campo.HandleEvent(juigo.FocusEvent{Gained: true})
	ui := juigo.NewVBox(
		juigo.NewText("Digite algo e clique em Enviar").Subtitle().Center(),
		campo,
		juigo.NewText("31 caracteres").Right(),
		juigo.NewButton("Enviar", nil),
		juigo.NewCheckbox("Notificações"),
		juigo.NewSlider(0, 1),
		juigo.NewText("Volume: 30%").Right(),
	).Pad(16)
	juigo.Mount(ui, th)
	buf := image.NewRGBA(image.Rect(0, 0, 480, 320))
	ui.Layout(buf.Bounds())
	ui.Draw(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.FillRect(buf, buf.Bounds(), th.Background)
		ui.Layout(buf.Bounds())
		ui.Draw(buf)
	}
}

// BenchmarkStrokePolyline mede a série de um gráfico: 12 segmentos AA de
// espessura 2 atravessando um buffer 480×320.
func BenchmarkStrokePolyline(b *testing.B) {
	buf := image.NewRGBA(image.Rect(0, 0, 480, 320))
	pts := []image.Point{
		{10, 300}, {50, 120}, {90, 200}, {130, 60}, {170, 180}, {210, 90},
		{250, 240}, {290, 40}, {330, 160}, {370, 100}, {420, 220}, {470, 30},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.StrokePolyline(buf, pts, 2, corTinta)
	}
}

// BenchmarkCrossFade mede um quadro de transição de tela do Navigator: a
// mistura de dois retratos 480×320 (o custo por quadro do Fade).
func BenchmarkCrossFade(b *testing.B) {
	r := image.Rect(0, 0, 480, 320)
	a := image.NewRGBA(r)
	bb := image.NewRGBA(r)
	dst := image.NewRGBA(r)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.CrossFade(dst, a, bb, r, 0.5)
	}
}

// BenchmarkLabelDraw mede o desenho de um parágrafo de ~60 palavras com o
// wrap CACHEADO (o caminho quente de um Label estável).
func BenchmarkLabelDraw(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatal(err)
	}
	l := juigo.NewLabel(strings.Repeat("palavras que quebram sozinhas no papel ", 10))
	juigo.Mount(l, th)
	buf := image.NewRGBA(image.Rect(0, 0, 480, 320))
	l.Layout(image.Rect(16, 16, 464, 304))
	l.Draw(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Draw(buf)
	}
}

// BenchmarkLabelRewrap mede o pior caso do Label: reflow completo do
// parágrafo por mudança de largura (um resize por iteração).
func BenchmarkLabelRewrap(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatal(err)
	}
	l := juigo.NewLabel(strings.Repeat("palavras que quebram sozinhas no papel ", 10))
	juigo.Mount(l, th)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := 300 + (i%2)*100 // alterna a largura para invalidar o cache
		l.Layout(image.Rect(0, 0, w, 400))
		_ = l.PreferredSize()
	}
}

// BenchmarkChartLine mede um gráfico de linha completo (eixos, série AA e
// rótulos) em 300×180.
func BenchmarkChartLine(b *testing.B) {
	th, err := juigo.ClaudeTheme()
	if err != nil {
		b.Fatal(err)
	}
	c := chart.NewLine([]float64{4, 6, 5, 9, 7, 11, 10, 14, 12, 16, 15, 19}).Min(0)
	widget.Mount(c, th)
	buf := image.NewRGBA(image.Rect(0, 0, 300, 180))
	c.Layout(buf.Bounds())
	c.Draw(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Draw(buf)
	}
}

// BenchmarkListaScrollIncremental mede o caminho completo de um passo de
// rolagem numa List de 10 mil linhas: evento, revinculação do pool e o
// frame incremental sobre buffer persistente.
func BenchmarkListaScrollIncremental(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatal(err)
	}
	lista := juigo.NewList(10000,
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, i int) { t.SetText(nomes[i%len(nomes)]) },
	)
	s := widget.NewSession(th)
	s.Resize(image.Pt(480, 320))
	s.SetRoot(juigo.NewScroll(lista))
	buf := image.NewRGBA(image.Rect(0, 0, 480, 320))
	s.Render(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Direção alternada: a rolagem nunca satura no limite (limite = sem
		// dano = frame pulado, que mediria outra coisa).
		dy := -3.0
		if i%2 == 1 {
			dy = 3.0
		}
		s.Scroll(image.Pt(240, 160), 0, dy)
		s.Render(buf)
	}
}

// nomes evita SetText com fmt (alocação fora do que se quer medir).
var nomes = []string{
	"Ana Lima", "Bruno Reis", "Carla Dias", "Davi Rocha", "Elisa Prado",
	"Fábio Nunes", "Gabi Torres", "Heitor Melo",
}
