package quick

import (
	"strconv"
	"strings"

	"juigo/form"
	"juigo/state"
	"juigo/widget"
)

// Field é o núcleo comum dos campos do quick.Form: um HANDLE TIPADO que é
// dono do State do valor. A variável do campo é a "chave" do formulário —
// tipada, verificada pelo compilador e refatorável:
//
//	nome := quick.Text("Nome:").Required("Informe o nome")
//	quick.Form(nome, …).Submit("Salvar", func() { salvar(nome.Value()) })
//
// O estado nasce interno por padrão; Bind (nos tipos concretos) o troca por
// um State externo quando o valor precisa viver fora do formulário.
type Field[T any] struct {
	label string
	st    *state.State[T]
}

// Value devolve o valor atual do campo.
func (f *Field[T]) Value() T {
	return f.st.Get()
}

// Set define o valor do campo (refletido no controle pelo binding).
func (f *Field[T]) Set(v T) {
	f.st.Set(v)
}

// State devolve o State do valor — a saída de emergência para reatividade
// além do formulário (Map, Watch, bindings em outros widgets).
func (f *Field[T]) State() *state.State[T] {
	return f.st
}

// FormItem é o contrato dos itens aceitos por Form (campos e Section).
// É selado: implementado apenas pelos tipos deste pacote.
type FormItem interface {
	addTo(a *assembly)
}

// assembly acumula a montagem do formulário em duas fases: primeiro as
// specs de validação (form.New precisa de todas), depois as linhas da grade
// (os controles precisam do model para Touch/ErrorOf). Section fecha a
// grade corrente e abre outra — por isso as células ficam aqui.
type assembly struct {
	specs []form.FieldSpec
	steps []func(m *form.Form, v *FormView)
	cells []widget.Widget
}

// addRow acrescenta a linha rótulo+controle e, com validação, a linha de
// erro (vazia até o touch/envio, na cor Danger do tema).
func (a *assembly) addRow(label string, control widget.Widget, m *form.Form, key any, hasErr bool) {
	a.cells = append(a.cells, widget.NewText(label), widget.Grow(control, 1))
	if hasErr {
		a.cells = append(a.cells,
			widget.NewText(""),
			widget.NewText("").BindText(m.ErrorOf(key)).Danger(),
		)
	}
}

// flush fecha a grade corrente (se houver células pendentes).
func (a *assembly) flush(v *FormView) {
	if len(a.cells) > 0 {
		v.Add(widget.NewGrid(2, a.cells...))
		a.cells = nil
	}
}

// TextField é o campo textual (Input de linha única ou TextArea via Notes).
type TextField struct {
	Field[string]
	multiline   bool
	placeholder string
	validators  []form.Validator
}

// Text declara um campo de linha única; o valor nasce num State interno
// (acesse com Value/State; troque por um externo com Bind).
func Text(label string) *TextField {
	return &TextField{Field: Field[string]{label: label, st: state.New("")}}
}

// Notes declara um campo multilinha (TextArea).
func Notes(label string) *TextField {
	f := Text(label)
	f.multiline = true
	return f
}

// Bind troca o State interno pelo externo dado (chame antes de Form).
// Encadeável.
func (f *TextField) Bind(s *state.State[string]) *TextField {
	f.st = s
	return f
}

// Required exige valor não-vazio, com a mensagem dada. Encadeável.
func (f *TextField) Required(msg string) *TextField {
	f.validators = append(f.validators, form.Required(msg))
	return f
}

// Min exige pelo menos n runes (campos vazios passam — combine com
// Required). Encadeável.
func (f *TextField) Min(n int, msg string) *TextField {
	f.validators = append(f.validators, form.MinRunes(n, msg))
	return f
}

// Max exige no máximo n runes. Encadeável.
func (f *TextField) Max(n int, msg string) *TextField {
	f.validators = append(f.validators, form.MaxRunes(n, msg))
	return f
}

// Email exige o formato local@domínio.tld (campos vazios passam — combine
// com Required). Encadeável.
func (f *TextField) Email(msg string) *TextField {
	f.validators = append(f.validators, form.Email(msg))
	return f
}

// Rules acrescenta validadores arbitrários de juigo/form — a saída de
// emergência para regras que os modificadores prontos não cobrem.
// Encadeável.
func (f *TextField) Rules(rules ...form.Validator) *TextField {
	f.validators = append(f.validators, rules...)
	return f
}

// Placeholder define o texto exibido com o campo vazio. Encadeável.
func (f *TextField) Placeholder(s string) *TextField {
	f.placeholder = s
	return f
}

