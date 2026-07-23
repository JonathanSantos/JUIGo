// Package hooks liga a aplicação em execução (juigo.App) aos pacotes que não
// podem depender dela — state e widget — sem criar ciclos de import: o App
// registra as funções na inicialização e os widgets/estados as consomem.
//
// É um par de "ganchos" por processo, coerente com o modelo single-threaded
// de uma aplicação por vez do JUIGo. Fora de uma aplicação (testes headless,
// renderização offscreen), ficam nil e as operações viram no-ops — testes
// podem injetar implementações falsas.
package hooks

// Repaint é registrado pelo App para agendar um redesenho da interface.
var Repaint func()

// ClipboardRead e ClipboardWrite são registrados pelo App apontando para a
// área de transferência do sistema (GLFW).
var (
	ClipboardRead  func() string
	ClipboardWrite func(string)
)

// RequestRepaint agenda um redesenho se houver uma aplicação em execução.
func RequestRepaint() {
	if Repaint != nil {
		Repaint()
	}
}

// ReadClipboard lê o texto da área de transferência ("" sem aplicação).
func ReadClipboard() string {
	if ClipboardRead == nil {
		return ""
	}
	return ClipboardRead()
}

// WriteClipboard escreve o texto na área de transferência, se houver
// aplicação em execução.
func WriteClipboard(s string) {
	if ClipboardWrite == nil {
		return
	}
	ClipboardWrite(s)
}
