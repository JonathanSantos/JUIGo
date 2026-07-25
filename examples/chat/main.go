// Chat — a demo bandeira do design system "papel e tinta": uma conversa
// com a Tinta, a assistente de mentira do JUIGo. Junta num app só a barra
// de menus com atalhos (e a paleta Ctrl/Cmd+K de brinde), o SplitPane com
// a lista de conversas em pílulas, balões de Card com Label (parágrafos
// que quebram sozinhos), resposta "digitando" pelos timers da aplicação e
// a serif Lora nos títulos. Nenhuma IA de verdade foi ferida: as respostas
// são cartas da casa.
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
)

func main() {
	app, err := juigo.New("Chat — papel e tinta", 900, 560)
	if err != nil {
		log.Fatal(err)
	}
	if th, err := juigo.ClaudeTheme(); err == nil {
		if err := app.SetTheme(th); err != nil {
			log.Fatal(err)
		}
	}
	v := nova()
	app.SetRoot(v.Raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// mensagem é uma fala da conversa; minha=true é o lado direito (Card).
type mensagem struct {
	minha bool
	texto string
}

// conversa é um item da sidebar com o histórico próprio.
type conversa struct {
	titulo string
	msgs   []mensagem
}

// respostas são as cartas da casa — a Tinta responde na ordem, circular.
var respostas = []string{
	"Boa pergunta. No papel e tinta, a resposta certa costuma ser a mais quente: neutros de papel, uma única terracota de ação e títulos com serifa.",
	"Pensa assim: se tudo na tela é um cartão, nada é. Agrupe o que conversa entre si e deixe o papel respirar no resto.",
	"Um título por tela, uma ação primária por bloco — o resto é hierarquia. É disso que o DESIGN.md vive.",
	"Anotei. Agora me conta: isso precisa mesmo de um modal, ou um toast resolve com menos cerimônia?",
}

// vista é a interface do chat: sidebar de conversas + painel de mensagens.
type vista struct {
	Raiz juigo.Widget

	conversas []*conversa
	atual     int
	respIdx   int

	titulo   *juigo.State[string]
	selecao  *juigo.State[int]
	lado     *juigo.List[*juigo.Text]
	feed     *juigo.VBox
	rolagem  *juigo.Scroll
	campo    *juigo.Input
	digitando bool
	// fluxo é o texto da resposta em curso ("digitando"), ligado ao Label
	// da última mensagem.
	fluxo  *juigo.State[string]
	cancel func()
}

// nova monta a vista com uma conversa inicial.
func nova() *vista {
	v := &vista{
		titulo:  juigo.NewState(""),
		selecao: juigo.NewState(0),
		fluxo:   juigo.NewState(""),
	}
	v.conversas = []*conversa{{titulo: "Primeiras ideias"}}

	v.lado = juigo.NewList(len(v.conversas),
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, i int) {
			if i >= 0 && i < len(v.conversas) {
				t.SetText(v.conversas[i].titulo)
			}
		},
	).BindSelected(v.selecao)
	v.selecao.Watch(func(i int) { v.trocar(i) })

	v.feed = juigo.NewVBox().Pad(16).Gap(10)
	v.rolagem = juigo.NewScroll(v.feed)
	v.campo = juigo.NewInput("Escreva para a Tinta…").OnSubmit(v.enviar)

	barra := juigo.NewMenuBar().
		Menu("Conversa",
			juigo.Command{Title: "Nova conversa", Key: juigo.LetterKey('n'),
				Mods: juigo.ModControl, Action: v.novaConversa},
			juigo.MenuSeparator(),
			juigo.Command{Title: "Limpar conversa", Action: v.limpar},
		).
		Menu("Ajuda",
			juigo.Command{Title: "Sobre o chat", Action: func() {
				quick.Alert("Feito com JUIGo, papel e tinta. As respostas são cartas da casa.")
			}},
		)

	sidebar := juigo.NewVBox(
		juigo.NewText("Conversas").Subtitle(),
		juigo.Grow(juigo.NewScroll(v.lado), 1),
	).Pad(10).Gap(8)

	painel := juigo.NewVBox(
		juigo.NewText("").BindText(v.titulo).Title(),
		juigo.NewDivider(),
		juigo.Grow(v.rolagem, 1),
		juigo.NewHBox(
			juigo.Grow(v.campo, 1),
			juigo.NewButton("Enviar", v.enviar),
		).Gap(8),
	).Pad(16).Gap(8)

	v.Raiz = juigo.NewVBox(
		barra,
		juigo.Grow(juigo.NewSplitPane(sidebar, painel).Ratio(0.26).Min(120), 1),
	)
	v.trocar(0)
	return v
}

// atualConversa devolve a conversa selecionada.
func (v *vista) atualConversa() *conversa {
	return v.conversas[v.atual]
}

// trocar muda a conversa exibida e reprojeta o histórico.
func (v *vista) trocar(i int) {
	if i < 0 || i >= len(v.conversas) {
		return
	}
	v.pararFluxo()
	v.atual = i
	v.titulo.Set(v.conversas[i].titulo)
	v.reprojeta()
}

// novaConversa abre uma conversa vazia e a seleciona.
func (v *vista) novaConversa() {
	v.pararFluxo()
	v.conversas = append(v.conversas, &conversa{
		titulo: fmt.Sprintf("Conversa %d", len(v.conversas)+1),
	})
	v.lado.SetCount(len(v.conversas))
	v.lado.Refresh()
	v.selecao.Set(len(v.conversas) - 1)
}

// limpar zera o histórico da conversa atual.
func (v *vista) limpar() {
	v.pararFluxo()
	v.atualConversa().msgs = nil
	v.reprojeta()
}

// enviar publica a mensagem digitada e inicia a resposta da Tinta.
func (v *vista) enviar() {
	texto := strings.TrimSpace(v.campo.Text())
	if texto == "" || v.digitando {
		return
	}
	v.campo.SetText("")
	c := v.atualConversa()
	c.msgs = append(c.msgs, mensagem{minha: true, texto: texto})
	v.responder(respostas[v.respIdx%len(respostas)])
	v.respIdx++
}

// responder inicia o fluxo "digitando": a resposta aparece palavra a
// palavra pelos timers da aplicação (determinístico no uitest via Advance).
func (v *vista) responder(texto string) {
	c := v.atualConversa()
	c.msgs = append(c.msgs, mensagem{texto: ""})
	v.digitando = true
	v.fluxo.Set("")
	v.reprojeta()

	palavras := strings.Fields(texto)
	i := 0
	v.cancel = juigo.Every(45*time.Millisecond, func() { // uma palavra por tique
		if i < len(palavras) {
			atual := v.fluxo.Get()
			if atual != "" {
				atual += " "
			}
			v.fluxo.Set(atual + palavras[i])
			i++
			v.rolagem.ScrollTo(1 << 30)
			return
		}
		v.pararFluxo()
	})
}

// pararFluxo conclui a resposta em curso, gravando o texto no histórico.
func (v *vista) pararFluxo() {
	if !v.digitando {
		return
	}
	v.digitando = false
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	c := v.atualConversa()
	if n := len(c.msgs); n > 0 && !c.msgs[n-1].minha {
		c.msgs[n-1].texto = v.fluxo.Get()
	}
	v.reprojeta()
}

// reprojeta reconstrói o feed a partir do histórico (padrão do TodoMVC:
// reconstruir é a resposta) — a última mensagem da Tinta, em curso, fica
// ligada ao State do fluxo.
func (v *vista) reprojeta() {
	v.feed.Clear()
	c := v.atualConversa()
	for i, m := range c.msgs {
		if m.minha {
			balao := juigo.NewCard(juigo.NewLabel(m.texto).MaxWidth(320)).Pad(10)
			v.feed.Add(juigo.AtEnd(balao))
			continue
		}
		nome := juigo.NewText("Tinta").Caption()
		var corpo *juigo.Label
		if v.digitando && i == len(c.msgs)-1 {
			corpo = juigo.NewLabel("").BindText(v.fluxo)
		} else {
			corpo = juigo.NewLabel(m.texto)
		}
		corpo.MaxWidth(420)
		v.feed.Add(juigo.NewVBox(nome, corpo).Gap(2))
	}
	v.rolagem.ScrollTo(1 << 30)
}

