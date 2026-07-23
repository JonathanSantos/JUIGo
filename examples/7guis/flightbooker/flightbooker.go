// Package flightbooker é o 7GUIs nº 3: reserva de voo — agora inteiro na
// camada quick. Um Options escolhe só ida/ida e volta; dois quick.Date
// (máscara por filtro, valor time.Time) validam formato; a regra
// multi-fonte do campo de volta exige volta ≥ ida (só quando há volta) e o
// campo fica desabilitado com "só ida" (.Disabled). Campos inválidos ganham
// borda vermelha automática (BindInvalid via quick) e o botão Reservar fica
// preso à validade do formulário (Model().Invalid()).
package flightbooker

import (
	"time"

	"juigo"
	"juigo/quick"
)

// UI monta a tela de reserva.
func UI() juigo.Widget {
	inicial := time.Now().AddDate(0, 0, 7)

	tipo := quick.Options("Voo:", "só ida", "ida e volta")
	soIda := juigo.Map(tipo.State(), func(v string) bool { return v != "ida e volta" })

	ida := quick.Date("Ida:", "Data inválida (DD/MM/AAAA)").Required("Informe a ida")
	ida.Set(inicial)
	var volta *quick.DateField
	volta = quick.Date("Volta:", "Data inválida (DD/MM/AAAA)").
		Disabled(soIda).
		Rule(func() string {
			if soIda.Get() || volta.Value().IsZero() || ida.Value().IsZero() {
				return "" // sem volta (ou formato ainda inválido): nada a comparar
			}
			if volta.Value().Before(ida.Value()) {
				return "A volta deve ser depois da ida"
			}
			return ""
		}, tipo.State(), ida.State())
	volta.Set(inicial)

	f := quick.Form(tipo, ida, volta)
	reservar := juigo.BindDisabled(juigo.NewButton("Reservar", func() {
		f.Model().Submit(func() {
			msg := "Reservado: só ida em " + ida.Value().Format("02/01/2006")
			if !soIda.Get() {
				msg = "Reservado: ida " + ida.Value().Format("02/01/2006") +
					", volta " + volta.Value().Format("02/01/2006")
			}
			quick.Alert(msg)
		})
	}), f.Model().Invalid())

	return juigo.NewVBox(f, quick.Buttons(reservar)).Pad(16)
}
