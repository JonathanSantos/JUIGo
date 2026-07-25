package uitest_test

import (
	"image"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// celulaDoDia devolve o centro da célula do dia dado, na grade do mês
// exibido (domingo primeiro), a partir das métricas do tema.
func celulaDoDia(h *uitest.Harness, cal *juigo.Calendar, dia time.Time) image.Point {
	th := h.Session().Theme()
	b := cal.Bounds()
	cw, ch := th.Px(32), th.Px(26)
	headerH := th.LineHeight() + 2*th.Px(4)
	weekH := th.Caption().LineHeight() + th.Px(4)
	primeiro := time.Date(dia.Year(), dia.Month(), 1, 0, 0, 0, 0, dia.Location())
	inicio := primeiro.AddDate(0, 0, -int(primeiro.Weekday()))
	idx := int(dia.Sub(inicio).Hours() / 24)
	x := b.Min.X + (idx%7)*cw + cw/2
	y := b.Min.Y + headerH + weekH + (idx/7)*ch + ch/2
	return image.Pt(x, y)
}

// TestCalendarSelecionaENavega: clique num dia sincroniza o State (duas
// vias) e as setas trocam o mês.
func TestCalendarSelecionaENavega(t *testing.T) {
	data := juigo.NewState(time.Date(2026, time.July, 24, 0, 0, 0, 0, time.UTC))
	cal := juigo.NewCalendar().BindValue(data)
	h := uitest.New(t, juigo.NewVBox(cal), 300, 260)

	// Clique no dia 10 do mês exibido (julho/2026).
	h.ClickAt(celulaDoDia(h, cal, time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)))
	if got := data.Get(); got.Day() != 10 || got.Month() != time.July {
		t.Fatalf("clicar no dia 10 deveria selecioná-lo; veio %v", got)
	}

	// Seta ‹ recua o mês; clicar num dia de junho seleciona em junho.
	th := h.Session().Theme()
	b := cal.Bounds()
	h.ClickAt(image.Pt(b.Min.X+th.Px(16), b.Min.Y+th.Px(8))) // ‹
	h.ClickAt(celulaDoDia(h, cal, time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)))
	if got := data.Get(); got.Day() != 15 || got.Month() != time.June {
		t.Fatalf("após ‹, clicar no 15 deveria dar junho; veio %v", got)
	}

	// Duas vias: Set externo move a seleção e o mês exibido.
	data.Set(time.Date(2027, time.January, 3, 0, 0, 0, 0, time.UTC))
	if got := cal.Selected(); got.Day() != 3 || got.Month() != time.January {
		t.Fatalf("Set externo deveria mover a seleção; veio %v", got)
	}
}

// TestQuickDateComCalendario: o botão ao lado do campo abre o popup e
// escolher um dia preenche o texto DD/MM/AAAA e fecha.
func TestQuickDateComCalendario(t *testing.T) {
	ida := quick.Date("Ida:", "data inválida")
	f := quick.Form(ida).Submit("Reservar", nil)
	h := uitest.New(t, juigo.NewVBox(f).Pad(12), 460, 420)

	h.Click(uitest.Text("↓"))
	if h.Session().Overlay() == nil {
		t.Fatal("o botão deveria abrir o popup do calendário")
	}
	cal := h.Find(uitest.OfType[*juigo.Calendar]()).(*juigo.Calendar)
	hoje := time.Now()
	alvo := time.Date(hoje.Year(), hoje.Month(), 15, 0, 0, 0, 0, hoje.Location())
	h.ClickAt(celulaDoDia(h, cal, alvo))
	if h.Session().Overlay() != nil {
		t.Fatal("escolher um dia deveria fechar o popup")
	}
	if got := ida.Value(); got.Day() != 15 {
		t.Fatalf("o campo deveria valer o dia escolhido; veio %v", got)
	}
	campo := h.Find(uitest.Placeholder("DD/MM/AAAA")).(*juigo.Input)
	if campo.Text() != alvo.Format("02/01/2006") {
		t.Fatalf("o texto deveria refletir a data; veio %q", campo.Text())
	}
}
