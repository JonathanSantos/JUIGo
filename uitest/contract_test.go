package uitest_test

import (
	"image"
	"runtime/debug"
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/chart"
)

// todosOsWidgets devolve uma instância de cada widget da biblioteca.
func todosOsWidgets() map[string]juigo.Widget {
	return map[string]juigo.Widget{
		"Button":      juigo.NewButton("b", nil),
		"Calendar":    juigo.NewCalendar(),
		"Card":        juigo.NewCard(juigo.NewText("c")),
		"Checkbox":    juigo.NewCheckbox("c"),
		"CodeEditor":  juigo.NewCodeEditor().SetText("a\nb"),
		"Container":   juigo.NewContainer(juigo.NewText("c")),
		"Divider":     juigo.NewDivider(),
		"Dropdown":    juigo.NewDropdown("a", "b"),
		"Grid":        juigo.NewGrid(2, juigo.NewText("a"), juigo.NewText("b")),
		"HBox":        juigo.NewHBox(juigo.NewText("a")),
		"Image":       juigo.NewImage(image.NewRGBA(image.Rect(0, 0, 4, 4))),
		"Input":       juigo.NewInput("p").SetText("texto"),
		"Label":       juigo.NewLabel("um parágrafo"),
		"List":        juigo.NewTextList(3, func(int) string { return "l" }),
		"MenuBar":     juigo.NewMenuBar().Menu("M", juigo.Command{Title: "c"}),
		"Navigator":   juigo.NewNavigator(),
		"ProgressBar": juigo.NewProgressBar(0, 1),
		"Radio":       juigo.NewRadio("r", "v"),
		"Scroll":      juigo.NewScroll(juigo.NewText("s")),
		"Sized":       juigo.NewSized(juigo.NewText("s"), 10, 10),
		"Slider":      juigo.NewSlider(0, 1),
		"Spacer":      juigo.NewSpacer(),
		"SplitPane":   juigo.NewSplitPane(juigo.NewText("a"), juigo.NewText("b")),
		"Table":       juigo.NewTable([]string{"A"}, 2, func(int, int) string { return "x" }),
		"Tabs":        juigo.NewTabs().Add("um", juigo.NewText("p")),
		"Text":        juigo.NewText("t"),
		"TextArea":    juigo.NewTextArea("p").SetText("a\nb"),
		"Tree": juigo.NewTextTree(
			func() []string { return []string{"a"} },
			func(string) []string { return nil },
			func(id string) string { return id },
		),
		"VBox":  juigo.NewVBox(juigo.NewText("a")),
		"Spark": chart.NewSpark([]float64{1, 2, 3}),
		"Line":  chart.NewLine([]float64{1, 2, 3}),
		"Bars":  chart.NewBars([]float64{1, 2, 3}),
	}
}

// TestWidgetsToleramUsoAntesDoMount: sem tema e sem layout, nenhum widget
// pode derrubar a aplicação — consultas devolvem zero, desenho e eventos
// viram no-ops. (Código de biblioteca não entra em pânico.)
func TestWidgetsToleramUsoAntesDoMount(t *testing.T) {
	buf := image.NewRGBA(image.Rect(0, 0, 64, 64))
	events := []juigo.Event{
		juigo.FocusEvent{Gained: true},
		juigo.KeyEvent{Key: juigo.KeyRight},
		juigo.KeyEvent{Key: juigo.KeyEnter},
		juigo.CharEvent{Rune: 'x'},
		juigo.MouseEvent{Kind: juigo.MouseEnter, Pos: image.Pt(3, 3)},
		juigo.MouseEvent{Kind: juigo.MouseDown, Pos: image.Pt(3, 3)},
		juigo.MouseEvent{Kind: juigo.MouseMove, Pos: image.Pt(5, 5)},
		juigo.MouseEvent{Kind: juigo.MouseUp, Pos: image.Pt(5, 5)},
		juigo.ScrollEvent{Pos: image.Pt(3, 3), DY: -1},
		juigo.FocusEvent{Gained: false},
	}
	for name, w := range todosOsWidgets() {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s entrou em pânico antes do mount: %v\n%s", name, r, debug.Stack())
				}
			}()
			_ = w.PreferredSize()
			_ = w.Bounds()
			_ = w.Focusable()
			w.Draw(buf)
			for _, ev := range events {
				w.HandleEvent(ev)
			}
			w.Layout(buf.Bounds()) // layout sem tema também não pode falhar
			w.Draw(buf)
			_ = w.PreferredSize()
		})
	}
}

// TestSetterSincronizaStateSemDispararOnChange: a regra dos setters —
// mudar por código atualiza o widget E o State vinculado, em silêncio.
func TestSetterSincronizaStateSemDispararOnChange(t *testing.T) {
	disparos := 0

	marcado := juigo.NewState(false)
	check := juigo.NewCheckbox("c").BindChecked(marcado).OnChange(func(bool) { disparos++ })
	check.SetChecked(true)
	if !marcado.Get() {
		t.Error("Checkbox.SetChecked deveria sincronizar o State")
	}

	volume := juigo.NewState(0.0)
	slider := juigo.NewSlider(0, 1).BindValue(volume).OnChange(func(float64) { disparos++ })
	slider.SetValue(0.5)
	if volume.Get() != 0.5 {
		t.Errorf("Slider.SetValue deveria sincronizar o State; State = %v", volume.Get())
	}

	texto := juigo.NewState("")
	campo := juigo.NewInput("p").BindValue(texto).OnChange(func(string) { disparos++ })
	campo.SetText("novo")
	if texto.Get() != "novo" {
		t.Errorf("Input.SetText deveria sincronizar o State; State = %q", texto.Get())
	}

	notas := juigo.NewState("")
	area := juigo.NewTextArea("p").BindValue(notas).OnChange(func(string) { disparos++ })
	area.SetText("a\nb")
	if notas.Get() != "a\nb" {
		t.Errorf("TextArea.SetText deveria sincronizar o State; State = %q", notas.Get())
	}

	opcao := juigo.NewState("a")
	drop := juigo.NewDropdown("a", "b").BindValue(opcao).OnChange(func(string) { disparos++ })
	drop.SetValue("b")
	if opcao.Get() != "b" || drop.Selected() != 1 {
		t.Errorf("Dropdown.SetValue: State = %q, Selected() = %d", opcao.Get(), drop.Selected())
	}

	aba := juigo.NewState(0)
	tabs := juigo.NewTabs().Add("um", juigo.NewText("1")).Add("dois", juigo.NewText("2")).
		BindSelected(aba).OnChange(func(int) { disparos++ })
	tabs.Select(1)
	if aba.Get() != 1 {
		t.Errorf("Tabs.Select deveria sincronizar o State; State = %d", aba.Get())
	}

	data := juigo.NewState(time.Time{})
	dia := time.Date(2026, time.May, 4, 0, 0, 0, 0, time.Local)
	cal := juigo.NewCalendar().BindValue(data).OnChange(func(time.Time) { disparos++ })
	cal.SetValue(dia)
	if !data.Get().Equal(dia) {
		t.Errorf("Calendar.SetValue deveria sincronizar o State; State = %v", data.Get())
	}

	if disparos != 0 {
		t.Errorf("setters programáticos dispararam OnChange %d vez(es); esperado 0", disparos)
	}
}
