package juigo

import (
	"fmt"
	"image"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/internal/hooks"
	"github.com/JonathanSantos/JUIGo/theme"
	"github.com/JonathanSantos/JUIGo/widget"
)

func init() {
	// GLFW e OpenGL exigem que tudo aconteça na main thread do SO.
	runtime.LockOSThread()
}

// App é a aplicação JUIGo: dona das JANELAS (uma ou várias — ver NewWindow),
// dos timers e do loop de eventos. Cada janela tem o próprio buffer, tema e
// widget.Session; toda a lógica de INTERAÇÃO (roteamento, foco, captura,
// hover, overlay, tooltip) vive na Session — a janela apenas traduz os
// eventos do GLFW para ela, o que permite testar o comportamento real
// headless com juigo/uitest.
//
// Os métodos de conveniência do App (SetRoot, SetTheme, Theme, Session)
// operam na JANELA PRINCIPAL — aplicações de uma janela não precisam saber
// que Window existe. O loop termina quando todas as janelas fecham.
type App struct {
	windows []*Window
	// main é a janela principal, criada por New; os métodos de conveniência
	// do App delegam a ela.
	main *Window
	bus  *event.Bus
	// corrente é a janela cujo callback está em execução: é para a sessão
	// dela que os hooks globais (overlay, foco, toast, arrasto) roteiam.
	corrente *Window
	// fatalErr registra uma falha ocorrida dentro de um callback (que não
	// pode devolver erro); Run a detecta e encerra.
	fatalErr error
	// timers são os agendamentos pendentes (hooks.Schedule): o loop usa
	// WaitEventsTimeout para acordar no vencimento mais próximo.
	timers   []appTimer
	timerSeq int
	// postMu protege posted — a ÚNICA estrutura do JUIGo tocada por outras
	// goroutines (ver Post).
	postMu sync.Mutex
	posted []func()
}

// appTimer é um agendamento pendente de hooks.Schedule.
type appTimer struct {
	id int
	at time.Time
	fn func()
}

// New cria a aplicação com a janela principal (título e tamanho dados) e
// inicializa o contexto OpenGL. Deve ser chamada na main thread (o pacote
// já trava a goroutine corrente na thread do SO via runtime.LockOSThread).
func New(title string, width, height int) (*App, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("juigo: falha ao inicializar GLFW: %w", err)
	}
	th, err := theme.Default()
	if err != nil {
		glfw.Terminate()
		return nil, err
	}
	a := &App{bus: event.NewBus()}
	w, err := a.createWindow(title, width, height, th)
	if err != nil {
		glfw.Terminate()
		return nil, err
	}
	a.main = w
	a.installHooks()
	return a, nil
}

// Run cria a aplicação com New, define root como raiz da janela principal e
// executa o loop de eventos até todas as janelas fecharem. É o caminho
// curto para a maioria das aplicações; use New quando precisar do *App
// (tema, NewWindow, Invalidate, Bus).
func Run(title string, width, height int, root widget.Widget) error {
	app, err := New(title, width, height)
	if err != nil {
		return err
	}
	app.SetRoot(root)
	return app.Run()
}

// NewWindow abre uma janela ADICIONAL com o título e tamanho dados: tema
// próprio (um clone do tema da janela principal, levado à escala do monitor
// onde ela abrir) e Session própria — foco, overlay e toast independentes.
// Defina o conteúdo com SetRoot; feche com Close (ou o botão do sistema).
// Deve ser chamada na main thread — de um handler de evento ou de um
// App.Post. A aplicação termina quando TODAS as janelas fecham.
func (a *App) NewWindow(title string, width, height int) (*Window, error) {
	base := a.main
	if base == nil || base.closed {
		if len(a.windows) == 0 {
			return nil, fmt.Errorf("juigo: aplicação sem janelas")
		}
		base = a.windows[0]
	}
	th, err := base.theme.Clone()
	if err != nil {
		return nil, err
	}
	w, err := a.createWindow(title, width, height, th)
	if err != nil {
		return nil, err
	}
	w.render() // primeiro frame imediato: a janela abre pintada
	return w, nil
}

// dispatch executa fn com w marcada como a janela em interação — os hooks
// globais roteiam para a sessão dela durante a entrega do evento.
func (a *App) dispatch(w *Window, fn func()) {
	prev := a.corrente
	a.corrente = w
	fn()
	a.corrente = prev
}