func (f *TextField) addTo(a *assembly) {
	hasErr := len(f.validators) > 0
	if hasErr {
		a.specs = append(a.specs, form.Field(f.st, f.validators...))
	}
	a.steps = append(a.steps, func(m *form.Form, v *FormView) {
		var control widget.Widget
		if f.multiline {
			control = widget.NewTextArea(f.placeholder).BindValue(f.st).
				OnBlur(func() { m.Touch(f.st) })
		} else {
			in := widget.NewInput(f.placeholder).BindValue(f.st).
				OnBlur(func() { m.Touch(f.st) })
			v.inputs = append(v.inputs, in)
			control = in
		}
		a.addRow(f.label, control, m, f.st, hasErr)
	})
}

// NumberField é o campo numérico inteiro: um Input que só aceita dígitos
// (e um sinal de menos inicial), com o valor tipado em int.
type NumberField struct {
	Field[int]
	initial int
	checks  []func(v int) string
}

// Number declara um campo numérico inteiro com o valor inicial dado. O
// campo vazio (ou só o sinal) vale o valor INICIAL — combine Min/Max para
// faixas. Value devolve int.
func Number(label string, initial int) *NumberField {
	return &NumberField{
		Field:   Field[int]{label: label, st: state.New(initial)},
		initial: initial,
	}
}

// Bind troca o State interno pelo externo dado (chame antes de Form).
// Encadeável.
func (f *NumberField) Bind(s *state.State[int]) *NumberField {
	f.st = s
	return f
}

// Min exige valor >= n, com a mensagem dada. Encadeável.
func (f *NumberField) Min(n int, msg string) *NumberField {
	f.checks = append(f.checks, func(v int) string {
		if v < n {
			return msg
		}
		return ""
	})
	return f
}

// Max exige valor <= n, com a mensagem dada. Encadeável.
func (f *NumberField) Max(n int, msg string) *NumberField {
	f.checks = append(f.checks, func(v int) string {
		if v > n {
			return msg
		}
		return ""
	})
	return f
}

func (f *NumberField) addTo(a *assembly) {
	hasErr := len(f.checks) > 0
	if hasErr {
		a.specs = append(a.specs, form.Rule(f.st, func() string {
			v := f.st.Get()
			for _, c := range f.checks {
				if msg := c(v); msg != "" {
					return msg
				}
			}
			return ""
		}, f.st))
	}
	a.steps = append(a.steps, func(m *form.Form, v *FormView) {
		// O texto editável vive num State próprio, sincronizado em duas
		// vias com o valor int; o filtro garante que ele sempre parseia
		// (vazio ou só "-" caem no valor inicial).
		str := state.New(strconv.Itoa(f.st.Get()))
		var in *widget.Input
		in = widget.NewInput("").BindValue(str).
			Filter(func(r rune) bool {
				if r >= '0' && r <= '9' {
					return true
				}
				return r == '-' && in.Cursor() == 0 && !strings.ContainsRune(in.Text(), '-')
			}).
			OnBlur(func() { m.Touch(f.st) })
		syncing := false
		str.Watch(func(s string) {
			v, err := strconv.Atoi(s)
			if err != nil {
				v = f.initial
			}
			if f.st.Get() != v {
				syncing = true
				f.st.Set(v)
				syncing = false
			}
		})
		f.st.Watch(func(v int) {
			if syncing {
				return
			}
			if s := strconv.Itoa(v); str.Get() != s {
				str.Set(s)
			}
		})
		v.inputs = append(v.inputs, in)
		a.addRow(f.label, in, m, f.st, hasErr)
	})
}

// OptionsField é o seletor de uma opção (Dropdown).
type OptionsField struct {
	Field[string]
	options     []string
	placeholder string
	validators  []form.Validator
}

// Options declara um seletor com as opções dadas. Com o valor vazio e sem
// Placeholder, ele adota a primeira opção (controle e valor sempre de
// acordo); com Placeholder, nada começa selecionado.
func Options(label string, options ...string) *OptionsField {
	return &OptionsField{
		Field:   Field[string]{label: label, st: state.New("")},
		options: options,
	}
}

// Bind troca o State interno pelo externo dado (chame antes de Form).
// Encadeável.
func (f *OptionsField) Bind(s *state.State[string]) *OptionsField {
	f.st = s
	return f
}

// Required exige uma opção escolhida — útil com Placeholder, em que nada
// começa selecionado. Encadeável.
func (f *OptionsField) Required(msg string) *OptionsField {
	f.validators = append(f.validators, form.Required(msg))
	return f
}

// Placeholder define o texto exibido sem seleção. Encadeável.
func (f *OptionsField) Placeholder(s string) *OptionsField {
	f.placeholder = s
	return f
}

