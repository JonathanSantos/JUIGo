package juigo

// clipboardRead e clipboardWrite são registrados pela aplicação em execução
// (App) apontando para a área de transferência do sistema, via GLFW. Fora de
// uma aplicação (ex.: testes headless), ficam nil e as operações viram
// no-ops — testes podem injetar implementações falsas. Mesmo modelo do
// repaintHook: uma aplicação por processo, single-threaded.
var (
	clipboardRead  func() string
	clipboardWrite func(string)
)

// clipboardReadText lê o texto da área de transferência ("" sem aplicação).
func clipboardReadText() string {
	if clipboardRead == nil {
		return ""
	}
	return clipboardRead()
}

// clipboardWriteText escreve o texto na área de transferência, se houver
// aplicação em execução.
func clipboardWriteText(s string) {
	if clipboardWrite == nil {
		return
	}
	clipboardWrite(s)
}
