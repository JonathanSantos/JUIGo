package widget

import (
	"image"
	"math"
	"time"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
)

// axisLock trava um GESTO de rolagem no eixo dominante: no trackpad o dedo
// raramente anda perfeitamente reto, e sem a trava rolar para baixo mexeria
// um pouco na horizontal. O gesto expira quando os eventos param de chegar
// dentro da janela do tema (Theme.ScrollAxisLock); uma dominância FORTE do
// outro eixo (mais de 2×) retrava na hora — mudar de direção não espera.
type axisLock struct {
	locked   bool
	vertical bool
	cancel   func()
}

// filter aplica a trava ao delta do evento e devolve o delta só no eixo do
// gesto; timeout <= 0 desliga (deltas passam intactos).
func (a *axisLock) filter(dx, dy float64, timeout time.Duration) (float64, float64) {
	if timeout <= 0 {
		return dx, dy
	}
	adx, ady := math.Abs(dx), math.Abs(dy)
	switch {
	case !a.locked:
		a.locked = true
		a.vertical = ady >= adx
	case a.vertical && adx > 2*ady:
		a.vertical = false
	case !a.vertical && ady > 2*adx:
		a.vertical = true
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.cancel = hooks.ScheduleAfter(timeout, func() {
		a.locked = false
		a.cancel = nil
	})
	if a.vertical {
		return 0, dy
	}
	return dx, 0
}

// doubleClick detecta o segundo clique dentro da janela do tema
// (Theme.DoubleClick) e perto do primeiro — o gatilho do "selecionar a
// palavra" dos campos de texto.
type doubleClick struct {
	armed  bool
	at     image.Point
	cancel func()
}

// hit registra o clique em p e devolve true quando ele completa um duplo
// clique; window <= 0 desliga.
func (d *doubleClick) hit(p image.Point, window time.Duration, tolerance int) bool {
	if window <= 0 {
		return false
	}
	if d.armed && abs(p.X-d.at.X) <= tolerance && abs(p.Y-d.at.Y) <= tolerance {
		d.armed = false
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
		}
		return true
	}
	d.armed = true
	d.at = p
	if d.cancel != nil {
		d.cancel()
	}
	d.cancel = hooks.ScheduleAfter(window, func() {
		d.armed = false
		d.cancel = nil
	})
	return false
}

// wordRangeAt devolve a corrida homogênea que contém a coluna col (em
// runes): identificadores (letras, dígitos, '_' e runas não-ASCII),
// brancos, ou pontuação — o intervalo selecionado pelo duplo clique.
func wordRangeAt(runes []rune, col int) (start, end int) {
	if len(runes) == 0 {
		return 0, 0
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}
	if col < 0 {
		col = 0
	}
	class := charClass(runes[col])
	start, end = col, col+1
	for start > 0 && charClass(runes[start-1]) == class {
		start--
	}
	for end < len(runes) && charClass(runes[end]) == class {
		end++
	}
	return start, end
}

// charClass classifica a rune para o duplo clique: 0 = branco, 1 = parte
// de identificador, 2 = pontuação.
func charClass(r rune) int {
	switch {
	case r == ' ' || r == '\t' || r == '\n':
		return 0
	case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r >= 0x80:
		return 1
	default:
		return 2
	}
}
