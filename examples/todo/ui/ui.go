// Package ui é a camada de interface do TodoMVC: uma Vista (view-model) que
// PROJETA o domínio em widgets e traduz eventos em chamadas de domínio.
// O fluxo é um ciclo único e explícito:
//
//	evento → aplica(mutação) → domínio muda → repositório salva → reprojeta()
//
// reprojeta() recalcula os estados derivados e RECONSTRÓI as linhas da
// lista (Clear + Add) a partir da projeção filtrada — o padrão para listas
// dinâmicas com linhas interativas: sem reconciliação mágica, a lista é
// função dos dados. A edição inline abre no duplo clique (widget custom
// "titulo" com janela de tempo via juigo.After) e o campo nasce focado
// (juigo.Focus).
package ui

import (
	"fmt"
	"image"
	"log"

	"juigo"
	"juigo/examples/todo/tarefas"
	"juigo/render"
	"time"
)

// Vista é o view-model: estados derivados do domínio e a raiz da árvore.
type Vista struct {
	lista *tarefas.Lista
	repo  tarefas.Repositorio

	filtro        *juigo.State[string]
	novoTexto     *juigo.State[string]
	resumo        *juigo.State[string]
	temConcluidas *juigo.State[bool]
	editando      int // ID da tarefa em edição inline; -1 = nenhuma

	linhas      *juigo.VBox
	marcarTodas *juigo.Checkbox

	// Raiz é a árvore de widgets pronta para App.SetRoot.
	Raiz juigo.Widget
}

// New monta a Vista sobre o domínio e o repositório dados.
func New(lista *tarefas.Lista, repo tarefas.Repositorio) *Vista {
	v := &Vista{
		lista:         lista,
		repo:          repo,
		editando:      -1,
		filtro:        juigo.NewState(string(tarefas.FiltroTodas)),
		novoTexto:     juigo.NewState(""),
		resumo:        juigo.NewState(""),
		temConcluidas: juigo.NewState(false),
		linhas:        juigo.NewVBox(),
	}

	v.marcarTodas = juigo.Tooltip(juigo.NewCheckbox("").OnChange(func(marcar bool) {
		v.aplica(func() { v.lista.MarcarTodas(marcar) })
	}), "Marcar/desmarcar todas")

	novo := juigo.NewInput("O que precisa ser feito?").BindValue(v.novoTexto)
	novo.OnSubmit(func() {
		v.aplica(func() { v.lista.Adicionar(v.novoTexto.Get()) })
		v.novoTexto.Set("")
	})

	v.filtro.Watch(func(string) { v.reprojeta() })

	v.Raiz = juigo.NewVBox(
		juigo.NewText("tarefas").Center(),
		juigo.NewHBox(
			juigo.Centered(v.marcarTodas),
			juigo.Grow(novo, 1),
		),
		juigo.Grow(juigo.NewScroll(v.linhas), 1),
		juigo.NewHBox(
			juigo.Centered(juigo.NewText("").BindText(v.resumo)),
			juigo.NewSpacer(),
			juigo.BindDisabled(juigo.NewButton("Limpar concluídas", func() {
				v.aplica(func() { v.lista.LimparConcluidas() })
			}), juigo.Not(v.temConcluidas)),
		),
		juigo.NewHBox(
			juigo.NewSpacer(),
			juigo.NewRadio(string(tarefas.FiltroTodas), string(tarefas.FiltroTodas)).BindValue(v.filtro),
			juigo.NewRadio(string(tarefas.FiltroAtivas), string(tarefas.FiltroAtivas)).BindValue(v.filtro),
			juigo.NewRadio(string(tarefas.FiltroConcluidas), string(tarefas.FiltroConcluidas)).BindValue(v.filtro),
			juigo.NewSpacer(),
		),
	).Pad(16).Gap(8)

	v.reprojeta()
	return v
}

// aplica executa a mutação no domínio, persiste e re-projeta — TODO evento
// que muda dados passa por aqui (fluxo único).
func (v *Vista) aplica(muta func()) {
	muta()
	if err := v.repo.Salvar(v.lista.Todas()); err != nil {
		log.Printf("todo: falha ao salvar: %v", err)
	}
	v.reprojeta()
}

