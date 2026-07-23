// Package hooks liga a aplicação em execução (juigo.App) aos pacotes que não
// podem depender dela — state e widget — sem criar ciclos de import: o App
// registra as funções na inicialização e os widgets/estados as consomem.
//
// É um par de "ganchos" por processo, coerente com o modelo single-threaded
// de uma aplicação por vez do JUIGo. Fora de uma aplicação (testes headless,
// renderização offscreen), ficam nil e as operações viram no-ops — testes
// podem injetar implementações falsas.
package hooks

import (
	"image"
	"time"
)

// Repaint é registrado pelo App para agendar um redesenho COMPLETO da
// interface (dano total). É a válvula de escape correta quando não se sabe a
// região afetada.
var Repaint func()

// Damage é registrado pelo App/harness para acumular DANO PARCIAL: a região
// dada precisa ser repintada e reenviada à GPU no próximo frame.
var Damage func(r image.Rectangle)

// Frame é registrado pelo App/harness para agendar um frame SEM acrescentar
// dano: se nada tiver sido danificado até o render, o frame é pulado.
var Frame func()

// ClipboardRead e ClipboardWrite são registrados pelo App apontando para a
// área de transferência do sistema (GLFW).
var (
	ClipboardRead  func() string
	ClipboardWrite func(string)
)

// Schedule é registrado pelo App: agenda fn para executar na main thread
// após o atraso dado (o loop usa WaitEventsTimeout para acordar a tempo) e
// devolve uma função de cancelamento. Base de comportamentos temporais como
// o cursor piscante e o atraso do tooltip.
var Schedule func(d time.Duration, fn func()) (cancel func())

// OpenOverlay e CloseOverlay são registrados pelo App para exibir/remover a
// camada de sobreposição (popups como o do Dropdown). O valor é um
// widget.Widget — tipado como any para evitar ciclo de import.
var (
	OpenOverlay  func(w any)
	CloseOverlay func(w any)
)

// ScheduleAfter agenda fn se houver aplicação em execução; sem aplicação,
// devolve um cancelamento inerte (fn nunca executa).
func ScheduleAfter(d time.Duration, fn func()) (cancel func()) {
	if Schedule == nil {
		return func() {}
	}
	return Schedule(d, fn)
}

// RequestRepaint agenda um redesenho COMPLETO se houver uma aplicação em
// execução; sem aplicação, não faz nada.
func RequestRepaint() {
	if Repaint != nil {
		Repaint()
	}
}

// AddDamage acumula dano parcial (e agenda um frame), se houver aplicação.
func AddDamage(r image.Rectangle) {
	if Damage != nil {
		Damage(r)
	}
}

// RequestFrame agenda um frame sem dano, se houver aplicação.
func RequestFrame() {
	if Frame != nil {
		Frame()
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
