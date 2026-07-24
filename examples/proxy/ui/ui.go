// Package ui é a camada de interface do mini-proxy: uma Vista que projeta o
// domínio (proxy.Store/Mocks) em widgets e traduz interações em chamadas de
// domínio. O proxy corre em goroutines do servidor HTTP; a captura chega à
// UI pela ponte Store.OnChange → App.Post (salta para a main thread), e daí
// o fluxo é o mesmo dos outros exemplos: evento → aplica → reprojeta.
//
// A seleção da tabela é preservada por IDENTIDADE (ID da troca) através do
// filtro. Os visores de requisição/resposta são CodeEditor em modo
// ReadOnly (juigo.CodeEditor.ReadOnly).
package ui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/examples/proxy/proxy"
	"github.com/JonathanSantos/JUIGo/quick"
	"github.com/JonathanSantos/JUIGo/syntax"
)

// Vista é o view-model do proxy.
type Vista struct {
	prox *proxy.Proxy
	post func(func()) // ponte para a main thread (App.Post)

	metodo   *juigo.State[string]
	filtro   *juigo.State[string]
	sel      *juigo.State[int]
	selID    int
	visiveis []*proxy.Exchange

	tabela   *juigo.Table
	abas     *juigo.Tabs
	reqView  *juigo.CodeEditor
	respView *juigo.CodeEditor
	mocks    *juigo.VBox
	status   *juigo.State[string]

	// Raiz é a árvore pronta para App.SetRoot.
	Raiz juigo.Widget
}

// New monta a Vista sobre o proxy dado; post é o App.Post (a captura chega
// de outras goroutines e precisa saltar para a main thread).
func New(prox *proxy.Proxy, post func(func())) *Vista {
	v := &Vista{
		prox:   prox,
		post:   post,
		metodo: juigo.NewState("Todos"),
		filtro: juigo.NewState(""),
		sel:    juigo.NewState(-1),
		selID:  -1,
		status: juigo.NewState("parado"),
	}

	// Mestre: filtros + tabela de trocas.
	filtroMetodo := juigo.NewDropdown("Todos", "GET", "POST", "PUT", "DELETE", "PATCH", "CONNECT").
		BindValue(v.metodo)
	v.metodo.Watch(func(string) { v.reprojeta() })
	filtroTexto := juigo.NewInput("filtrar por URL…").BindValue(v.filtro)
	v.filtro.Watch(func(string) { v.reprojeta() })
	limpar := juigo.NewButton("Limpar", func() {
		prox.Store.Clear()
	}).Pad(4)

	v.tabela = juigo.NewTable([]string{"Método", "Status", "Tipo", "URL"}, 0, v.celula).
		BindSelected(v.sel).
		Widths(70, 60, 60)

	// Detalhe: visores ReadOnly em abas + ações de mock.
	v.reqView = juigo.NewCodeEditor().ReadOnly(true).WrapLines(true)
	v.respView = juigo.NewCodeEditor().ReadOnly(true).WrapLines(true)
	v.mocks = juigo.NewVBox().Pad(8).Gap(6)

	mockar := juigo.NewButton("Simular esta resposta", v.criaMockDaSelecao)
	novoMock := juigo.NewButton("Novo mock…", v.novoMock).Pad(4)

	v.abas = juigo.NewTabs().
		Add("Requisição", juigo.NewVBox(juigo.Grow(v.reqView, 1))).
		Add("Resposta", juigo.NewVBox(
			juigo.Grow(v.respView, 1),
			juigo.NewHBox(juigo.NewSpacer(), mockar),
		).Gap(6)).
		Add("Mocks", juigo.NewVBox(
			juigo.NewHBox(juigo.NewText("Regras de simulação").Center(), juigo.NewSpacer(), novoMock),
			juigo.Grow(juigo.NewScroll(v.mocks), 1),
		).Gap(6))

	v.sel.Watch(func(i int) {
		if i >= 0 && i < len(v.visiveis) {
			v.selID = v.visiveis[i].ID
		} else {
			v.selID = -1
		}
		v.carregaDetalhe()
	})

	v.Raiz = juigo.NewVBox(
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("Status:")),
			juigo.Centered(juigo.NewText("").BindText(v.status)),
			juigo.NewSpacer(),
			juigo.Centered(juigo.NewText("Método:")),
			filtroMetodo,
			juigo.Grow(filtroTexto, 1),
			limpar,
		).Gap(8),
		juigo.Grow(juigo.NewHBox(
			juigo.Grow(juigo.NewScroll(v.tabela), 1),
			juigo.Grow(v.abas, 1),
		).Gap(12), 1),
	).Pad(12).Gap(8)

	// A captura vem de goroutines: salta para a main thread e reprojeta.
	prox.Store.OnChange(func() {
		v.post(v.reprojeta)
	})
	v.rebuildMocks()
	return v
}

// SetStatus atualiza o texto de status (main thread).
func (v *Vista) SetStatus(s string) {
	v.status.Set(s)
}

// SelectFirst seleciona a primeira troca visível, se houver — atalho para
// scripts e capturas.
func (v *Vista) SelectFirst() {
	if len(v.visiveis) > 0 {
		v.sel.Set(0)
	}
}

// ShowResponse abre a aba Resposta — atalho para scripts e capturas.
func (v *Vista) ShowResponse() {
	v.abas.Select(1)
}