// reprojeta recalcula os estados derivados e reconstrói as linhas a partir
// da projeção filtrada do domínio.
func (v *Vista) reprojeta() {
	n := v.lista.Restantes()
	texto := fmt.Sprintf("%d itens restantes", n)
	if n == 1 {
		texto = "1 item restante"
	}
	v.resumo.Set(texto)
	v.temConcluidas.Set(v.lista.TemConcluidas())
	v.marcarTodas.SetChecked(v.lista.TodasConcluidas()) // SetChecked não dispara OnChange

	v.linhas.Clear()
	for _, t := range v.lista.Filtradas(tarefas.Filtro(v.filtro.Get())) {
		v.linhas.Add(v.linha(t))
	}
}

// linha monta a linha de uma tarefa: normal (checkbox + título + excluir)
// ou em edição inline (um campo focado que confirma no Enter/blur; editar
// para vazio remove — semântica TodoMVC).
func (v *Vista) linha(t tarefas.Tarefa) juigo.Widget {
	if t.ID == v.editando {
		campo := juigo.NewInput("")
		campo.SetText(t.Titulo)
		commit := func() {
			if v.editando != t.ID {
				return // Enter e blur podem ambos disparar; só o 1º vale
			}
			v.editando = -1
			texto := campo.Text()
			v.aplica(func() { v.lista.Renomear(t.ID, texto) })
		}
		campo.OnSubmit(func() { commit(); juigo.Blur() }).OnBlur(commit)
		juigo.Focus(campo) // a edição abre pronta para digitar
		return juigo.NewHBox(juigo.Grow(campo, 1))
	}

	marcada := juigo.NewCheckbox("").OnChange(func(bool) {
		v.aplica(func() { v.lista.Alternar(t.ID) })
	})
	marcada.SetChecked(t.Concluida)
	rotulo := &titulo{texto: t.Titulo, riscado: t.Concluida}
	rotulo.aoDuploClique = func() {
		v.editando = t.ID
		v.reprojeta() // sem mutação de domínio: só a projeção muda
	}
	excluir := juigo.Tooltip(juigo.NewButton("×", func() {
		v.aplica(func() { v.lista.Remover(t.ID) })
	}).Pad(4), "Excluir tarefa")
	return juigo.NewHBox(
		juigo.Centered(marcada),
		juigo.Grow(rotulo, 1),
		juigo.Centered(excluir),
	)
}

// titulo é o widget custom do título da tarefa: risca quando concluída e
// reconhece DUPLO clique (janela de 400ms medida com juigo.After — o mesmo
// relógio da aplicação, virtual no uitest).
type titulo struct {
	juigo.BaseWidget
	texto         string
	riscado       bool
	aoDuploClique func()
	armado        bool
}

func (t *titulo) PreferredSize() image.Point {
	th := t.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Point{X: th.MeasureString(t.texto), Y: th.LineHeight() + th.Px(4)}
}

func (t *titulo) HandleEvent(ev juigo.Event) bool {
	e, ok := ev.(juigo.MouseEvent)
	if !ok || e.Kind != juigo.MouseDown || e.Button != juigo.MouseButtonLeft {
		return false
	}
	if t.armado {
		t.armado = false
		if t.aoDuploClique != nil {
			t.aoDuploClique()
		}
		return true
	}
	t.armado = true
	juigo.After(400*time.Millisecond, func() { t.armado = false })
	return true
}

func (t *titulo) Draw(dst *image.RGBA) {
	th := t.Theme()
	if th == nil {
		return
	}
	b := t.Bounds()
	cor := th.Text
	if t.riscado {
		cor = th.Placeholder
	}
	baseline := b.Min.Y + (b.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(dst, t.texto, image.Pt(b.Min.X, baseline), cor)
	if t.riscado {
		meio := b.Min.Y + b.Dy()/2
		render.FillRect(dst, image.Rect(b.Min.X, meio, b.Min.X+th.MeasureString(t.texto), meio+1), th.Placeholder)
	}
}
