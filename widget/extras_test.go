package widget

import (
	"image"
	"image/color"
	"testing"

	"juigo/event"
	"juigo/internal/hooks"
	"juigo/render"
	"juigo/state"
)

func TestProgressBar(t *testing.T) {
	th := newTestTheme(t)
	vol := state.New(0.25)
	p := NewProgressBar(0, 1).BindValue(vol)
	p.SetTheme(th)
	p.Layout(image.Rect(0, 0, 100, 8))

	if p.Value() != 0.25 {
		t.Fatalf("BindValue deveria adotar o valor; Value=%v", p.Value())
	}
	vol.Set(0.5)
	if p.Value() != 0.5 {
		t.Fatalf("Set deveria espelhar; Value=%v", p.Value())
	}
	p.SetValue(7)
	if p.Value() != 1 {
		t.Fatalf("SetValue deveria limitar ao Max; Value=%v", p.Value())
	}

	// Metade preenchida: pixel em x=25 na cor de destaque, x=75 no trilho.
	p.SetValue(0.5)
	buf := image.NewRGBA(image.Rect(0, 0, 100, 8))
	p.Draw(buf)
	if got := buf.RGBAAt(25, 4); got != th.Accent {
		t.Fatalf("metade esquerda deveria ser Accent; got %v", got)
	}
	if got := buf.RGBAAt(75, 4); got != th.InputBorder {
		t.Fatalf("metade direita deveria ser trilho; got %v", got)
	}
}

func TestImageEscalaEProporcao(t *testing.T) {
	th := newTestTheme(t)
	// Imagem 20×10 vermelha.
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	render.FillRect(src, src.Bounds(), color.RGBA{R: 0xFF, A: 0xFF})

	im := NewImage(src)
	im.SetTheme(th)
	if got := im.PreferredSize(); got != image.Pt(20, 10) {
		t.Fatalf("PreferredSize = %v, esperado 20×10 na escala 1", got)
	}

	// Bounds 40×40: proporção 2:1 → alvo 40×20 centralizado (y 10..30).
	im.Layout(image.Rect(0, 0, 40, 40))
	buf := image.NewRGBA(image.Rect(0, 0, 40, 40))
	im.Draw(buf)
	if got := buf.RGBAAt(20, 20); got.R != 0xFF {
		t.Fatalf("centro deveria ser vermelho; got %v", got)
	}
	if got := buf.RGBAAt(20, 5); got.R != 0 {
		t.Fatalf("faixa acima do alvo deveria ficar vazia; got %v", got)
	}
}

func TestRadioGrupoViaState(t *testing.T) {
	th := newTestTheme(t)
	plano := state.New("pro")
	var chosen string

	free := NewRadio("Grátis", "free").BindValue(plano)
	pro := NewRadio("Pro", "pro").BindValue(plano)
	free.OnChange(func(v string) { chosen = v })
	free.SetTheme(th)
	pro.SetTheme(th)
	free.Layout(image.Rect(0, 0, 120, 20))
	pro.Layout(image.Rect(0, 30, 120, 50))

	if free.Checked() || !pro.Checked() {
		t.Fatal("bind inicial deveria marcar só o rádio do valor atual")
	}

	// Clique em "Grátis": exclusividade emerge do State compartilhado.
	pos := image.Pt(10, 10)
	free.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: event.MouseButtonLeft})
	free.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: pos, Button: event.MouseButtonLeft})
	if !free.Checked() || pro.Checked() || plano.Get() != "free" || chosen != "free" {
		t.Fatalf("clique: free=%v pro=%v state=%q OnChange=%q", free.Checked(), pro.Checked(), plano.Get(), chosen)
	}

	// Selecionar um já marcado não dispara nada.
	chosen = ""
	free.HandleEvent(event.KeyEvent{Key: event.KeySpace})
	if chosen != "" {
		t.Fatal("re-selecionar um rádio marcado não deveria disparar OnChange")
	}

	// Set externo troca o grupo inteiro.
	plano.Set("pro")
	if free.Checked() || !pro.Checked() {
		t.Fatal("Set externo deveria mover a marcação")
	}
}

