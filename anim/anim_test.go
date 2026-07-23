package anim

import (
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/state"
)

// pump instala um scheduler falso (fila com cancelamento por id, como o
// real) e devolve uma função que executa o próximo quadro pendente (true se
// havia um).
func pump(t *testing.T) func() bool {
	t.Helper()
	type timer struct {
		id int
		fn func()
	}
	var fila []timer
	seq := 0
	hooks.Schedule = func(_ time.Duration, fn func()) func() {
		seq++
		id := seq
		fila = append(fila, timer{id: id, fn: fn})
		return func() {
			for i := range fila {
				if fila[i].id == id {
					fila = append(fila[:i], fila[i+1:]...)
					return
				}
			}
		}
	}
	t.Cleanup(func() { hooks.Schedule = nil })
	return func() bool {
		if len(fila) == 0 {
			return false
		}
		fn := fila[0].fn
		fila = fila[1:]
		fn()
		return true
	}
}

func TestEasingsContrato(t *testing.T) {
	for nome, e := range map[string]Easing{"Linear": Linear, "EaseIn": EaseIn, "EaseOut": EaseOut, "EaseInOut": EaseInOut} {
		if e(0) != 0 || e(1) != 1 {
			t.Fatalf("%s deveria mapear 0→0 e 1→1", nome)
		}
	}
	if EaseIn(0.5) >= 0.5 {
		t.Fatal("EaseIn deveria estar atrás do linear na metade")
	}
	if EaseOut(0.5) <= 0.5 {
		t.Fatal("EaseOut deveria estar à frente do linear na metade")
	}
}

func TestTweenProgrideECompleta(t *testing.T) {
	passo := pump(t)
	s := state.New(0.0)
	a := Tween(s, 1, 160*time.Millisecond, Linear) // 10 quadros de 16ms
	done := 0
	a.OnDone = func() { done++ }

	anterior := 0.0
	for i := 0; i < 9; i++ {
		if !passo() {
			t.Fatalf("faltou quadro %d", i)
		}
		if s.Get() <= anterior {
			t.Fatalf("progresso deveria ser monotônico: %v → %v", anterior, s.Get())
		}
		anterior = s.Get()
	}
	if done != 0 || !a.Running() {
		t.Fatal("não deveria ter completado antes do último quadro")
	}
	passo() // quadro final
	if s.Get() != 1 || done != 1 || a.Running() {
		t.Fatalf("final: valor=%v done=%d running=%v", s.Get(), done, a.Running())
	}
	if passo() {
		t.Fatal("não deveria haver quadros após completar")
	}
}

func TestRetargetEStop(t *testing.T) {
	passo := pump(t)
	s := state.New(0.0)

	primeira := Tween(s, 1, 160*time.Millisecond, Linear)
	doneA := false
	primeira.OnDone = func() { doneA = true }
	passo()
	passo()
	meio := s.Get()

	// Retarget: nova animação sobre o MESMO state para a anterior e parte
	// do valor atual.
	segunda := Tween(s, 0, 160*time.Millisecond, Linear)
	if primeira.Running() {
		t.Fatal("retarget deveria parar a animação anterior")
	}
	if doneA {
		t.Fatal("animação interrompida não dispara OnDone")
	}
	passo()
	if s.Get() >= meio {
		t.Fatalf("nova animação deveria partir de %v em direção a 0; got %v", meio, s.Get())
	}

	// Stop congela onde está.
	valor := s.Get()
	segunda.Stop()
	for passo() {
	}
	if s.Get() != valor || segunda.Running() {
		t.Fatalf("Stop deveria congelar em %v; got %v", valor, s.Get())
	}
}

func TestTweenImediato(t *testing.T) {
	// Sem scheduler (headless puro): salta direto ao alvo.
	s := state.New(2.0)
	a := Tween(s, 7, 300*time.Millisecond, nil)
	if s.Get() != 7 || a.Running() {
		t.Fatalf("sem aplicação deveria saltar ao alvo; valor=%v", s.Get())
	}

	// Duração zero idem, mesmo com scheduler.
	passo := pump(t)
	b := Tween(s, 3, 0, nil)
	if s.Get() != 3 || b.Running() || passo() {
		t.Fatalf("duração zero deveria completar imediatamente; valor=%v", s.Get())
	}
}