func (f *OptionsField) addTo(a *assembly) {
	hasErr := len(f.validators) > 0
	if hasErr {
		a.specs = append(a.specs, form.Field(f.st, f.validators...))
	}
	a.steps = append(a.steps, func(m *form.Form, v *FormView) {
		dd := widget.NewDropdown(f.options...).Placeholder(f.placeholder).
			BindValue(f.st).OnChange(func(string) { m.Touch(f.st) })
		if f.st.Get() == "" {
			if f.placeholder != "" {
				dd.Select(-1)
			} else if dd.Value() != "" {
				f.st.Set(dd.Value())
			}
		}
		a.addRow(f.label, dd, m, f.st, hasErr)
	})
}

// CheckField é a caixa de marcação (Checkbox); o rótulo vive na própria
// caixa.
type CheckField struct {
	Field[bool]
	requiredMsg string
}

// Check declara uma caixa de marcação. Required exige a marcação (ex.:
// aceite de termos).
func Check(label string) *CheckField {
	return &CheckField{Field: Field[bool]{label: label, st: state.New(false)}}
}

// Bind troca o State interno pelo externo dado (chame antes de Form).
// Encadeável.
func (f *CheckField) Bind(s *state.State[bool]) *CheckField {
	f.st = s
	return f
}

// Required exige a caixa marcada, com a mensagem dada. Encadeável.
func (f *CheckField) Required(msg string) *CheckField {
	f.requiredMsg = msg
	return f
}

func (f *CheckField) addTo(a *assembly) {
	hasErr := f.requiredMsg != ""
	if hasErr {
		a.specs = append(a.specs, form.Check(f.st, f.requiredMsg))
	}
	a.steps = append(a.steps, func(m *form.Form, v *FormView) {
		cb := widget.NewCheckbox(f.label).BindChecked(f.st).
			OnChange(func(bool) { m.Touch(f.st) })
		a.addRow("", cb, m, f.st, hasErr)
	})
}

// SectionItem é o divisor de seções de um formulário (ver Section).
type SectionItem struct {
	title string
}

// Section abre uma seção no formulário: fecha a grade corrente, exibe o
// título na largura total e começa outra grade (as larguras de coluna são
// medidas por seção).
func Section(title string) *SectionItem {
	return &SectionItem{title: title}
}

func (s *SectionItem) addTo(a *assembly) {
	a.steps = append(a.steps, func(m *form.Form, v *FormView) {
		a.flush(v)
		v.Add(widget.NewText(s.title))
	})
}

// FormView é o widget montado por Form: as grades rótulo+campo com linhas
// de erro por campo e, após Submit, a barra com o botão de envio. É um VBox
// — componha-o como qualquer widget (dentro de um Modal, por exemplo).
type FormView struct {
	*widget.VBox
	model  *form.Form
	inputs []*widget.Input
}

// Form monta um formulário validado a partir dos itens (campos e seções):
// rótulos alinhados em grade, controles vinculados aos States dos handles,
// erros exibidos por campo (no blur ou no envio, com a semântica "touched"
// de juigo/form) na cor de erro do tema. Leia os valores pelos próprios
// handles (campo.Value()).
func Form(items ...FormItem) *FormView {
	a := &assembly{}
	for _, it := range items {
		it.addTo(a)
	}
	model := form.New(a.specs...)
	v := &FormView{VBox: widget.NewVBox(), model: model}
	for _, step := range a.steps {
		step(model, v)
	}
	a.flush(v)
	return v
}

// Gap define o espaço entre as linhas do formulário, em unidades lógicas.
// Encadeável.
func (v *FormView) Gap(spacing int) *FormView {
	v.VBox.Gap(spacing)
	return v
}

// Pad define o espaço interno das bordas, em unidades lógicas. Encadeável.
func (v *FormView) Pad(padding int) *FormView {
	v.VBox.Pad(padding)
	return v
}

// Submit acrescenta a barra com o botão de envio: o clique (ou Enter em
// qualquer campo de linha única) valida tudo, passa a exibir os erros e,
// com o formulário válido, chama fn. Encadeável.
func (v *FormView) Submit(label string, fn func()) *FormView {
	submit := func() { v.model.Submit(fn) }
	for _, in := range v.inputs {
		in.OnSubmit(submit)
	}
	v.Add(Buttons(widget.NewButton(label, submit)))
	return v
}

// Model devolve o form subjacente — a saída de emergência para Touch,
// Valid/Invalid e ErrorOf diretos (BindDisabled no botão, por exemplo).
func (v *FormView) Model() *form.Form {
	return v.model
}
