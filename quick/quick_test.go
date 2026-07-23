package quick_test

import (
	"testing"

	"juigo"
	"juigo/quick"
	"juigo/uitest"
)

// TestFormFluxoDeValidacao cobre o ciclo completo do quick.Form: erros só
// aparecem depois do envio (ou do blur), preencher corrige ao vivo, o envio
// válido chama o callback e Enter em um campo também envia.
func TestFormFluxoDeValidacao(t *testing.T) {
	nome := juigo.NewState("")
	mail := juigo.NewState("")
	termos := juigo.NewState(false)
	salvos := 0
	f := quick.Form(
		quick.Text("Nome:", nome).Placeholder("nome").
			Required("Informe o nome").Min(3, "Mínimo de 3 caracteres"),
		quick.Text("E-mail:", mail).Placeholder("mail").Email("E-mail inválido"),
		quick.Check("Aceito os termos", termos).Required("Aceite os termos"),
	).Submit("Salvar", func() { salvos++ })
	h := uitest.New(t, f, 480, 360)

	// Nada tocado: nenhum erro na tela.
	if h.Find(uitest.Text("Informe o nome")) != nil {
		t.Fatal("erro não deveria aparecer antes do envio ou do blur")
	}

	// Envio inválido: revela os erros e não chama o callback.
	h.Click(uitest.Text("Salvar"))
	if salvos != 0 {
		t.Fatal("envio inválido não deveria chamar o callback")
	}
	if h.Find(uitest.Text("Informe o nome")) == nil {
		t.Fatal("o envio deveria revelar o erro do nome")
	}
	if h.Find(uitest.Text("Aceite os termos")) == nil {
		t.Fatal("o envio deveria revelar o erro do checkbox")
	}

	// Preencher corrige ao vivo (pós-envio, os erros acompanham a digitação).
	h.Click(uitest.Placeholder("nome"))
	h.Type("Jo")
	if h.Find(uitest.Text("Mínimo de 3 caracteres")) == nil {
		t.Fatal("erro de mínimo deveria aparecer ao vivo")
	}
	h.Type("natan")
	if h.Find(uitest.Text("Mínimo de 3 caracteres")) != nil {
		t.Fatal("erro de mínimo deveria sumir com o campo válido")
	}
	h.Click(uitest.Placeholder("mail"))
	h.Type("jo@exemplo.com")
	h.Click(uitest.Text("Aceito os termos"))

	// Envio válido chama o callback.
	h.Click(uitest.Text("Salvar"))
	if salvos != 1 {
		t.Fatalf("envio válido deveria chamar o callback; salvos = %d", salvos)
	}

	// Enter em um campo de linha única também envia.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyEnter)
	if salvos != 2 {
		t.Fatalf("Enter no campo deveria enviar; salvos = %d", salvos)
	}
}

// TestFormOptionsAdotaPrimeira: com o State vazio e sem placeholder, o
// Dropdown e o State concordam na primeira opção desde o início.
func TestFormOptionsAdotaPrimeira(t *testing.T) {
	plano := juigo.NewState("")
	f := quick.Form(quick.Options("Plano:", plano, "free", "pro"))
	uitest.New(t, f, 400, 200)
	if plano.Get() != "free" {
		t.Fatalf("State deveria adotar a primeira opção; got %q", plano.Get())
	}
}

// TestConfirmResultado cobre os dois desfechos do Confirm: OK devolve true e
// Escape (com o foco no botão Cancel, via autofoco da overlay) devolve false.
func TestConfirmResultado(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("raiz")), 400, 300)

	var got *bool
	quick.Confirm("Apagar tudo?", func(ok bool) { got = &ok })
	if h.Session().Overlay() == nil {
		t.Fatal("Confirm deveria abrir a overlay")
	}
	h.Click(uitest.Text("OK"))
	if got == nil || !*got {
		t.Fatal("OK deveria entregar true")
	}
	if h.Session().Overlay() != nil {
		t.Fatal("o diálogo deveria fechar após o OK")
	}

	got = nil
	quick.Confirm("De novo?", func(ok bool) { got = &ok })
	h.Key(juigo.KeyEscape)
	if got == nil || *got {
		t.Fatal("Escape deveria entregar false")
	}
}

// TestAlertEnterDispensa: o OK do Alert abre focado — Enter dispensa.
func TestAlertEnterDispensa(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("raiz")), 400, 300)
	quick.Alert("Algo deu errado")
	if h.Find(uitest.Text("Algo deu errado")) == nil {
		t.Fatal("a mensagem deveria estar na tela")
	}
	h.Key(juigo.KeyEnter)
	if h.Session().Overlay() != nil {
		t.Fatal("Enter no OK focado deveria dispensar o alerta")
	}
}

// TestPromptDigitaEEnvia: o campo do Prompt abre focado — digitar e Enter
// entregam o valor; Escape descarta sem chamar o callback.
func TestPromptDigitaEEnvia(t *testing.T) {
	h := uitest.New(t, juigo.NewVBox(juigo.NewText("raiz")), 400, 300)

	got := ""
	quick.Prompt("Seu nome?", "digite", func(v string) { got = v })
	h.Type("Ana çã") // sem clique nenhum: o autofoco já está no campo
	h.Key(juigo.KeyEnter)
	if got != "Ana çã" {
		t.Fatalf("Enter deveria entregar o valor digitado; got %q", got)
	}
	if h.Session().Overlay() != nil {
		t.Fatal("o Prompt deveria fechar após o envio")
	}

	chamado := false
	quick.Prompt("De novo?", "digite", func(string) { chamado = true })
	h.Type("descartado")
	h.Key(juigo.KeyEscape)
	if chamado {
		t.Fatal("Escape não deveria chamar o callback")
	}
}

// TestLabeledEButtons cobre a geometria dos helpers de layout: o campo do
// Labeled cresce além do rótulo e as ações do Buttons vão para a direita.
func TestLabeledEButtons(t *testing.T) {
	campo := juigo.NewInput("v")
	rotulo := juigo.NewText("Nome:")
	linha := quick.Labeled("ignorado", campo)
	_ = rotulo
	ok := juigo.NewButton("OK", nil)
	barra := quick.Buttons(ok)
	h := uitest.New(t, juigo.NewVBox(linha, barra), 400, 200)
	_ = h

	if campo.Bounds().Dx() <= campo.PreferredSize().X {
		t.Fatal("o campo do Labeled deveria crescer além da largura preferida")
	}
	meio := barra.Bounds().Min.X + barra.Bounds().Dx()/2
	if ok.Bounds().Min.X < meio {
		t.Fatalf("o botão do Buttons deveria estar na metade direita; Min.X = %d", ok.Bounds().Min.X)
	}
}