// activeSession devolve a sessão da janela em interação; fora de um
// callback, a da janela focada; senão, a da primeira janela aberta.
func (a *App) activeSession() *widget.Session {
	if a.corrente != nil {
		return a.corrente.session
	}
	for _, w := range a.windows {
		if w.window.GetAttrib(glfw.Focused) == glfw.True {
			return w.session
		}
	}
	if len(a.windows) > 0 {
		return a.windows[0].session
	}
	return nil
}

// installHooks liga os ganchos globais do processo à aplicação. Setters de
// widgets e State.Set redesenham por aqui (o dano de widgets já montados
// vai DIRETO à sessão dona; este é o fallback repartido); o Input
// copia/cola através da área de transferência do sistema; overlay, foco
// programático, toast e arrasto vão à janela em interação; timers acordam
// o loop.
func (a *App) installHooks() {
	hooks.Repaint = func() {
		for _, w := range a.windows {
			w.session.InvalidateAll()
			w.dirty = true
		}
		glfw.PostEmptyEvent()
	}
	hooks.Damage = func(r image.Rectangle) {
		// Fallback de widgets ainda sem sessão anexada (antes do primeiro
		// mount): reparte entre as janelas — repintura a mais é segura.
		for _, w := range a.windows {
			w.session.AddDamage(r)
		}
	}
	hooks.Frame = func() {
		// Frame sem dano: janelas sem dano acumulado pulam o render.
		for _, w := range a.windows {
			w.dirty = true
		}
	}
	hooks.ClipboardRead = func() string {
		if len(a.windows) == 0 {
			return ""
		}
		return a.windows[0].window.GetClipboardString()
	}
	hooks.ClipboardWrite = func(s string) {
		if len(a.windows) == 0 {
			return
		}
		a.windows[0].window.SetClipboardString(s)
	}
	hooks.Schedule = a.schedule
	hooks.OpenOverlay = func(v any) {
		if w, ok := v.(widget.Widget); ok {
			if s := a.activeSession(); s != nil {
				s.OpenOverlay(w)
			}
		}
	}
	hooks.CloseOverlay = func(v any) {
		w, ok := v.(widget.Widget)
		if !ok {
			return
		}
		// O fechamento vale para a janela que estiver exibindo a camada.
		for _, j := range a.windows {
			j.session.CloseOverlayIf(w)
		}
	}
	hooks.Focus = func(v any) {
		w, _ := v.(widget.Widget)
		if s := a.activeSession(); s != nil {
			s.Focus(w)
		}
	}
	hooks.Toast = func(text string, d time.Duration) {
		if s := a.activeSession(); s != nil {
			s.ShowToast(text, d)
		}
	}
	hooks.StartDrag = func(payload any, label string) {
		if s := a.activeSession(); s != nil {
			s.StartDrag(payload, label)
		}
	}
}

// schedule agenda fn para executar na main thread após d e devolve o
// cancelamento. Implementação de hooks.Schedule.
func (a *App) schedule(d time.Duration, fn func()) func() {
	a.timerSeq++
	id := a.timerSeq
	a.timers = append(a.timers, appTimer{id: id, at: time.Now().Add(d), fn: fn})
	glfw.PostEmptyEvent() // acorda o loop para recalcular o timeout
	return func() {
		for i := range a.timers {
			if a.timers[i].id == id {
				a.timers = append(a.timers[:i], a.timers[i+1:]...)
				return
			}
		}
	}
}

// nextTimerWait devolve quanto falta para o agendamento mais próximo.
func (a *App) nextTimerWait() (time.Duration, bool) {
	if len(a.timers) == 0 {
		return 0, false
	}
	next := a.timers[0].at
	for _, t := range a.timers[1:] {
		if t.at.Before(next) {
			next = t.at
		}
	}
	return time.Until(next), true
}

// runDueTimers executa os agendamentos vencidos. Os callbacks rodam após a
// remoção da lista e podem agendar novos timers com segurança.
func (a *App) runDueTimers() {
	now := time.Now()
	var due []func()
	kept := a.timers[:0]
	for _, t := range a.timers {
		if t.at.After(now) {
			kept = append(kept, t)
		} else {
			due = append(due, t.fn)
		}
	}
	a.timers = kept
	for _, fn := range due {
		fn()
	}
}

