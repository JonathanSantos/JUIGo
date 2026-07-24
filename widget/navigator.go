package widget

import (
	"image"
	"image/draw"
	"time"

	"github.com/JonathanSantos/JUIGo/anim"
	"github.com/JonathanSantos/JUIGo/event"
	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/theme"
)

// Transition é o efeito visual da troca de telas do Navigator.
type Transition int

const (
	// TransitionNone troca a tela num corte seco, sem animação.
	TransitionNone Transition = iota
	// TransitionFade funde a tela antiga na nova (crossfade).
	TransitionFade
	// TransitionSlideLeft desliza as telas para a esquerda: a nova entra
	// pela direita — a transição clássica de "avançar" (padrão do Push).
	TransitionSlideLeft
	// TransitionSlideRight desliza as telas para a direita: a nova entra
	// pela esquerda — a transição clássica de "voltar".
	TransitionSlideRight
	// TransitionSlideUp desliza as telas para cima: a nova entra por baixo,
	// como uma folha que sobe.
	TransitionSlideUp
	// TransitionSlideDown desliza as telas para baixo: a nova entra por
	// cima.
	TransitionSlideDown
)

// reversed devolve a transição oposta — a que desfaz visualmente esta. É o
// que o Pop usa por padrão: uma tela que entrou deslizando para a esquerda
// sai deslizando para a direita.
func (t Transition) reversed() Transition {
	switch t {
	case TransitionSlideLeft:
		return TransitionSlideRight
	case TransitionSlideRight:
		return TransitionSlideLeft
	case TransitionSlideUp:
		return TransitionSlideDown
	case TransitionSlideDown:
		return TransitionSlideUp
	}
	return t
}

// navEntry é um nível da pilha de navegação: a tela e a transição com que
// ela entrou (o Pop reverte essa transição por padrão).
type navEntry struct {
	w       Widget
	entered Transition
}

// Navigator gerencia uma pilha de telas com transições animadas: Push empilha
// uma tela nova, Pop volta à anterior e Replace troca a atual. Só a tela do
// topo participa do desenho, do roteamento e do foco — as demais ficam
// dormentes na pilha, com estado preservado (rolagem, campos preenchidos),
// prontas para o retorno.
//
//	nav := juigo.NewNavigator()
//	nav.Push(telaInicial(nav))
//	juigo.Run("App", 800, 600, nav)
//
//	// em um botão da tela:
//	nav.Push(telaDetalhes(nav))                  // desliza da direita
//	nav.Push(telaAjuda(nav), juigo.TransitionSlideUp) // sobe como uma folha
//	nav.Pop()                                    // reverte a transição de entrada
//
// A transição pode ser passada por chamada (argumento opcional) ou definida
// como padrão do Push com Transition; a duração vem de
// Theme.TransitionDuration (Duration a sobrepõe por Navigator). Durante a
// animação as duas telas são retratos (snapshots) e a interação é engolida;
// no App real e no uitest a animação corre pelos mesmos timers (anim.Tween),
// determinística nos testes via Advance. A primeira tela empilhada entra sem
// animação.
type Navigator struct {
	BaseWidget
	stack    []navEntry
	defTrans Transition
	dur      time.Duration
	onChange func()

	// Transição em andamento: retratos da tela que sai e da que entra,
	// compostos por progress (já com easing, animado por anim.Tween).
	animating bool
	trans     Transition
	progress  *state.State[float64]
	tween     *anim.Animation
	outBuf    *image.RGBA
	inBuf     *image.RGBA

	// child é o scratch devolvido por Children (sem alocação por chamada).
	child [1]Widget
}

// NewNavigator cria um navegador com a pilha vazia; empilhe a tela inicial
// com Push. O tema é herdado no mount.
func NewNavigator() *Navigator {
	n := &Navigator{defTrans: TransitionSlideLeft, progress: state.New(0.0)}
	n.progress.Watch(func(float64) { n.Invalidate() })
	return n
}