func TestTextAreaEdicaoMultilinha(t *testing.T) {
	th := newTestTheme(t)
	ta := NewTextArea("")
	ta.SetTheme(th)
	ta.Layout(image.Rect(0, 0, 300, 100))
	ta.HandleEvent(event.FocusEvent{Gained: true})

	typeString2 := func(s string) {
		for _, r := range s {
			ta.HandleEvent(event.CharEvent{Rune: r})
		}
	}

	// Enter cria linhas.
	typeString2("olá")
	ta.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	typeString2("ação")
	ta.HandleEvent(event.KeyEvent{Key: event.KeyEnter})
	typeString2("fim")
	if ta.Text() != "olá\nação\nfim" {
		t.Fatalf("Text() = %q", ta.Text())
	}
	if len(ta.lines) != 3 || ta.caretLine != 2 {
		t.Fatalf("lines=%d caretLine=%d", len(ta.lines), ta.caretLine)
	}

	// Home/End são por linha.
	ta.HandleEvent(event.KeyEvent{Key: event.KeyHome})
	if ta.Cursor() != len([]rune("olá\nação\n")) {
		t.Fatalf("Home deveria ir ao início da linha; cursor=%d", ta.Cursor())
	}

	// Cima preserva a coluna desejada (início da linha → início da linha).
	ta.HandleEvent(event.KeyEvent{Key: event.KeyUp})
	if ta.caretLine != 1 || ta.Cursor() != len([]rune("olá\n")) {
		t.Fatalf("Up: line=%d cursor=%d", ta.caretLine, ta.Cursor())
	}

	// End + Baixo: coluna alvo além do fim da última linha → clampa.
	ta.HandleEvent(event.KeyEvent{Key: event.KeyEnd}) // fim de "ação"
	ta.HandleEvent(event.KeyEvent{Key: event.KeyDown})
	if ta.caretLine != 2 || ta.Cursor() != len([]rune("olá\nação\nfim")) {
		t.Fatalf("Down além do fim: line=%d cursor=%d", ta.caretLine, ta.Cursor())
	}

	// Seleção multilinha + copiar preserva o \n.
	fake := ""
	hooks.ClipboardWrite = func(s string) { fake = s }
	hooks.ClipboardRead = func() string { return fake }
	defer func() { hooks.ClipboardRead, hooks.ClipboardWrite = nil, nil }()

	ta.HandleEvent(event.KeyEvent{Key: event.KeyA, Mods: event.ModControl})
	ta.HandleEvent(event.KeyEvent{Key: event.KeyC, Mods: event.ModControl})
	if fake != "olá\nação\nfim" {
		t.Fatalf("copiar tudo = %q", fake)
	}

	// Colar substituindo tudo mantém as quebras.
	fake = "a\nb"
	ta.HandleEvent(event.KeyEvent{Key: event.KeyV, Mods: event.ModControl})
	if ta.Text() != "a\nb" || len(ta.lines) != 2 {
		t.Fatalf("colar: Text=%q lines=%d", ta.Text(), len(ta.lines))
	}

	// Backspace no início de linha funde com a anterior.
	ta.HandleEvent(event.KeyEvent{Key: event.KeyHome})
	ta.HandleEvent(event.KeyEvent{Key: event.KeyBackspace})
	if ta.Text() != "ab" {
		t.Fatalf("fusão de linhas: Text=%q", ta.Text())
	}
}

