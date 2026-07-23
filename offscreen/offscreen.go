// Package offscreen renderiza árvores de widgets do JUIGo sem janela e sem
// OpenGL — todo o desenho do JUIGo é por software, então uma árvore pode ser
// montada, medida e desenhada direto em um *image.RGBA.
//
// Usos típicos: testes visuais (golden tests — a renderização é
// determinística: mesma árvore, tema e tamanho produzem sempre os mesmos
// bytes), screenshots para documentação e depuração de layout. Exemplo:
//
//	th, _ := juigo.DefaultTheme()
//	img := offscreen.Render(ui, th, 480, 320)
//	offscreen.SavePNG("captura.png", img)
//
// Para simular interação antes de desenhar, entregue eventos diretamente aos
// widgets (HandleEvent) entre o Render e um novo Render — ou monte e faça o
// layout uma vez com widget.Mount e Layout e desenhe quando quiser.
package offscreen

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
)

// Render monta a árvore com o tema (widget.Mount), faz o layout no retângulo
// (0,0)–(width,height), pinta o fundo com a cor do tema e desenha os
// widgets, devolvendo o buffer resultante. root ou tema nil devolvem um
// buffer vazio.
func Render(root widget.Widget, t *theme.Theme, width, height int) *image.RGBA {
	buf := image.NewRGBA(image.Rect(0, 0, width, height))
	if root == nil || t == nil {
		return buf
	}
	widget.Mount(root, t)
	root.Layout(buf.Bounds())
	render.FillRect(buf, buf.Bounds(), t.Background)
	root.Draw(buf)
	return buf
}

// SavePNG grava a imagem em disco como PNG.
func SavePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("offscreen: falha ao criar %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("offscreen: falha ao codificar PNG: %w", err)
	}
	return nil
}
