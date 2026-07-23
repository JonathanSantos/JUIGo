package form_test

import (
	"testing"

	"juigo/form"
	"juigo/state"
)

func TestValidadores(t *testing.T) {
	cases := []struct {
		nome  string
		v     form.Validator
		in    string
		valid bool
	}{
		{"Required vazio", form.Required("x"), "", false},
		{"Required espaços", form.Required("x"), "   ", false},
		{"Required ok", form.Required("x"), "olá", true},
		{"MinRunes curto", form.MinRunes(3, "x"), "çã", false},
		{"MinRunes exato (runes, não bytes)", form.MinRunes(3, "x"), "çãé", true},
		{"MinRunes vazio passa", form.MinRunes(3, "x"), "", true},
		{"MaxRunes longo", form.MaxRunes(3, "x"), "abcd", false},
		{"Email inválido", form.Email("x"), "foo@bar", false},
		{"Email ok", form.Email("x"), "foo@bar.com", true},
		{"Email vazio passa", form.Email("x"), "", true},
	}
	for _, c := range cases {
		got := c.v(c.in) == ""
		if got != c.valid {
			t.Fatalf("%s: válido=%v, esperado %v", c.nome, got, c.valid)
		}
	}
}

func TestFormularioCicloCompleto(t *testing.T) {
	nome := state.New("")
	email := state.New("")
	termos := state.New(false)

	f := form.New(
		form.Field(nome, form.Required("nome obrigatório"), form.MinRunes(3, "nome curto")),
		form.Field(email, form.Required("e-mail obrigatório"), form.Email("e-mail inválido")),
		form.Check(termos, "aceite os termos"),
	)

	// Validade real é ao vivo; erros exibidos começam vazios.
	if f.Valid().Get() || !f.Invalid().Get() {
		t.Fatal("formulário vazio deveria ser inválido")
	}
	if f.ErrorOf(nome).Get() != "" {
		t.Fatal("erro não deveria ser exibido antes de Touch/Submit")
	}

	// Touch exibe o erro do campo — e ele passa a acompanhar a digitação.
	f.Touch(nome)
	if f.ErrorOf(nome).Get() != "nome obrigatório" {
		t.Fatalf("após Touch: %q", f.ErrorOf(nome).Get())
	}
	nome.Set("çã")
	if f.ErrorOf(nome).Get() != "nome curto" {
		t.Fatalf("erro deveria acompanhar a digitação: %q", f.ErrorOf(nome).Get())
	}
	nome.Set("Jonathan")
	if f.ErrorOf(nome).Get() != "" {
		t.Fatalf("campo válido deveria limpar o erro: %q", f.ErrorOf(nome).Get())
	}
	// O outro campo continua com erro oculto.
	if f.ErrorOf(email).Get() != "" {
		t.Fatal("campo não tocado não deveria exibir erro")
	}

	// Submit inválido: não chama fn, exibe TODOS os erros.
	chamado := false
	if f.Submit(func() { chamado = true }) || chamado {
		t.Fatal("Submit inválido não deveria chamar fn")
	}
	if f.ErrorOf(email).Get() != "e-mail obrigatório" || f.ErrorOf(termos).Get() != "aceite os termos" {
		t.Fatalf("Submit deveria exibir todos: email=%q termos=%q",
			f.ErrorOf(email).Get(), f.ErrorOf(termos).Get())
	}

	// Corrige tudo: validade reage; erros exibidos limpam ao vivo.
	email.Set("jonathan@exemplo.com")
	termos.Set(true)
	if !f.Valid().Get() || f.ErrorOf(email).Get() != "" {
		t.Fatalf("formulário corrigido: valid=%v erro=%q", f.Valid().Get(), f.ErrorOf(email).Get())
	}

	// Submit válido chama fn.
	if !f.Submit(func() { chamado = true }) || !chamado {
		t.Fatal("Submit válido deveria chamar fn")
	}
}

func TestRuleMultiplasFontes(t *testing.T) {
	senha := state.New("")
	confirma := state.New("")
	f := form.New(
		form.Field(senha, form.Required("senha obrigatória")),
		form.Rule("confirma", func() string {
			if senha.Get() != confirma.Get() {
				return "senhas não coincidem"
			}
			return ""
		}, senha, confirma),
	)

	senha.Set("segredo")
	if f.Valid().Get() {
		t.Fatal("confirmação divergente deveria invalidar")
	}
	confirma.Set("segredo")
	if !f.Valid().Get() {
		t.Fatal("senhas iguais deveriam validar")
	}
	// Regra multi-fonte reage a QUALQUER fonte.
	senha.Set("outra")
	if f.Valid().Get() {
		t.Fatal("mudar a senha deveria invalidar de novo")
	}
	if f.Submit(nil) {
		t.Fatal("Submit inválido deveria devolver false")
	}
	if f.ErrorOf("confirma").Get() != "senhas não coincidem" {
		t.Fatalf("erro da regra: %q", f.ErrorOf("confirma").Get())
	}
}