func TestTextAreaCliqueEBinding(t *testing.T) {
	th := newTestTheme(t)
	conteudo := state.New("um\ndois\ntrês")
	ta := NewTextArea("").BindValue(conteudo)
	ta.SetTheme(th)
	ta.Layout(image.Rect(0, 0, 300, 100))
	ta.HandleEvent(event.FocusEvent{Gained: true})

	// Clique na segunda linha posiciona lá.
	pad := th.PaddingPx()
	pos := image.Pt(pad, pad+th.LineHeight()+th.LineHeight()/2)
	ta.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: pos, Button: event.MouseButtonLeft})
	if ta.caretLine != 1 {
		t.Fatalf("clique deveria ir à linha 1; caretLine=%d", ta.caretLine)
	}
	ta.HandleEvent(event.MouseEvent{Kind: event.MouseUp, Pos: pos, Button: event.MouseButtonLeft})

	// Edição propaga ao State com quebras preservadas.
	ta.HandleEvent(event.KeyEvent{Key: event.KeyEnd})
	ta.HandleEvent(event.CharEvent{Rune: '!'})
	if conteudo.Get() != "um\ndois!\ntrês" {
		t.Fatalf("binding: State=%q", conteudo.Get())
	}
}

func TestModalAbreFechaEBackdrop(t *testing.T) {
	var opened, closed Widget
	hooks.OpenOverlay = func(w any) { opened = w.(Widget) }
	hooks.CloseOverlay = func(w any) { closed = w.(Widget) }
	defer func() { hooks.OpenOverlay, hooks.CloseOverlay = nil, nil }()

	th := newTestTheme(t)
	fechado := 0
	m := NewModal(NewVBox(NewText("Título"), NewButton("OK", nil)))
	m.OnClose(func() { fechado++ })
	m.SetTheme(th)
	m.Show()
	if opened != Widget(m) || !m.Shown() {
		t.Fatal("Show deveria abrir o modal na overlay")
	}
	if !m.SpansWindow() {
		t.Fatal("modal deveria cobrir a janela")
	}

	// Layout centraliza o painel.
	m.Layout(image.Rect(0, 0, 400, 300))
	if m.panel.Empty() || m.panel.Min.X <= 0 || m.panel.Max.X >= 400 {
		t.Fatalf("painel mal centralizado: %v", m.panel)
	}
	cx := (m.panel.Min.X + m.panel.Max.X) / 2
	// Margens iguais (±1px de arredondamento inteiro).
	if right := 400 - m.panel.Max.X; abs(right-m.panel.Min.X) > 1 {
		t.Fatalf("painel não centrado: margens %d/%d", m.panel.Min.X, right)
	}

	// Clique no painel não fecha; no backdrop, fecha (CloseOnBackdrop).
	inPanel := image.Pt(cx, (m.panel.Min.Y+m.panel.Max.Y)/2)
	m.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: inPanel, Button: event.MouseButtonLeft})
	if !m.Shown() {
		t.Fatal("clique dentro do painel não deveria fechar")
	}
	m.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: image.Pt(5, 5), Button: event.MouseButtonLeft})
	if m.Shown() || closed != Widget(m) || fechado != 1 {
		t.Fatalf("backdrop deveria fechar: shown=%v fechado=%d", m.Shown(), fechado)
	}

	// Escape fecha; com CloseOnBackdrop desligado, backdrop não fecha.
	m.CloseOnBackdrop(false)
	m.Show()
	m.HandleEvent(event.MouseEvent{Kind: event.MouseDown, Pos: image.Pt(5, 5), Button: event.MouseButtonLeft})
	if !m.Shown() {
		t.Fatal("com CloseOnBackdrop=false, backdrop não deveria fechar")
	}
	m.HandleEvent(event.KeyEvent{Key: event.KeyEscape})
	if m.Shown() || fechado != 2 {
		t.Fatalf("Escape deveria fechar; shown=%v fechado=%d", m.Shown(), fechado)
	}

	// Draw com backdrop translúcido escurece fora do painel.
	m.Show()
	m.Layout(image.Rect(0, 0, 400, 300))
	buf := image.NewRGBA(image.Rect(0, 0, 400, 300))
	render.FillRect(buf, buf.Bounds(), th.Background)
	m.Draw(buf)
	corner := buf.RGBAAt(2, 2)
	if corner == th.Background {
		t.Fatal("backdrop deveria escurecer o fundo")
	}
	if got := buf.RGBAAt(cx, m.panel.Min.Y+2); got != th.InputBorder && got != th.InputBackground {
		t.Fatalf("painel deveria ser opaco; got %v", got)
	}
}
