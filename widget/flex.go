package widget

// crossAlign define como um filho é posicionado no eixo TRANSVERSAL de um
// VBox/HBox (largura no VBox, altura no HBox).
type crossAlign int

const (
	// crossStretch (padrão) estica o filho por todo o eixo transversal.
	crossStretch crossAlign = iota
	// crossStart usa o tamanho preferido, alinhado ao início.
	crossStart
	// crossCenter usa o tamanho preferido, centralizado.
	crossCenter
	// crossEnd usa o tamanho preferido, alinhado ao fim.
	crossEnd
)

// setGrow e setCrossAlign guardam os parâmetros de layout no BaseWidget;
// growOf e crossOf os leem via interface, com defaults para widgets que não
// embutem BaseWidget.
func (b *BaseWidget) setGrow(weight int)         { b.grow = weight }
func (b *BaseWidget) setCrossAlign(a crossAlign) { b.cross = a }
func (b *BaseWidget) layoutGrow() int            { return b.grow }
func (b *BaseWidget) layoutCross() crossAlign    { return b.cross }

func growOf(w Widget) int {
	if p, ok := w.(interface{ layoutGrow() int }); ok {
		return p.layoutGrow()
	}
	return 0
}

func crossOf(w Widget) crossAlign {
	if p, ok := w.(interface{ layoutCross() crossAlign }); ok {
		return p.layoutCross()
	}
	return crossStretch
}

// Grow marca w para EXPANDIR no eixo principal do VBox/HBox: o espaço que
// sobrar após os filhos de peso zero é dividido entre os filhos com peso,
// proporcionalmente. Peso 0 volta ao tamanho preferido. Devolve o próprio w
// (com o tipo concreto preservado), para uso inline na montagem da árvore:
//
//	juigo.NewHBox(juigo.Grow(campo, 1), botao)
//
// Em widgets que não embutem BaseWidget, é um no-op.
func Grow[W Widget](w W, weight int) W {
	if p, ok := any(w).(interface{ setGrow(int) }); ok {
		p.setGrow(weight)
	}
	return w
}

// Centered posiciona w no eixo transversal do box com o tamanho preferido,
// centralizado (em vez do padrão, que é esticar). Devolve o próprio w.
func Centered[W Widget](w W) W {
	return withCross(w, crossCenter)
}

// AtStart posiciona w no eixo transversal com o tamanho preferido, alinhado
// ao início (esquerda no VBox, topo no HBox). Devolve o próprio w.
func AtStart[W Widget](w W) W {
	return withCross(w, crossStart)
}

// AtEnd posiciona w no eixo transversal com o tamanho preferido, alinhado ao
// fim (direita no VBox, base no HBox). Devolve o próprio w.
func AtEnd[W Widget](w W) W {
	return withCross(w, crossEnd)
}

func withCross[W Widget](w W, a crossAlign) W {
	if p, ok := any(w).(interface{ setCrossAlign(crossAlign) }); ok {
		p.setCrossAlign(a)
	}
	return w
}

// Spacer é um widget invisível de tamanho preferido zero e peso de expansão
// 1: em um box, empurra os irmãos ocupando o espaço livre —
// NewHBox(texto, NewSpacer(), botao) alinha o botão à direita.
type Spacer struct {
	BaseWidget
}

// NewSpacer cria um Spacer com peso 1 (ajustável com Grow).
func NewSpacer() *Spacer {
	s := &Spacer{}
	s.setGrow(1)
	return s
}

// distribute divide leftover entre os filhos proporcionalmente aos pesos,
// sem sobra nem deriva de arredondamento: o i-ésimo quinhão é a diferença
// entre alvos acumulados, garantindo soma exata.
type distributor struct {
	leftover   int
	weightSum  int
	weightSeen int
	used       int
}

func (d *distributor) next(weight int) int {
	if d.weightSum <= 0 {
		return 0
	}
	d.weightSeen += weight
	target := d.leftover * d.weightSeen / d.weightSum
	share := target - d.used
	d.used = target
	return share
}

// boxMetrics calcula, para um eixo principal de tamanho avail, o espaço
// sobrando para os filhos com peso.
func boxLeftover(children []Widget, avail, spacing int, mainPref func(Widget) int) (leftover, weightSum int) {
	fixed := 0
	for _, ch := range children {
		if g := growOf(ch); g > 0 {
			weightSum += g
		} else {
			fixed += mainPref(ch)
		}
	}
	leftover = avail - fixed - spacing*(len(children)-1)
	if leftover < 0 {
		leftover = 0
	}
	return leftover, weightSum
}
