// Package flightbooker é o 7GUIs nº 3: reserva de voo. Um Dropdown escolhe
// só ida/ida e volta; dois campos de data (DD/MM/AAAA) validam formato e
// ordem (volta ≥ ida, só quando há volta); o botão Reservar fica preso à
// validade do formulário e confirma com um diálogo.
//
// Limitações do JUIGo registradas aqui (ver ../GAPS.md): não há campo de
// data dedicado (Input com validação de formato faz o papel, sem máscara
// nem calendário) e o 7GUIs original pinta o FUNDO do campo inválido de
// vermelho — o Input não expõe cor de fundo por instância, então o erro
// aparece como texto na cor Danger (o padrão da lib).
package flightbooker

import (
	"time"

	"juigo"
	"juigo/form"
	"juigo/quick"
)

const formatoData = "02/01/2006"

// UI monta a tela de reserva.
func UI() juigo.Widget {
	tipo := juigo.NewState("só ida")
	inicial := time.Now().AddDate(0, 0, 7).Format(formatoData)
	ida := juigo.NewState(inicial)
	volta := juigo.NewState(inicial)

	dataValida := func(msg string) form.Validator {
		return func(v string) string {
			if _, err := time.Parse(formatoData, v); err != nil {
				return msg
			}
			return ""
		}
	}
	temVolta := func() bool { return tipo.Get() == "ida e volta" }

	f := form.New(
		form.Field(ida, form.Required("Informe a ida"), dataValida("Data inválida (DD/MM/AAAA)")),
		// A volta só é validada quando existe: com "só ida" o campo fica
		// desabilitado e qualquer conteúdo dele é irrelevante — por isso a
		// regra multi-fonte em vez de validadores de campo.
		form.Rule("volta", func() string {
			if !temVolta() {
				return ""
			}
			dv, err := time.Parse(formatoData, volta.Get())
			if err != nil {
				return "Data inválida (DD/MM/AAAA)"
			}
			di, err := time.Parse(formatoData, ida.Get())
			if err == nil && dv.Before(di) {
				return "A volta deve ser depois da ida"
			}
			return ""
		}, volta, ida, tipo),
	)

	campoVolta := juigo.BindDisabled(
		juigo.NewInput("DD/MM/AAAA").BindValue(volta),
		juigo.Map(tipo, func(t string) bool { return t != "ida e volta" }),
	)

	reservar := juigo.BindDisabled(juigo.NewButton("Reservar", func() {
		f.Submit(func() {
			msg := "Reservado: só ida em " + ida.Get()
			if temVolta() {
				msg = "Reservado: ida " + ida.Get() + ", volta " + volta.Get()
			}
			quick.Alert(msg)
		})
	}), f.Invalid())

	return juigo.NewVBox(
		juigo.NewDropdown("só ida", "ida e volta").BindValue(tipo),
		juigo.NewGrid(2,
			juigo.NewText("Ida:"), juigo.Grow(juigo.NewInput("DD/MM/AAAA").BindValue(ida).OnBlur(func() { f.Touch(ida) }), 1),
			juigo.NewText(""), juigo.NewText("").BindText(f.ErrorOf(ida)).Danger(),
			juigo.NewText("Volta:"), juigo.Grow(campoVolta, 1),
			juigo.NewText(""), juigo.NewText("").BindText(f.ErrorOf("volta")).Danger(),
		),
		reservar,
	).Pad(16)
}