// pixelRatio calcula a razão framebuffer/janela (pixels físicos por unidade
// lógica de janela), usada para converter coordenadas de mouse.
func pixelRatio(window *glfw.Window, fbWidth int) float64 {
	winW, _ := window.GetSize()
	if winW <= 0 || fbWidth <= 0 {
		return 1
	}
	return float64(fbWidth) / float64(winW)
}

// Bus devolve o barramento de eventos da aplicação (Publish síncrono), para
// comunicação entre partes do código do usuário.
func (a *App) Bus() *event.Bus {
	return a.bus
}

// primary devolve a janela principal (ou a primeira aberta, se a principal
// fechou); nil sem janelas.
func (a *App) primary() *Window {
	if a.main != nil && !a.main.closed {
		return a.main
	}
	if len(a.windows) > 0 {
		return a.windows[0]
	}
	return nil
}

// Window devolve a janela principal da aplicação.
func (a *App) Window() *Window {
	return a.primary()
}

// Theme devolve o tema da janela principal, usado na construção dos widgets.
func (a *App) Theme() *theme.Theme {
	if w := a.primary(); w != nil {
		return w.theme
	}
	return nil
}

// Session devolve o núcleo de interação da janela principal.
func (a *App) Session() *widget.Session {
	if w := a.primary(); w != nil {
		return w.session
	}
	return nil
}

// SetTheme troca o tema da janela principal em runtime (ex.: claro ↔
// escuro) — ver Window.SetTheme; para as demais janelas, use o método delas.
func (a *App) SetTheme(th *theme.Theme) error {
	if w := a.primary(); w != nil {
		return w.SetTheme(th)
	}
	return nil
}

// AddCommand registra (ou substitui, pelo título) um comando global da
// janela principal: atalho, item de paleta e de MenuBar (ver
// widget.Command). Para outras janelas, use Window.AddCommand.
func (a *App) AddCommand(c widget.Command) {
	if w := a.primary(); w != nil {
		w.AddCommand(c)
	}
}

// ShowCommandPalette abre a paleta de comandos (Ctrl/Cmd+K) da janela
// principal.
func (a *App) ShowCommandPalette() {
	if w := a.primary(); w != nil {
		w.session.ShowCommandPalette()
	}
}

// SetRoot define o widget raiz da janela principal e agenda um redesenho.
func (a *App) SetRoot(w widget.Widget) {
	if j := a.primary(); j != nil {
		j.SetRoot(w)
	}
}

// Invalidate marca TODAS as janelas como sujas e acorda o loop, forçando
// renderizações completas. É a válvula de escape para mudanças feitas por
// fora dos setters/States (mutação direta de campos públicos); para uma
// janela só, use Window.Invalidate.
func (a *App) Invalidate() {
	for _, w := range a.windows {
		w.session.InvalidateAll()
		w.dirty = true
	}
	glfw.PostEmptyEvent()
}

// Post agenda fn para executar na MAIN THREAD, na próxima volta do loop de
// eventos. É o ÚNICO método do JUIGo seguro para chamar de outras goroutines
// — a ponte para trabalho assíncrono (rede, disco): faça o trabalho pesado
// em uma goroutine e entregue o resultado à interface via Post; DENTRO de fn,
// State.Set e os setters de widgets são seguros como sempre.
//
//	btn.SetLoading(true)
//	go func() {
//	    resultado := buscar()
//	    app.Post(func() {
//	        btn.SetLoading(false)
//	        estado.Set(resultado)
//	    })
//	}()
func (a *App) Post(fn func()) {
	if fn == nil {
		return
	}
	a.postMu.Lock()
	a.posted = append(a.posted, fn)
	a.postMu.Unlock()
	glfw.PostEmptyEvent() // thread-safe: acorda o loop
}

// runPosted executa, na main thread, os callbacks entregues via Post.
func (a *App) runPosted() {
	a.postMu.Lock()
	batch := a.posted
	a.posted = nil
	a.postMu.Unlock()
	for _, fn := range batch {
		fn()
	}
}

// closePending destrói as janelas marcadas para fechar (botão do sistema ou
// Close), chamando o OnClose de cada uma.
func (a *App) closePending() {
	kept := a.windows[:0]
	for _, w := range a.windows {
		if w.window.ShouldClose() {
			w.destroy()
			continue
		}
		kept = append(kept, w)
	}
	a.windows = kept
}