// celula projeta a célula da tabela a partir da projeção filtrada.
func (v *Vista) celula(linha, coluna int) string {
	if linha < 0 || linha >= len(v.visiveis) {
		return ""
	}
	e := v.visiveis[linha]
	switch coluna {
	case 0:
		if e.Mocked {
			return "◆ " + e.Method // ◆ = resposta simulada
		}
		return e.Method
	case 1:
		return e.StatusText()
	case 2:
		return e.RespType
	default:
		return e.URL
	}
}

// filtra devolve as trocas que passam pelo método e pelo texto correntes.
func (v *Vista) filtra() []*proxy.Exchange {
	met := v.metodo.Get()
	txt := strings.ToLower(strings.TrimSpace(v.filtro.Get()))
	todas := v.prox.Store.Snapshot()
	out := todas[:0]
	for _, e := range todas {
		if met != "Todos" && !strings.EqualFold(e.Method, met) {
			continue
		}
		if txt != "" && !strings.Contains(strings.ToLower(e.URL), txt) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// reprojeta refiltra e restaura a seleção pela identidade.
func (v *Vista) reprojeta() {
	id := v.selID
	v.visiveis = v.filtra()
	idx := -1
	for i, e := range v.visiveis {
		if e.ID == id {
			idx = i
			break
		}
	}
	v.tabela.SetCount(len(v.visiveis))
	if v.sel.Get() != idx {
		v.sel.Set(idx)
	} else {
		v.selID = id
		v.carregaDetalhe()
	}
	v.tabela.Refresh()
}

// carregaDetalhe projeta a troca selecionada nos visores.
func (v *Vista) carregaDetalhe() {
	i := v.sel.Get()
	if i < 0 || i >= len(v.visiveis) {
		v.reqView.SetText("")
		v.respView.SetText("")
		return
	}
	e := v.visiveis[i]
	v.reqView.SetText(e.RequestText)
	v.respView.SetText(e.ResponseText)
	// Realce JSON quando a resposta for JSON (colore o corpo).
	if e.RespType == "json" {
		v.respView.Highlight(syntax.JSON())
	} else {
		v.respView.Highlight(nil)
	}
}

// criaMockDaSelecao cria uma regra de simulação a partir da troca
// selecionada (método + caminho da URL + a resposta capturada).
func (v *Vista) criaMockDaSelecao() {
	i := v.sel.Get()
	if i < 0 || i >= len(v.visiveis) {
		quick.Toast("Selecione uma troca primeiro")
		return
	}
	e := v.visiveis[i]
	if e.Tunnel {
		quick.Toast("Túneis HTTPS não podem ser mockados")
		return
	}
	ct := "text/plain; charset=utf-8"
	if e.RespType == "json" {
		ct = "application/json"
	}
	v.prox.Mocks.Add(&proxy.MockRule{
		Enabled: true, Method: e.Method, URLContains: caminho(e.URL),
		Status: e.Status, ContentType: ct, Body: e.RespBody,
	})
	v.rebuildMocks()
	quick.Toast("Mock criado — próximas chamadas serão simuladas")
}

// novoMock abre um formulário para criar uma regra do zero.
func (v *Vista) novoMock() {
	metodo := quick.Options("Método:", "*", "GET", "POST", "PUT", "DELETE", "PATCH")
	urlSub := quick.Text("URL contém:").Placeholder("/api/saldo")
	status := quick.Number("Status:", 200).Min(100, "100–599").Max(599, "100–599")
	corpo := quick.Notes("Corpo:").Placeholder(`{"ok":true}`)
	quick.Form(metodo, urlSub, status, corpo).Submit("Criar", func() {
		v.prox.Mocks.Add(&proxy.MockRule{
			Enabled: true, Method: metodo.Value(), URLContains: urlSub.Value(),
			Status: status.Value(), ContentType: "application/json", Body: corpo.Value(),
		})
		v.rebuildMocks()
		quick.Toast("Mock criado")
	})
}

// rebuildMocks reconstrói a lista de regras (Clear + Add da projeção).
func (v *Vista) rebuildMocks() {
	v.mocks.Clear()
	regras := v.prox.Mocks.Snapshot()
	if len(regras) == 0 {
		v.mocks.Add(juigo.NewText("Nenhum mock. Crie um a partir de uma resposta ou em “Novo mock…”.").Danger())
		v.mocks.Invalidate()
		return
	}
	for _, r := range regras {
		v.mocks.Add(v.linhaMock(r))
	}
	v.mocks.Invalidate()
}

// linhaMock monta a linha de uma regra: liga/desliga, rótulo e remover.
func (v *Vista) linhaMock(r *proxy.MockRule) juigo.Widget {
	liga := juigo.NewCheckbox("").OnChange(func(bool) {
		v.prox.Mocks.Toggle(r.ID)
	})
	liga.SetChecked(r.Enabled)
	rotulo := fmt.Sprintf("%s → %d (%d bytes)", r.Label(), r.Status, len(r.Body))
	remover := juigo.NewButton("×", func() {
		v.prox.Mocks.Remove(r.ID)
		v.rebuildMocks()
	}).Pad(4)
	return juigo.NewHBox(
		juigo.Centered(liga),
		juigo.Grow(juigo.NewText(rotulo), 1),
		juigo.Centered(remover),
	).Gap(6)
}

// caminho extrai caminho (+query) de uma URL, para casar chamadas ao mesmo
// endpoint em qualquer host.
func caminho(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Path == "" {
		return raw
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}
