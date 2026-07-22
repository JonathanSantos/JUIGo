package juigo

// repaintHook é registrado pela aplicação em execução (App) para que
// mudanças de estado fora do fluxo de eventos — setters de widgets e
// State.Set — agendem um redesenho automaticamente, sem Invalidate manual.
// Uma única aplicação por processo, coerente com o modelo single-threaded
// da biblioteca.
var repaintHook func()

// requestRepaint agenda um redesenho se houver uma aplicação em execução;
// fora de uma aplicação (ex.: testes headless), não faz nada.
func requestRepaint() {
	if repaintHook != nil {
		repaintHook()
	}
}
