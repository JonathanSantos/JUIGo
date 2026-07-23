// Package form é a camada de validação declarativa do JUIGo — construída
// inteiramente sobre estados (juigo/state), sem depender de widgets: cada
// campo é um State com validadores, os erros são States derivados e a
// validade do formulário é o E-lógico de todos. Tudo compõe com os bindings
// existentes:
//
//	nome := state.New("")
//	email := state.New("")
//	f := form.New(
//	    form.Field(nome, form.Required("Informe o nome"), form.MinRunes(3, "Mínimo de 3 caracteres")),
//	    form.Field(email, form.Required("Informe o e-mail"), form.Email("E-mail inválido")),
//	)
//	// erros na tela: NewText("").BindText(f.ErrorOf(nome)) com Theme.Danger
//	// botão preso à validade: BindDisabled(salvar, f.Invalid())
//	// envio: salvar.OnClick = func() { f.Submit(salvarDeVerdade) }
//
// Exibição de erros ("touched"): ErrorOf devolve o erro EXIBIDO, que fica
// vazio até o campo ser tocado (Touch — normalmente ligado ao OnBlur do
// widget) ou até o primeiro Submit; a partir daí, acompanha a digitação ao
// vivo. Valid/Invalid sempre refletem a validade real, independentemente da
// exibição.
package form

import (
	"github.com/JonathanSantos/JUIGo/state"
)

// Validator valida um valor e devolve a mensagem de erro ("" = válido).
type Validator func(v string) string

// FieldSpec descreve um campo do formulário; construa com Field ou Check e
// entregue a New.
type FieldSpec struct {
	key     any
	compute func() string
	sources []state.Observable
}

// Field declara um campo textual: o State do valor (o mesmo vinculado ao
// widget com BindValue) e seus validadores, aplicados na ordem — a primeira
// mensagem não-vazia vence.
func Field(value *state.State[string], validators ...Validator) FieldSpec {
	return FieldSpec{
		key: value,
		compute: func() string {
			for _, v := range validators {
				if msg := v(value.Get()); msg != "" {
					return msg
				}
			}
			return ""
		},
		sources: []state.Observable{value},
	}
}

// Check declara uma regra booleana (ex.: "aceite os termos"): inválida
// enquanto o State for false.
func Check(value *state.State[bool], msg string) FieldSpec {
	return FieldSpec{
		key: value,
		compute: func() string {
			if !value.Get() {
				return msg
			}
			return ""
		},
		sources: []state.Observable{value},
	}
}

// Rule declara uma regra arbitrária dependente de múltiplos estados (ex.:
// "senhas devem coincidir"): compute devolve a mensagem ("" = válido) e é
// reavaliada quando qualquer fonte muda. key identifica a regra em ErrorOf
// (use qualquer valor comparável, ex.: uma string).
func Rule(key any, compute func() string, sources ...state.Observable) FieldSpec {
	return FieldSpec{key: key, compute: compute, sources: sources}
}

// fieldState é o estado interno de um campo no formulário.
type fieldState struct {
	err     *state.State[string] // erro ao vivo
	shown   *state.State[string] // erro exibido (touched/submitted)
	touched bool
	compute func() string
}

// Form agrega os campos e deriva a validade.
type Form struct {
	order     []any
	fields    map[any]*fieldState
	valid     *state.State[bool]
	invalid   *state.State[bool]
	submitted bool
}

// New monta o formulário e começa a observar os estados dos campos.
func New(specs ...FieldSpec) *Form {
	f := &Form{fields: make(map[any]*fieldState)}
	for _, spec := range specs {
		fs := &fieldState{
			err:     state.New(spec.compute()),
			shown:   state.New(""),
			compute: spec.compute,
		}
		f.order = append(f.order, spec.key)
		f.fields[spec.key] = fs
		for _, src := range spec.sources {
			src.WatchChange(func() {
				setIfChanged(fs.err, fs.compute())
				if fs.touched || f.submitted {
					setIfChanged(fs.shown, fs.err.Get())
				}
				f.refresh()
			})
		}
	}
	f.valid = state.New(f.allValid())
	f.invalid = state.New(!f.valid.Get())
	return f
}

// Valid devolve o State da validade real do formulário (sempre ao vivo).
func (f *Form) Valid() *state.State[bool] {
	return f.valid
}

// Invalid devolve o inverso de Valid — pronto para BindDisabled.
func (f *Form) Invalid() *state.State[bool] {
	return f.invalid
}

// ErrorOf devolve o State do erro EXIBIDO do campo identificado por key (o
// mesmo State/chave passado a Field/Check/Rule) — vazio até Touch ou o
// primeiro Submit. Para chaves desconhecidas, devolve um State vazio inerte.
func (f *Form) ErrorOf(key any) *state.State[string] {
	if fs, ok := f.fields[key]; ok {
		return fs.shown
	}
	return state.New("")
}

// Touch marca o campo como tocado (normalmente no OnBlur do widget): o erro
// dele passa a ser exibido e acompanhará a digitação.
func (f *Form) Touch(key any) {
	fs, ok := f.fields[key]
	if !ok {
		return
	}
	fs.touched = true
	setIfChanged(fs.shown, fs.err.Get())
}

// Submit valida tudo, passa a exibir todos os erros (ao vivo daqui em
// diante) e, se o formulário estiver válido, chama fn (que pode ser nil).
// Devolve a validade.
func (f *Form) Submit(fn func()) bool {
	f.submitted = true
	for _, k := range f.order {
		fs := f.fields[k]
		setIfChanged(fs.err, fs.compute())
		setIfChanged(fs.shown, fs.err.Get())
	}
	f.refresh()
	if !f.valid.Get() {
		return false
	}
	if fn != nil {
		fn()
	}
	return true
}

// refresh recalcula os agregados de validade.
func (f *Form) refresh() {
	ok := f.allValid()
	setIfChanged(f.valid, ok)
	setIfChanged(f.invalid, !ok)
}

func (f *Form) allValid() bool {
	for _, k := range f.order {
		if f.fields[k].err.Get() != "" {
			return false
		}
	}
	return true
}

// setIfChanged evita notificações (e repaints) redundantes.
func setIfChanged[T comparable](s *state.State[T], v T) {
	if s.Get() != v {
		s.Set(v)
	}
}