// Transition define a transição padrão do Push (inicial: TransitionSlideLeft).
// Use TransitionNone para um navegador sem animações. Encadeável.
func (n *Navigator) Transition(t Transition) *Navigator {
	n.defTrans = t
	return n
}

// Duration sobrepõe a duração das transições deste navegador; zero volta ao
// padrão do tema (Theme.TransitionDuration). Encadeável.
func (n *Navigator) Duration(d time.Duration) *Navigator {
	n.dur = d
	return n
}

// OnChange define o callback chamado a cada navegação (Push, Pop, Replace),
// no momento da troca — a animação segue depois dele. Encadeável.
func (n *Navigator) OnChange(fn func()) *Navigator {
	n.onChange = fn
	return n
}

// Push empilha w como tela atual. Sem argumento, anima com a transição
// padrão do navegador; com um, usa a transição dada só nesta navegação. A
// primeira tela da pilha entra sem animação.
func (n *Navigator) Push(w Widget, t ...Transition) {
	if w == nil {
		return
	}
	n.finishNow()
	var old Widget
	if top := n.top(); top != nil {
		old = top
	}
	tr := n.defTrans
	if len(t) > 0 {
		tr = t[0]
	}
	if old == nil {
		tr = TransitionNone
	}
	n.stack = append(n.stack, navEntry{w: w, entered: tr})
	n.navigate(old, w, tr)
}

// Pop desempilha a tela atual e volta à anterior, revertendo a transição com
// que a tela entrou (ou usando a transição dada). Com uma tela só (ou
// nenhuma), não faz nada — a raiz não sai da pilha.
func (n *Navigator) Pop(t ...Transition) {
	if len(n.stack) < 2 {
		return
	}
	n.finishNow()
	last := len(n.stack) - 1
	old := n.stack[last]
	n.stack[last] = navEntry{}
	n.stack = n.stack[:last]
	tr := old.entered.reversed()
	if len(t) > 0 {
		tr = t[0]
	}
	n.navigate(old.w, n.top(), tr)
}

// PopToRoot desempilha tudo até a primeira tela, numa única transição —
// revertendo a da tela atual (ou usando a dada).
func (n *Navigator) PopToRoot(t ...Transition) {
	if len(n.stack) < 2 {
		return
	}
	n.finishNow()
	old := n.stack[len(n.stack)-1]
	for i := 1; i < len(n.stack); i++ {
		n.stack[i] = navEntry{}
	}
	n.stack = n.stack[:1]
	tr := old.entered.reversed()
	if len(t) > 0 {
		tr = t[0]
	}
	n.navigate(old.w, n.top(), tr)
}

// Replace troca a tela atual por w sem crescer a pilha, com TransitionFade
// por padrão. O nível mantém a transição com que entrou — um Pop posterior
// segue revertendo a navegação que o criou.
func (n *Navigator) Replace(w Widget, t ...Transition) {
	if w == nil {
		return
	}
	if len(n.stack) == 0 {
		n.Push(w, t...)
		return
	}
	n.finishNow()
	last := len(n.stack) - 1
	old := n.stack[last]
	tr := TransitionFade
	if len(t) > 0 {
		tr = t[0]
	}
	n.stack[last] = navEntry{w: w, entered: old.entered}
	n.navigate(old.w, w, tr)
}

// Depth devolve a profundidade da pilha (0 em navegadores vazios).
func (n *Navigator) Depth() int {
	return len(n.stack)
}

// CanPop informa se há tela anterior para voltar — a condição de exibir um
// botão "Voltar".
func (n *Navigator) CanPop() bool {
	return len(n.stack) > 1
}

// Top devolve a tela atual (nil com a pilha vazia).
func (n *Navigator) Top() Widget {
	return n.top()
}

// Animating informa se uma transição está em andamento.
func (n *Navigator) Animating() bool {
	return n.animating
}

