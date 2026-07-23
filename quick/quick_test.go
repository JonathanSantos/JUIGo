package quick_test

import (
	"testing"

	"juigo"
	"juigo/quick"
	"juigo/uitest"
)

// TestFormFluxoDeValidacao cobre o ciclo completo do quick.Form com handles
// tipados: erros só aparecem depois do envio (ou do blur), preencher corrige
// ao vivo, o envio válido chama o callback lendo os valores pelos handles e
// Enter em um campo também envia.
func TestFormFluxoDeValidacao(t *testing.T) {
	nome := quick.Text("Nome:").Placeholder("nome").
		Required("Informe o nome").Min(3, "Mínimo de 3 caracteres")
	mail := quick.Text("E-mail:").Placeholder("mail").Email("E-mail inválido")
	termos := quick.Check("Aceito os termos").Required("Aceite os termos")

	salvos := 0
	salvo := ""
	f := quick.Form(nome, mail, termos).Submit("Salvar", func() {
		salvos++
		salvo = nome.Value()
	})
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
	if !termos.Value() {
		t.Fatal("o clique deveria refletir no handle do checkbox")
	}

	// Envio válido chama o callback com os valores dos handles.
	h.Click(uitest.Text("Salvar"))
	if salvos != 1 {
		t.Fatalf("envio válido deveria chamar o callback; salvos = %d", salvos)
	}
	if salvo != "Jonatan" || mail.Value() != "jo@exemplo.com" {
		t.Fatalf("handles deveriam entregar os valores digitados; nome=%q mail=%q", salvo, mail.Value())
	}

	// Enter em um campo de linha única também envia.
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyEnter)
	if salvos != 2 {
		t.Fatalf("Enter no campo deveria enviar; salvos = %d", salvos)
	}
}

// TestNumberFiltraEValida: o campo numérico ignora letras, entrega int pelo
// handle, cai no valor inicial quando vazio e valida a faixa.
func TestNumberFiltraEValida(t *testing.T) {
	idade := quick.Number("Idade:", 18).Min(0, "Idade negativa").Max(120, "Confere a idade?")
	f := quick.Form(idade).Submit("OK", nil)
	h := uitest.New(t, f, 420, 240)

	campo := h.Find(uitest.OfType[*juigo.Input]())
	if campo == nil {
		t.Fatal("Number deveria renderizar um Input")
	}
	in := campo.(*juigo.Input)
	if in.Text() != "18" {
		t.Fatalf("o campo deveria nascer com o valor inicial; got %q", in.Text())
	}

	// Letras são filtradas; dígitos entram e o handle devolve int.
	h.Click(uitest.OfType[*juigo.Input]())
	h.Type("abc9")
	if in.Text() != "189" {
		t.Fatalf("filtro deveria descartar letras; got %q", in.Text())
	}
	if idade.Value() != 189 {
		t.Fatalf("Value deveria acompanhar o texto; got %d", idade.Value())
	}

	// Fora da faixa: envio revela o erro.
	h.Click(uitest.Text("OK"))
	if h.Find(uitest.Text("Confere a idade?")) == nil {
		t.Fatal("envio deveria revelar o erro de faixa")
	}

	// Vazio vale o valor inicial (e o erro some).
	h.Click(uitest.OfType[*juigo.Input]())
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Key(juigo.KeyBackspace)
	if in.Text() != "" {
		t.Fatalf("o campo deveria estar vazio; got %q", in.Text())
	}
	if idade.Value() != 18 {
		t.Fatalf("vazio deveria valer o inicial; got %d", idade.Value())
	}
	if h.Find(uitest.Text("Confere a idade?")) != nil {
		t.Fatal("com o valor de volta à faixa, o erro deveria sumir")
	}

	// Set externo pelo handle reflete no texto do campo.
	idade.Set(65)
	if in.Text() != "65" {
		t.Fatalf("Set no handle deveria atualizar o campo; got %q", in.Text())
	}
}

// TestBindTrocaOEstado: Bind liga o campo a um State externo — digitação
// atualiza o State e um Set externo atualiza o campo.
func TestBindTrocaOEstado(t *testing.T) {
	externo := juigo.NewState("inicial")
	nome := quick.Text("Nome:").Placeholder("nome").Bind(externo)
	f := quick.Form(nome)
	h := uitest.New(t, f, 420, 200)

	if nome.Value() != "inicial" {
		t.Fatalf("o handle deveria ler o State externo; got %q", nome.Value())
	}
	h.Click(uitest.Placeholder("nome"))
	h.Key(juigo.KeyA, juigo.ModControl)
	h.Type("editado")
	if externo.Get() != "editado" {
		t.Fatalf("digitação deveria atualizar o State externo; got %q", externo.Get())
	}
	externo.Set("de fora")
	if h.Find(uitest.Text("de fora")) == nil && nome.Value() != "de fora" {
		t.Fatal("Set externo deveria refletir no campo")
	}
}

// TestSectionDivideEmGrades: Section fecha a grade corrente, mostra o
// título e abre outra — os campos dos dois lados continuam funcionais.
func TestSectionDivideEmGrades(t *testing.T) {
	nome := quick.Text("Nome:").Placeholder("nome")
	rua := quick.Text("Rua:").Placeholder("rua")
	f := quick.Form(nome, quick.Section("Endereço"), rua).Gap(8).Pad(4)
	h := uitest.New(t, f, 420, 260)

	if h.Find(uitest.Text("Endereço")) == nil {
		t.Fatal("o título da seção deveria estar na tela")
	}
	grades := 0
	for _, ch := range f.Children() {
		if _, ok := ch.(*juigo.Grid); ok {
			grades++
		}
	}
	if grades != 2 {
		t.Fatalf("a seção deveria dividir em 2 grades; got %d", grades)
	}
	h.Click(uitest.Placeholder("rua"))
	h.Type("Av. Central")
	if rua.Value() != "Av. Central" {
		t.Fatalf("campo pós-seção deveria funcionar; got %q", rua.Value())
	}
}

// TestFormOptionsAdotaPrimeira: com o valor vazio e sem placeholder, o
// Dropdown e o handle concordam na primeira opção desde o início.
func TestFormOptionsAdotaPrimeira(t *testing.T) {
	plano := quick.Options("Plano:", "free", "pro")
	f := quick.Form(plano)
	uitest.New(t, f, 400, 200)
	if plano.Value() != "free" {
		t.Fatalf("handle deveria adotar a primeira opção; got %q", plano.Value())
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
	linha := quick.Labeled("Nome:", campo)
	ok := juigo.NewButton("OK", nil)
	barra := quick.Buttons(ok)
	uitest.New(t, juigo.NewVBox(linha, barra), 400, 200)

	if campo.Bounds().Dx() <= campo.PreferredSize().X {
		t.Fatal("o campo do Labeled deveria crescer além da largura preferida")
	}
	meio := barra.Bounds().Min.X + barra.Bounds().Dx()/2
	if ok.Bounds().Min.X < meio {
		t.Fatalf("o botão do Buttons deveria estar na metade direita; Min.X = %d", ok.Bounds().Min.X)
	}
}
