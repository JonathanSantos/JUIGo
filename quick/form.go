package quick

import (
	"juigo/form"
	"juigo/state"
	"juigo/widget"
)

// fieldKind identifica o tipo de controle de um Field.
type fieldKind int

const (
	kindText fieldKind = iota
	kindNotes
	kindOptions
	kindCheck
)

// Field descreve um campo do formulário: rótulo, State vinculado e regras de
// validação. Construa com Text, Notes, Options ou Check, ajuste com os
// modificadores encadeáveis e entregue a Form.
type Field struct {
	kind        fieldKind
	label       string
	str         *state.State[string]
	boolean     *state.State[bool]
	options     []string
	placeholder string
	validators  []form.Validator
	checkMsg    string
}

// Text declara um campo de linha única (Input) vinculado ao State.
func Text(label string, value *state.State[string]) *Field {
	return &Field{kind: kindText, label: label, str: value}
}

// Notes declara um campo multilinha (TextArea) vinculado ao State.
func Notes(label string, value *state.State[string]) *Field {
	return &Field{kind: kindNotes, label: label, str: value}
}

// Options declara um seletor de uma opção (Dropdown) vinculado ao State.
// Com o State vazio e sem Placeholder, ele adota a primeira opção (widget e
// estado sempre de acordo); com Placeholder, nada começa selecionado.
func Options(label string, value *state.State[string], options ...string) *Field {
	return &Field{kind: kindOptions, label: label, str: value, options: options}
}

// Check declara uma caixa de marcação (Checkbox) vinculada ao State; o
// rótulo vive na própria caixa. Required exige a marcação (ex.: termos).
func Check(label string, value *state.State[bool]) *Field {
	return &Field{kind: kindCheck, label: label, boolean: value}
}

// Required exige valor não-vazio (ou, em Check, a caixa marcada), com a
// mensagem dada. Encadeável.
func (f *Field) Required(msg string) *Field {
	if f.kind == kindCheck {
		f.checkMsg = msg
		return f
	}
	f.validators = append(f.validators, form.Required(msg))
	return f
}

// Min exige pelo menos n runes (campos vazios passam — combine com
// Required). Encadeável.
func (f *Field) Min(n int, msg string) *Field {
	f.validators = append(f.validators, form.MinRunes(n, msg))
	return f
}

// Max exige no máximo n runes. Encadeável.
func (f *Field) Max(n int, msg string) *Field {
	f.validators = append(f.validators, form.MaxRunes(n, msg))
	return f
}

// Email exige o formato local@domínio.tld (campos vazios passam — combine
// com Required). Encadeável.
func (f *Field) Email(msg string) *Field {
	f.validators = append(f.validators, form.Email(msg))
	return f
}

// Rules acrescenta validadores arbitrários de juigo/form — a saída de
// emergência para regras que os modificadores prontos não cobrem.
// Encadeável.
func (f *Field) Rules(rules ...form.Validator) *Field {
	f.validators = append(f.validators, rules...)
	return f
}

// Placeholder define o texto exibido com o campo vazio. Encadeável.
func (f *Field) Placeholder(s string) *Field {
	f.placeholder = s
	return f
}

// hasValidation informa se o campo participa da validação (e ganha linha de
// erro na grade).
func (f *Field) hasValidation() bool {
	return len(f.validators) > 0 || f.checkMsg != ""
}

// FormView é o widget montado por Form: uma grade rótulo+campo com linhas de
// erro por campo e, após Submit, a barra com o botão de envio. É um VBox —
// componha-o como qualquer widget (dentro de um Modal, por exemplo).
type FormView struct {
	*widget.VBox
	model  *form.Form
	inputs []*widget.Input
}

// Form monta um formulário validado a partir dos campos: rótulos alinhados
// em grade, controles vinculados aos States, erros exibidos por campo (no
// blur ou no envio, com a semântica "touched" de juigo/form) na cor de erro
// do tema.
func Form(fields ...*Field) *FormView {
	specs := make([]form.FieldSpec, 0, len(fields))
	for _, f := range fields {
		switch {
		case f.kind == kindCheck:
			if f.checkMsg != "" {
				specs = append(specs, form.Check(f.boolean, f.checkMsg))
			}
		default:
			specs = append(specs, form.Field(f.str, f.validators...))
		}
	}
	model := form.New(specs...)

	v := &FormView{VBox: widget.NewVBox(), model: model}
	cells := make([]widget.Widget, 0, 4*len(fields))
	for _, f := range fields {
		var control widget.Widget
		var key any
		switch f.kind {
		case kindText:
			in := widget.NewInput(f.placeholder).BindValue(f.str).
				OnBlur(func() { model.Touch(f.str) })
			v.inputs = append(v.inputs, in)
			control, key = in, f.str
		case kindNotes:
			control = widget.NewTextArea(f.placeholder).BindValue(f.str).
				OnBlur(func() { model.Touch(f.str) })
			key = f.str
		case kindOptions:
			dd := widget.NewDropdown(f.options...).Placeholder(f.placeholder).
				BindValue(f.str).OnChange(func(string) { model.Touch(f.str) })
			if f.str.Get() == "" {
				if f.placeholder != "" {
					dd.Select(-1)
				} else if dd.Value() != "" {
					f.str.Set(dd.Value())
				}
			}
			control, key = dd, f.str
		case kindCheck:
			control = widget.NewCheckbox(f.label).BindChecked(f.boolean).
				OnChange(func(bool) { model.Touch(f.boolean) })
			key = f.boolean
		}
		label := f.label
		if f.kind == kindCheck {
			label = "" // o rótulo vive no próprio checkbox
		}
		cells = append(cells, widget.NewText(label), widget.Grow(control, 1))
		if f.hasValidation() {
			cells = append(cells,
				widget.NewText(""),
				widget.NewText("").BindText(model.ErrorOf(key)).Danger(),
			)
		}
	}
	v.Add(widget.NewGrid(2, cells...))
	return v
}

// Submit acrescenta a barra com o botão de envio: o clique (ou Enter em
// qualquer campo de linha única) valida tudo, passa a exibir os erros e, com
// o formulário válido, chama fn. Encadeável.
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