// top devolve a tela do topo da pilha, ou nil.
func (n *Navigator) top() Widget {
	if len(n.stack) == 0 {
		return nil
	}
	return n.stack[len(n.stack)-1].w
}

// navigate inicia a transição de old (pode ser nil) para a tela que acabou
// de virar o topo. Sem tema, sem geometria, sem duração ou com
// TransitionNone, a troca é um corte seco.
func (n *Navigator) navigate(old, novo Widget, tr Transition) {
	n.blurFocusIn(old)
	if n.onChange != nil {
		n.onChange()
	}
	b := n.Bounds()
	if n.theme != nil {
		// Monta e posiciona a tela nova na hora (como OpenOverlay): ela nasce
		// com geometria válida, pronta para eventos antes do primeiro frame.
		n.mountIncoming(novo, b)
	}
	d := n.dur
	if d == 0 && n.theme != nil {
		d = n.theme.TransitionDuration
	}
	if old == nil || tr == TransitionNone || d <= 0 || n.theme == nil || b.Empty() {
		n.Invalidate()
		return
	}
	n.snapshot(old, &n.outBuf)
	n.snapshot(novo, &n.inBuf)
	n.trans = tr
	n.animating = true
	n.progress.Set(0)
	tw := anim.Tween(n.progress, 1, d, nil)
	if !tw.Running() {
		// Sem aplicação/harness o Tween salta direto ao alvo (e OnDone não
		// dispara): a transição termina aqui mesmo.
		n.finishTransition()
		return
	}
	tw.OnDone = n.finishTransition
	n.tween = tw
	n.Invalidate()
}

// mountIncoming injeta o tema na tela nova (com identidade de sessão, quando
// montado) e a posiciona nos bounds atuais.
func (n *Navigator) mountIncoming(w Widget, b image.Rectangle) {
	if n.session != nil {
		n.session.mount(w, n.theme)
	} else {
		Mount(w, n.theme)
	}
	w.Layout(b)
}

// blurFocusIn tira o foco do teclado da tela que sai, se estiver nela — a
// tela deixa a árvore e não pode reter o foco.
func (n *Navigator) blurFocusIn(old Widget) {
	if n.session == nil || old == nil {
		return
	}
	if f := n.session.Focused(); f != nil && Contains(old, f) {
		n.session.Focus(nil)
	}
}

// snapshot desenha w num retrato reutilizável alinhado aos bounds do
// navegador (fundo do tema por baixo, como o frame real). Aloca só quando o
// tamanho muda — nunca por quadro.
func (n *Navigator) snapshot(w Widget, buf **image.RGBA) {
	b := n.Bounds()
	if *buf == nil || (*buf).Rect != b {
		*buf = image.NewRGBA(b)
	}
	render.FillRect(*buf, b, n.theme.Background)
	w.Draw(*buf)
}

// finishTransition conclui a transição em andamento e volta ao desenho vivo
// da tela do topo.
func (n *Navigator) finishTransition() {
	if !n.animating {
		return
	}
	n.animating = false
	n.tween = nil
	n.Invalidate()
}

// finishNow interrompe uma transição em andamento saltando ao estado final —
// navegação nova no meio da animação e resize caem aqui.
func (n *Navigator) finishNow() {
	if n.tween != nil {
		n.tween.Stop()
		n.tween = nil
	}
	if n.animating {
		n.animating = false
		n.Invalidate()
	}
}

// Children devolve apenas a tela do topo; durante uma transição não devolve
// nada — as telas são retratos e não recebem eventos.
func (n *Navigator) Children() []Widget {
	if n.animating {
		return nil
	}
	top := n.top()
	if top == nil {
		return nil
	}
	n.child[0] = top
	return n.child[:]
}

