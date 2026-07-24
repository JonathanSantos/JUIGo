package uitest_test

import (
	"testing"
	"time"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/form"
	"github.com/JonathanSantos/JUIGo/uitest"
)

// TestFormularioNaInterface é a integração completa: form + bindings + blur
// real (mudança de foco pela Session) + BindDisabled + Submit.
func TestFormularioNaInterface(t *testing.T) {
	nome := juigo.NewState("")
	email := juigo.NewState("")

	f := form.New(
		form.Field(nome, form.Required("Informe o nome"), form.MinRunes(3, "Mínimo de 3 caracteres")),
		form.Field(email, form.Required("Informe o e-mail"), form.Email("E-mail inválido")),
	)

	campoNome := juigo.NewInput("Nome").BindValue(nome)
	campoNome.OnBlur(func() { f.Touch(nome) })
	campoEmail := juigo.NewInput("E-mail").BindValue(email)
	campoEmail.OnBlur(func() { f.Touch(email) })

	salvos := 0
	salvar := juigo.BindDisabled(juigo.NewButton("Salvar", func() {
		f.Submit(func() { salvos++ })
	}), f.Invalid())

	erroNome := juigo.NewText("").BindText(f.ErrorOf(nome))
	erroEmail := juigo.NewText("").BindText(f.ErrorOf(email))

	ui := juigo.NewVBox(campoNome, erroNome, campoEmail, erroEmail, salvar).Pad(8)
	h := uitest.New(t, ui, 420, 300)

	// Botão nasce desabilitado (form inválido) e o clique não salva.
	h.Click(uitest.Text("Salvar"))
	if salvos != 0 || !salvar.Disabled() {
		t.Fatal("com o formulário inválido, Salvar deveria estar desabilitado")
	}

	// Digita nome curto e sai do campo: o BLUR REAL (troca de foco pela
	// Session) toca o campo e o erro aparece na interface.
	h.Click(uitest.Placeholder("Nome"))
	h.Type("çã")
	if erroNome.Text() != "" {
		t.Fatal("erro não deveria aparecer antes do blur")
	}
	h.Click(uitest.Placeholder("E-mail")) // muda o foco → OnBlur do nome
	if erroNome.Text() != "Mínimo de 3 caracteres" {
		t.Fatalf("após blur, erro do nome = %q", erroNome.Text())
	}

	// Corrige o nome: o erro exibido limpa ao vivo enquanto digita. (O
	// relógio avança para o novo clique não virar duplo clique.)
	h.Advance(time.Second)
	h.Click(uitest.Placeholder("Nome"))
	h.Type("o!") // "ção!" ≥ 3 runes
	if erroNome.Text() != "" {
		t.Fatalf("erro deveria limpar ao corrigir: %q", erroNome.Text())
	}

	// E-mail inválido: preenche, o botão segue desabilitado.
	h.Advance(time.Second)
	h.Click(uitest.Placeholder("E-mail"))
	h.Type("jon@exemplo")
	if !salvar.Disabled() {
		t.Fatal("e-mail inválido deveria manter Salvar desabilitado")
	}
	h.Type(".com")
	if salvar.Disabled() {
		t.Fatal("formulário válido deveria habilitar Salvar")
	}

	// Salva de verdade.
	h.Click(uitest.Text("Salvar"))
	if salvos != 1 {
		t.Fatalf("Submit válido deveria salvar; salvos=%d", salvos)
	}
}
