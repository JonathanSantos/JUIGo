package uitest_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestBindNaoPrendeWidgetDescartado: widgets vinculados e depois removidos
// da árvore (padrão Clear+Add das listas reconstruídas) não podem ficar
// presos ao State — a assinatura é fraca e se cancela depois da coleta.
func TestBindNaoPrendeWidgetDescartado(t *testing.T) {
	titulo := juigo.NewState("a")
	ligado := juigo.NewState(false)
	box := juigo.NewVBox()
	h := uitest.New(t, box, 300, 200)

	for range 50 {
		box.Clear()
		box.Add(
			juigo.NewText("").BindText(titulo),
			juigo.BindDisabled(juigo.NewButton("x", nil), ligado),
			juigo.NewCheckbox("c").BindChecked(ligado),
		)
		h.Layout()
	}
	if n := titulo.Watchers(); n != 50 {
		t.Fatalf("antes da coleta: %d assinaturas, esperado 50", n)
	}

	runtime.GC()
	runtime.GC()
	titulo.Set("b") // assinaturas de donos coletados se cancelam aqui
	ligado.Set(true)

	if n := titulo.Watchers(); n != 1 {
		t.Fatalf("titulo: %d assinaturas vivas, esperado 1 (só a árvore atual)", n)
	}
	if n := ligado.Watchers(); n != 2 {
		t.Fatalf("ligado: %d assinaturas vivas, esperado 2 (botão e caixa atuais)", n)
	}
	// A árvore viva segue reagindo.
	if h.Find(uitest.Text("b")) == nil {
		t.Fatal("o Text vivo deveria ter recebido o novo valor")
	}
	if !h.Find(uitest.OfType[*juigo.Button]()).(*juigo.Button).Disabled() {
		t.Fatal("o botão vivo deveria ter sido desabilitado pelo State")
	}
}

// TestFocoNaoSobreviveARemocao: um campo focado que sai da árvore deixa de
// ser o focado — não recebe mais teclado nem mantém a piscada do cursor.
func TestFocoNaoSobreviveARemocao(t *testing.T) {
	campo := juigo.NewInput("campo")
	box := juigo.NewVBox(campo)
	h := uitest.New(t, box, 300, 200)

	h.Click(uitest.Placeholder("campo"))
	if h.Focused() != juigo.Widget(campo) {
		t.Fatal("pré-condição: o campo deveria estar focado")
	}

	box.Clear()
	box.Add(juigo.NewText("vazio"))
	h.Layout()

	if f := h.Focused(); f != nil {
		t.Fatalf("o foco deveria ter sido solto; focado: %T", f)
	}
	h.Type("x")
	if campo.Text() != "" {
		t.Fatalf("campo removido recebeu teclado: %q", campo.Text())
	}
	// A piscada parou: avançar o relógio não gera novos frames.
	h.Screenshot()
	h.Advance(5 * time.Second)
	if _, dirty := h.Session().Render(h.Screenshot()); dirty {
		t.Fatal("um campo removido não deveria seguir danificando a tela")
	}
}

// TestFocoSolteAoTrocarDeAba: o focado numa página que deixou de ser a
// ativa também perde o foco (a página dormente sai do mount).
func TestFocoSoltoAoTrocarDeAba(t *testing.T) {
	campo := juigo.NewInput("na aba 1")
	abas := juigo.NewTabs().Add("Um", campo).Add("Dois", juigo.NewText("outra"))
	h := uitest.New(t, abas, 400, 200)

	h.Click(uitest.Placeholder("na aba 1"))
	abas.Select(1)
	h.Layout()
	if f := h.Focused(); f == juigo.Widget(campo) {
		t.Fatal("o campo da aba dormente não deveria seguir focado")
	}
}

// TestLoadingAntesDoMountAnima: SetLoading(true) antes do primeiro mount
// (sem tema) precisa começar a animar quando o botão entra na árvore.
func TestLoadingAntesDoMountAnima(t *testing.T) {
	btn := juigo.NewButton("Enviar", nil)
	btn.SetLoading(true)
	h := uitest.New(t, juigo.NewVBox(btn), 300, 120)

	before := h.Screenshot()
	h.Advance(btn.Theme().SpinnerStep + time.Millisecond)
	after := h.Screenshot()
	if string(before.Pix) == string(after.Pix) {
		t.Fatal("o spinner deveria ter avançado um quadro depois do mount")
	}
}

// TestSpinnerParaForaDaArvore: um botão em loading descartado para de se
// reagendar (senão o timer o manteria vivo e repintando para sempre).
func TestSpinnerParaForaDaArvore(t *testing.T) {
	btn := juigo.NewButton("Enviar", nil)
	box := juigo.NewVBox(btn)
	h := uitest.New(t, box, 300, 120)
	btn.SetLoading(true)
	h.Layout()

	box.Clear()
	box.Add(juigo.NewText("pronto"))
	h.Layout()
	step := btn.Theme().SpinnerStep
	for range 5 { // deixa os quadros pendentes perceberem a remoção
		h.Advance(step + time.Millisecond)
		h.Screenshot()
	}
	h.Advance(100 * step)
	if _, dirty := h.Session().Render(h.Screenshot()); dirty {
		t.Fatal("o spinner de um botão removido deveria ter parado")
	}
}

// TestRowRectAntesDoMount: consultar geometria antes do mount devolve vazio
// em vez de derrubar a aplicação.
func TestRowRectAntesDoMount(t *testing.T) {
	tab := juigo.NewTable([]string{"A"}, 3, func(r, c int) string { return "x" })
	if r := tab.RowRect(0); !r.Empty() {
		t.Fatalf("RowRect antes do mount = %v, esperado vazio", r)
	}
}