// renderAll renderiza as janelas sujas (cada uma só repinta e envia à GPU a
// própria região danificada).
func (a *App) renderAll() {
	for _, w := range a.windows {
		w.render()
	}
}

// Run executa o loop de eventos até TODAS as janelas fecharem e então
// libera os recursos da aplicação. Usa glfw.WaitEvents (bloqueante): só há
// trabalho de CPU quando chegam eventos, e só há renderização nas janelas
// sujas. Falhas ocorridas dentro de callbacks (ex.: reescalar o tema ao
// trocar de monitor) encerram o loop e são devolvidas aqui.
func (a *App) Run() error {
	a.renderAll()
	for len(a.windows) > 0 {
		// Com timers pendentes (piscada do cursor, tooltip), o loop acorda
		// no vencimento mais próximo; sem timers, bloqueia como sempre.
		if wait, ok := a.nextTimerWait(); ok {
			if wait > 0 {
				glfw.WaitEventsTimeout(wait.Seconds())
			}
		} else {
			glfw.WaitEvents()
		}
		a.runDueTimers()
		a.runPosted()
		if a.fatalErr != nil {
			a.destroy()
			return a.fatalErr
		}
		a.closePending()
		a.renderAll()
	}
	a.destroy()
	return nil
}

// destroy libera os ganchos globais, as janelas restantes e o GLFW.
func (a *App) destroy() {
	hooks.Repaint = nil
	hooks.Damage = nil
	hooks.Frame = nil
	hooks.ClipboardRead = nil
	hooks.ClipboardWrite = nil
	hooks.Schedule = nil
	hooks.OpenOverlay = nil
	hooks.CloseOverlay = nil
	hooks.Focus = nil
	hooks.Toast = nil
	hooks.StartDrag = nil
	for _, w := range a.windows {
		w.destroy()
	}
	a.windows = nil
	glfw.Terminate()
}

// mapMods converte os modificadores do GLFW para o tipo do JUIGo.
func mapMods(m glfw.ModifierKey) event.Modifiers {
	var mods event.Modifiers
	if m&glfw.ModShift != 0 {
		mods |= event.ModShift
	}
	if m&glfw.ModControl != 0 {
		mods |= event.ModControl
	}
	if m&glfw.ModAlt != 0 {
		mods |= event.ModAlt
	}
	if m&glfw.ModSuper != 0 {
		mods |= event.ModSuper
	}
	return mods
}

// mapKey converte teclas do GLFW para as teclas reconhecidas pelo JUIGo.
func mapKey(key glfw.Key) event.Key {
	switch key {
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return event.KeyEnter
	case glfw.KeySpace:
		return event.KeySpace
	case glfw.KeyTab:
		return event.KeyTab
	case glfw.KeyBackspace:
		return event.KeyBackspace
	case glfw.KeyDelete:
		return event.KeyDelete
	case glfw.KeyLeft:
		return event.KeyLeft
	case glfw.KeyRight:
		return event.KeyRight
	case glfw.KeyHome:
		return event.KeyHome
	case glfw.KeyEnd:
		return event.KeyEnd
	case glfw.KeyUp:
		return event.KeyUp
	case glfw.KeyDown:
		return event.KeyDown
	case glfw.KeyEscape:
		return event.KeyEscape
	default:
		// Letras e dígitos por faixa (contíguos no GLFW): a base dos
		// atalhos globais (widget.Command).
		if key >= glfw.KeyA && key <= glfw.KeyZ {
			return event.LetterKey(rune('A' + int(key-glfw.KeyA)))
		}
		if key >= glfw.Key0 && key <= glfw.Key9 {
			return event.LetterKey(rune('0' + int(key-glfw.Key0)))
		}
		return event.KeyUnknown
	}
}

// mapMouseButton converte o botão do GLFW para o tipo do JUIGo.
func mapMouseButton(b glfw.MouseButton) event.MouseButton {
	switch b {
	case glfw.MouseButtonRight:
		return event.MouseButtonRight
	case glfw.MouseButtonMiddle:
		return event.MouseButtonMiddle
	default:
		return event.MouseButtonLeft
	}
}
