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
	lista := juigo.NewList(200,
		func() *juigo.Text { return juigo.NewText("") },
		func(row *juigo.Text, i int) { row.SetText(fmt.Sprintf("linha %03d", i)) },
	)
	area := juigo.NewTextArea("obs…")
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
		juigo.NewGrid(2,
			juigo.NewText("Prio:"), juigo.NewDropdown("Baixa", "Média", "Alta"),
			juigo.NewText("Obs:"), juigo.Grow(area, 1),
		),
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

	// Rolagem da lista VIRTUALIZADA (pool reciclado + dano).
	h.Scroll(uitest.OfType[*juigo.Scroll](), -2)
	verifica("rolagem")
	h.Scroll(uitest.OfType[*juigo.Scroll](), -8)
	verifica("rolagem funda")

	// TextArea com SOFT WRAP: digita além da largura e navega visualmente.
	h.Click(uitest.Placeholder("obs…"))
	h.Type("uma observação bem comprida que certamente quebra em várias linhas visuais dentro da célula")
	verifica("textarea com wrap")
	h.Key(juigo.KeyUp)
	h.Key(juigo.KeyUp, juigo.ModShift)
	verifica("navegação e seleção no wrap")

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

	// Inspector: ligar desenha os contornos sobre tudo; desligar precisa
	// apagá-los DE UMA VEZ (regressão: fantasmas que sumiam aos poucos
	// conforme outras regiões repintavam).
	h.Key(juigo.KeyI, juigo.ModControl)
	verifica("inspector ligado")
	h.Key(juigo.KeyI, juigo.ModControl)
	verifica("inspector desligado")
}

// TestIncrementalNovosComponentes estende a rede de segurança das dirty
// regions aos componentes da rodada de gaps: Table (seleção + cabeçalho
// fixo sob rolagem), Popup ancorado (abrir, reposicionar, fechar), Scroll
// horizontal e o realce de erro do Input (BindInvalid).
func TestIncrementalNovosComponentes(t *testing.T) {
	selecionada := juigo.NewState(-1)
	tabela := juigo.NewTable([]string{"Nome", "Valor"}, 100, func(l, c int) string {
		return fmt.Sprintf("célula %d·%d", l, c)
	}).BindSelected(selecionada)
	invalido := juigo.NewState(false)
	campo := juigo.NewInput("campo").BindInvalid(invalido)
	largo := juigo.NewText("conteúdo largo que certamente não cabe na janela deste teste de rolagem horizontal")

	ui := juigo.NewVBox(
		campo,
		juigo.Grow(juigo.NewScroll(tabela), 1),
		juigo.NewScroll(largo).Horizontal(),
	).Pad(8)
	h := uitest.New(t, ui, 380, 300)

	verifica := func(passo string) {
		t.Helper()
		incremental := h.Screenshot()
		h.Session().InvalidateAll()
		completo := h.Screenshot()
		if !bytes.Equal(incremental.Pix, completo.Pix) {
			t.Fatalf("%s: render incremental divergiu do completo", passo)
		}
	}

	// Seleção na tabela: clique, troca externa e limpeza.
	h.ClickAt(tabela.RowRect(2).Min.Add(image.Pt(10, 5)))
	verifica("linha clicada")
	if selecionada.Get() != 2 {
		t.Fatalf("clique deveria selecionar a linha 2; got %d", selecionada.Get())
	}
	selecionada.Set(5)
	verifica("seleção externa")

	// Rolagem funda: o cabeçalho fica FIXO no topo da viewport.
	h.Scroll(uitest.OfType[*juigo.Scroll](), -12)
	verifica("tabela rolada")
	h.Scroll(uitest.OfType[*juigo.Scroll](), -40)
	verifica("tabela no fundo")

	// Realce de erro do campo liga e desliga.
	invalido.Set(true)
	verifica("campo inválido")
	invalido.Set(false)
	verifica("campo válido")

	// Rolagem horizontal do conteúdo largo.
	h.Session().Scroll(largo.Bounds().Min.Add(image.Pt(20, 5)), -4, 0)
	verifica("rolagem horizontal")

	// Popup ancorado: abre, reposiciona e fecha com Escape.
	pop := juigo.NewPopup(juigo.NewVBox(juigo.NewText("menu"), juigo.NewSlider(0, 1)))
	pop.ShowAt(image.Pt(60, 60))
	verifica("popup aberto")
	pop.ShowAt(image.Pt(180, 120))
	verifica("popup reposicionado")
	h.Key(juigo.KeyEscape)
	verifica("popup fechado")
	if h.Session().Overlay() != nil {
		t.Fatal("Escape deveria fechar o popup")
	}
}
