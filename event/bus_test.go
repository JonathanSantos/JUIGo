package event

import "testing"

func TestBusSincrono(t *testing.T) {
	bus := NewBus()
	order := []string{}
	bus.Subscribe("t", func(any) { order = append(order, "a") })
	bus.Subscribe("t", func(any) { order = append(order, "b") })
	bus.Publish("t", nil)
	// Síncrono: os handlers já executaram quando Publish retorna.
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("Publish síncrono fora de ordem: %v", order)
	}
}
