package uitest_test

import (
	"bytes"
	"fmt"
	"image"
	"testing"
	"time"

	"juigo"
	"juigo/anim"
	"juigo/uitest"
)

// TestIncrementalIgualCompleto é a rede de segurança das dirty regions: após
// CADA interação, o frame renderizado incrementalmente (só as regiões
// danificadas repintadas sobre o buffer persistente) deve ser BYTE A BYTE
// idêntico a uma repintura completa do zero. Se qualquer widget pintar fora
// do dano que reportou — ou reportar dano de menos — este teste acusa.
func TestIncrementalIgualCompleto(t *testing.T) {
	valor := juigo.NewState("")
	volume := juigo.NewState(0.3)
	plano := juigo.NewState("free")
	notif := juigo.NewState(true)

	campo := juigo.NewInput("Digite…").BindValue(valor)
	eco := juigo.NewText("").BindText(juigo.Map(valor, func(s string) string {
		return "eco: " + s
	}))
	deslizante := juigo.NewSlider(0, 1).BindValue(volume)
	lista := juigo.NewVBox().Gap(0).Pad(0)
	for i := 0; i < 30; i++ {
		lista.Add(juigo.NewText(fmt.Sprintf("linha %02d", i)))
	}
	modal := juigo.NewModal(juigo.NewVBox(juigo.NewText("Diálogo"), juigo.NewInput("interno")))

	ui := juigo.NewVBox(
		eco.Center(),
		juigo.NewHBox(
			juigo.Grow(campo, 1),
			juigo.Tooltip(juigo.NewButton("Enviar", nil), "dica"),
		),
		juigo.NewHBox(
			juigo.NewRadio("Grátis", "free").BindValue(plano),
			juigo.NewRadio("Pro", "pro").BindValue(plano),
			juigo.NewSpacer(),
			juigo.NewCheckbox("Notif").BindChecked(notif),
		),
		deslizante,
		juigo.NewProgressBar(0, 1).BindValue(volume),
		juigo.NewDropdown("Baixa", "Média", "Alta"),
		juigo.NewButton("Abrir", modal.Show),
		juigo.Grow(juigo.NewScroll(lista), 1),
	).Pad(12)

	h := uitest.New(t, ui, 480, 460)
	th := h.Session().Theme()

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot() // repintou só o dano sobre o buffer persistente
		h.Session().InvalidateAll()
		completo := h.Screenshot() // referência: repintura total
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			diff := 0
			for i := range incremental.Pix {
				if incremental.Pix[i] != completo.Pix[i] {
					diff++
				}
			}
			t.Fatalf("%s: render incremental divergiu do completo em %d bytes", passo, diff)
		}
	}

	// Digitação (com eco reativo mudando OUTRO widget do outro lado da tela).
	h.Click(uitest.Placeholder("Digite…"))
	h.Type("olá, ação")
	verifica("digitação")

	// Seleção + edição.
	h.Key(juigo.KeyLeft, juigo.ModShift)
	h.Key(juigo.KeyBackspace)
	verifica("seleção e backspace")

	// Piscada do caret (timer).
	h.Advance(th.CaretBlink)
	verifica("caret apagado")
	h.Advance(th.CaretBlink)
	verifica("caret aceso")

	// Hover + tooltip (timer + camada passiva).
	h.Hover(uitest.Text("Enviar"))
	verifica("hover no botão")
	h.Advance(th.TooltipDelay)
	verifica("tooltip visível")
	h.MoveTo(campo.Bounds().Min)
	verifica("tooltip escondido")

	// Grupo de radios (watchers desmarcando o irmão) e checkbox.
	h.Click(uitest.Text("Pro"))
	verifica("radio Pro")
	h.Click(uitest.Text("Notif"))
	verifica("checkbox")

	// Arraste do slider (captura) refletindo na ProgressBar via State.
	meio := deslizante.Bounds().Min.Add(deslizante.Bounds().Size().Div(2))
	h.Drag(meio, meio.Add(image.Pt(120, 0)))
	verifica("arraste do slider")

	// Rolagem da lista.
	h.Scroll(uitest.OfType[*juigo.Scroll](), -2)
	verifica("rolagem")

	// Foco por Tab (contorno de foco).
	h.Key(juigo.KeyTab)
	verifica("tab")

	// Dropdown: abre popup (overlay), realça, fecha com Escape.
	h.Click(uitest.Text("Baixa"))
	verifica("popup aberto")
	h.Key(juigo.KeyDown)
	verifica("realce no popup")
	h.Key(juigo.KeyEscape)
	verifica("popup fechado")

	// Modal: abre (backdrop translúcido), digita dentro, fecha.
	h.Click(uitest.Text("Abrir"))
	verifica("modal aberto")
	h.Click(uitest.Placeholder("interno"))
	h.Type("çã")
	verifica("digitação no modal")
	h.Key(juigo.KeyEscape)
	verifica("modal fechado")

	// Animação (Tween em quadros de 16ms).
	anim.Tween(volume, 0.9, 160*time.Millisecond, anim.Linear)
	h.Advance(80 * time.Millisecond)
	verifica("animação na metade")
	h.Advance(80 * time.Millisecond)
	verifica("animação concluída")

	// SetText que muda o layout (título cresce e o diff de bounds cobre).
	valor.Set("um texto bem mais comprido para deslocar o eco")
	verifica("relayout por SetText")
}