// SetTheme define um tema explícito e o propaga à tela atual, como os
// containers. Telas dormentes na pilha recebem o tema quando voltam ao topo.
func (n *Navigator) SetTheme(th *theme.Theme) {
	n.BaseWidget.SetTheme(th)
	Mount(n, th)
}

// Layout posiciona a tela do topo nos bounds do navegador. Um resize no meio
// de uma transição a conclui na hora — os retratos ficaram do tamanho antigo.
func (n *Navigator) Layout(bounds image.Rectangle) {
	changed := bounds != n.Bounds()
	n.BaseWidget.Layout(bounds)
	if changed && n.animating {
		n.finishNow()
	}
	if top := n.top(); top != nil {
		top.Layout(bounds)
	}
}

// PreferredSize devolve o máximo entre TODAS as telas da pilha, para o
// tamanho não pular ao navegar (como as páginas do Tabs).
func (n *Navigator) PreferredSize() image.Point {
	var pref image.Point
	for _, e := range n.stack {
		p := e.w.PreferredSize()
		pref.X = max(pref.X, p.X)
		pref.Y = max(pref.Y, p.Y)
	}
	return pref
}

// HandleEvent engole mouse e rolagem durante uma transição — as telas são
// retratos e as posições não correspondem a widgets vivos.
func (n *Navigator) HandleEvent(ev event.Event) bool {
	if !n.animating {
		return false
	}
	switch ev.(type) {
	case event.MouseEvent, event.ScrollEvent:
		return true
	}
	return false
}

// Draw desenha a tela do topo — ou, durante uma transição, a composição dos
// retratos da tela que sai e da que entra.
func (n *Navigator) Draw(dst *image.RGBA) {
	if n.animating {
		n.drawTransition(dst)
		n.drawDisabledOverlay(dst)
		return
	}
	if top := n.top(); top != nil && top.Bounds().Overlaps(dst.Bounds()) {
		top.Draw(dst)
	}
	n.drawDisabledOverlay(dst)
}

// drawTransition compõe os retratos conforme a transição e o progresso (já
// com easing). Nos slides as duas telas se movem juntas e cobrem a área
// inteira em qualquer progresso; no fade, mistura por pixel (CrossFade).
func (n *Navigator) drawTransition(dst *image.RGBA) {
	b := n.Bounds()
	p := n.progress.Get()
	switch n.trans {
	case TransitionFade:
		render.CrossFade(dst, n.outBuf, n.inBuf, b, p)
	case TransitionSlideLeft:
		dx := int(p*float64(b.Dx()) + 0.5)
		drawShifted(dst, n.outBuf, b, image.Pt(-dx, 0))
		drawShifted(dst, n.inBuf, b, image.Pt(b.Dx()-dx, 0))
	case TransitionSlideRight:
		dx := int(p*float64(b.Dx()) + 0.5)
		drawShifted(dst, n.outBuf, b, image.Pt(dx, 0))
		drawShifted(dst, n.inBuf, b, image.Pt(dx-b.Dx(), 0))
	case TransitionSlideUp:
		dy := int(p*float64(b.Dy()) + 0.5)
		drawShifted(dst, n.outBuf, b, image.Pt(0, -dy))
		drawShifted(dst, n.inBuf, b, image.Pt(0, b.Dy()-dy))
	case TransitionSlideDown:
		dy := int(p*float64(b.Dy()) + 0.5)
		drawShifted(dst, n.outBuf, b, image.Pt(0, dy))
		drawShifted(dst, n.inBuf, b, image.Pt(0, dy-b.Dy()))
	}
}

// drawShifted desenha src (um retrato alinhado a b) deslocado por off,
// recortado a b. draw.Src copia direto — retratos são opacos. Não aloca.
func drawShifted(dst *image.RGBA, src *image.RGBA, b image.Rectangle, off image.Point) {
	r := b.Add(off).Intersect(b)
	if r.Empty() {
		return
	}
	draw.Draw(dst, r, src, r.Min.Sub(off), draw.Src)
}
