package juigo

import (
	"testing"
	"time"
)

// Um timer cancelado por outro callback do MESMO lote não pode disparar.
func TestTimerQueueCancelDentroDoLote(t *testing.T) {
	var q timerQueue
	base := time.Unix(1000, 0)
	fired := []string{}

	var second int
	q.add(base.Add(1*time.Millisecond), func() {
		fired = append(fired, "primeiro")
		q.cancel(second)
	})
	second = q.add(base.Add(2*time.Millisecond), func() { fired = append(fired, "segundo") })

	q.runDue(base.Add(10 * time.Millisecond))
	if len(fired) != 1 || fired[0] != "primeiro" {
		t.Fatalf("disparos = %v, esperado só [primeiro]", fired)
	}
	if _, ok := q.next(); ok {
		t.Fatal("a fila deveria ter esvaziado")
	}
}

// Os vencidos rodam em ordem de vencimento, não de registro.
func TestTimerQueueOrdemDeVencimento(t *testing.T) {
	var q timerQueue
	base := time.Unix(1000, 0)
	var order []int
	q.add(base.Add(3*time.Millisecond), func() { order = append(order, 3) })
	q.add(base.Add(1*time.Millisecond), func() { order = append(order, 1) })
	q.add(base.Add(2*time.Millisecond), func() { order = append(order, 2) })
	q.add(base.Add(time.Hour), func() { order = append(order, 99) })

	q.runDue(base.Add(5 * time.Millisecond))
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("ordem = %v, esperado [1 2 3]", order)
	}
	if at, ok := q.next(); !ok || !at.Equal(base.Add(time.Hour)) {
		t.Fatalf("deveria restar só o timer futuro; next = %v %v", at, ok)
	}
}

// Um callback que se reagenda para o futuro roda uma vez por chamada.
func TestTimerQueueReagendamentoNaoRepete(t *testing.T) {
	var q timerQueue
	now := time.Unix(1000, 0)
	runs := 0
	var tick func()
	tick = func() {
		runs++
		q.add(now.Add(16*time.Millisecond), tick)
	}
	q.add(now, tick)
	q.runDue(now)
	if runs != 1 {
		t.Fatalf("execuções = %d, esperado 1", runs)
	}
}
