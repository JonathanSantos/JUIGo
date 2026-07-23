package juigo_test

import (
	"image"
	"testing"

	"juigo"
	"juigo/render"
)

// benchUI monta a árvore da demo (título, input com texto, contador, botão,
// checkbox e slider) na escala dada, pronta para desenhar.
func benchUI(b *testing.B, scale float64, w, h int) (juigo.Widget, *image.RGBA, *juigo.Theme) {
	b.Helper()
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatalf("DefaultTheme: %v", err)
	}
	if err := th.SetScale(scale); err != nil {
		b.Fatalf("SetScale: %v", err)
	}

	campo := juigo.NewInput("Digite aqui…")
	campo.SetText("Olá, ação! Texto de exemplo çãé")
	campo.HandleEvent(juigo.FocusEvent{Gained: true})

	ui := juigo.NewVBox(
		juigo.NewText("Digite algo e clique em Enviar").Center(),
		campo,
		juigo.NewText("31 caracteres").Right(),
		juigo.NewButton("Enviar", nil),
		juigo.NewCheckbox("Notificações"),
		juigo.NewSlider(0, 1),
		juigo.NewText("Volume: 30%").Right(),
	).Pad(16)

	juigo.Mount(ui, th)
	buf := image.NewRGBA(image.Rect(0, 0, w, h))
	ui.Layout(buf.Bounds())
	ui.Draw(buf) // aquece o cache de glyphs
	return ui, buf, th
}

// benchFrame mede o custo de CPU de um frame completo (fundo + layout +
// desenho), exatamente o que o App faz por renderização — só fica de fora o
// upload da textura e o swap, que exigem GL.
func benchFrame(b *testing.B, scale float64, w, h int) {
	ui, buf, th := benchUI(b, scale, w, h)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.FillRect(buf, buf.Bounds(), th.Background)
		ui.Layout(buf.Bounds())
		ui.Draw(buf)
	}
}

func BenchmarkFrame1x(b *testing.B) { benchFrame(b, 1, 480, 320) }

func BenchmarkFrame2x(b *testing.B) { benchFrame(b, 2, 960, 640) }

// BenchmarkFrameTela2x estima o pior caso: uma janela ocupando uma tela
// retina inteira (~13", 2560×1600 pixels físicos).
func BenchmarkFrameTela2x(b *testing.B) { benchFrame(b, 2, 2560, 1600) }

// BenchmarkTeclaDigitada mede o caminho de evento de edição: inserir e
// apagar uma rune no input (inclui a remedição do cursor).
func BenchmarkTeclaDigitada(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatalf("DefaultTheme: %v", err)
	}
	campo := juigo.NewInput("")
	campo.SetTheme(th)
	campo.SetText("Olá, ação! Texto de exemplo")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		campo.HandleEvent(juigo.CharEvent{Rune: 'ç'})
		campo.HandleEvent(juigo.KeyEvent{Key: juigo.KeyBackspace})
	}
}
