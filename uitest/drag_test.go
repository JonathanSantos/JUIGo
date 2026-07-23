package uitest_test

import (
	"image"
	"testing"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// fonteDeArrasto é um widget que inicia um arrasto no primeiro movimento
// com o botão pressionado — o padrão de widget FONTE.
type fonteDeArrasto struct {
	juigo.BaseWidget
	pressionado bool
}

func (f *fonteDeArrasto) PreferredSize() image.Point { return image.Pt(100, 28) }

func (f *fonteDeArrasto) HandleEvent(ev juigo.Event) bool {
	e, ok := ev.(juigo.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case juigo.MouseDown:
		if e.Button != juigo.MouseButtonLeft {
			return false
		}
		f.pressionado = true
		return true
	case juigo.MouseMove:
		if f.pressionado {
			f.pressionado = false
			juigo.StartDrag(42, "Item 42")
		}
		return true
	case juigo.MouseUp:
		f.pressionado = false
		return true
	}
	return false
}

// alvoDeArrasto aceita (ou não) qualquer payload e registra o que recebeu.
type alvoDeArrasto struct {
	juigo.BaseWidget
	aceita   bool
	recebido any
}

func (a *alvoDeArrasto) PreferredSize() image.Point      { return image.Pt(140, 48) }
func (a *alvoDeArrasto) CanDrop(payload any) bool        { return a.aceita }
func (a *alvoDeArrasto) Drop(payload any, _ image.Point) { a.recebido = payload }

// TestDragAndDrop cobre o ciclo do arrasto: fantasma e realce do alvo nos
// pixels, soltar entrega o payload só a quem aceita, e Escape cancela.
func TestDragAndDrop(t *testing.T) {
	fonte := &fonteDeArrasto{}
	sim := &alvoDeArrasto{aceita: true}
	nao := &alvoDeArrasto{}
	h := uitest.New(t, juigo.NewVBox(fonte, sim, nao).Pad(10).Gap(12), 320, 240)
	th := h.Session().Theme()
	pad := th.PaddingPx()
	h.Layout()
	origem := image.Pt(fonte.Bounds().Min.X+10, fonte.Bounds().Min.Y+10)
	meioSim := sim.Bounds().Min.Add(sim.Bounds().Size().Div(2))
	meioNao := nao.Bounds().Min.Add(nao.Bounds().Size().Div(2))

	// Arrasto completo até o alvo que aceita.
	h.Drag(origem, meioSim)
	if sim.recebido != 42 {
		t.Fatalf("soltar sobre o alvo deveria entregar o payload; veio %v", sim.recebido)
	}
	if h.Session().Dragging() {
		t.Fatal("após soltar, o arrasto deveria ter terminado")
	}

	// Passo a passo: fantasma segue o cursor e o alvo ganha o realce.
	sim.recebido = nil
	s := h.Session()
	h.MoveTo(origem)
	s.PointerDown(origem, juigo.MouseButtonLeft)
	s.PointerMove(meioSim)
	if !s.Dragging() || s.DragPayload() != 42 {
		t.Fatalf("o movimento com botão pressionado deveria iniciar o arrasto; payload=%v", s.DragPayload())
	}
	img := h.Screenshot()
	borda := sim.Bounds()
	if got := img.RGBAAt(borda.Min.X+borda.Dx()/2, borda.Min.Y); got != th.Accent {
		t.Fatalf("o alvo sob o cursor deveria ter o realce %v; veio %v", th.Accent, got)
	}
	fantasma := meioSim.Add(image.Pt(pad, pad))
	prefX := th.MeasureString("Item 42") + 2*pad
	if got := img.RGBAAt(fantasma.X+prefX/2, fantasma.Y+1); got != th.TooltipBackground {
		t.Fatalf("o fantasma deveria seguir o cursor; veio %v", got)
	}

	// Escape cancela sem soltar.
	h.Key(juigo.KeyEscape)
	if s.Dragging() {
		t.Fatal("Escape deveria cancelar o arrasto")
	}
	s.PointerUp(meioSim, juigo.MouseButtonLeft)
	if sim.recebido != nil {
		t.Fatal("após o cancelamento, soltar não deveria entregar nada")
	}

	// Alvo que não aceita: sem realce e sem entrega.
	s.PointerDown(origem, juigo.MouseButtonLeft)
	s.PointerMove(meioNao)
	img = h.Screenshot()
	bordaNao := nao.Bounds()
	if got := img.RGBAAt(bordaNao.Min.X+bordaNao.Dx()/2, bordaNao.Min.Y); got == th.Accent {
		t.Fatal("alvo que recusa o payload não deveria ganhar realce")
	}
	s.PointerUp(meioNao, juigo.MouseButtonLeft)
	if nao.recebido != nil {
		t.Fatalf("alvo que recusa não deveria receber Drop; veio %v", nao.recebido)
	}
	if s.Dragging() {
		t.Fatal("soltar em alvo nenhum deveria encerrar o arrasto")
	}
}
