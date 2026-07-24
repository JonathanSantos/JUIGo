// Navegação — o exemplo do Navigator do JUIGo: uma pilha de telas com
// transições animadas prontas. Início empilha Perfil (desliza da direita) e
// Ajuda (sobe como uma folha — e o Pop reverte sozinho: ela desce); dentro
// de Perfil, Preferências vai mais fundo e "Concluir" volta à raiz numa
// transição só (PopToRoot). O dropdown do Início troca a transição padrão
// do Push em runtime, e o campo de nome do Perfil prova que telas dormentes
// preservam o estado — vá e volte que o texto continua lá.
package main

import (
	"log"

	"github.com/JonathanSantos/JUIGo"
)

func main() {
	if err := juigo.Run("Navegação — JUIGo", 480, 360, ui()); err != nil {
		log.Fatal(err)
	}
}

// ui monta o navegador com a tela inicial já empilhada (a primeira tela
// entra sem animação).
func ui() *juigo.Navigator {
	nav := juigo.NewNavigator()
	nav.Push(telaInicio(nav))
	return nav
}

// barra é o cabeçalho das telas empilhadas: Voltar + título. É composição
// pura — não há widget de "app bar"; cinco linhas resolvem.
func barra(nav *juigo.Navigator, titulo string) juigo.Widget {
	return juigo.NewHBox(
		juigo.NewButton("← Voltar", func() { nav.Pop() }),
		juigo.Centered(juigo.NewText(titulo)),
	).Gap(12)
}

// telaInicio é a raiz da pilha: navega para as demais e escolhe a transição
// padrão do Push em runtime.
func telaInicio(nav *juigo.Navigator) juigo.Widget {
	// Uma instância só de Perfil: ir e voltar preserva os campos digitados.
	perfil := telaPerfil(nav)

	transicao := juigo.NewState("Deslizar")
	transicao.Watch(func(v string) {
		switch v {
		case "Deslizar":
			nav.Transition(juigo.TransitionSlideLeft)
		case "Esmaecer":
			nav.Transition(juigo.TransitionFade)
		case "Subir":
			nav.Transition(juigo.TransitionSlideUp)
		case "Corte seco":
			nav.Transition(juigo.TransitionNone)
		}
	})

	return juigo.NewVBox(
		juigo.NewText("Navegação em JUIGo"),
		juigo.NewText("Push empilha telas; Pop reverte a transição de entrada."),
		juigo.NewHBox(
			juigo.NewText("Transição do Push:"),
			juigo.NewDropdown("Deslizar", "Esmaecer", "Subir", "Corte seco").BindValue(transicao),
		).Gap(8),
		juigo.NewButton("Ver perfil →", func() { nav.Push(perfil) }),
		juigo.NewButton("Ajuda", func() { nav.Push(telaAjuda(nav), juigo.TransitionSlideUp) }),
	).Pad(16).Gap(12)
}

// telaPerfil demonstra estado preservado na pilha e navegação em cadeia.
func telaPerfil(nav *juigo.Navigator) juigo.Widget {
	nome := juigo.NewState("")
	saudacao := juigo.Map(nome, func(s string) string {
		if s == "" {
			return "Digite seu nome — telas dormentes preservam o estado."
		}
		return "Olá, " + s + "! Vá e volte: o campo continua preenchido."
	})
	return juigo.NewVBox(
		barra(nav, "Perfil"),
		juigo.NewInput("seu nome…").BindValue(nome),
		juigo.NewText("").BindText(saudacao),
		juigo.NewButton("Preferências →", func() { nav.Push(telaPrefs(nav)) }),
	).Pad(16).Gap(12)
}

// telaPrefs é o fundo da pilha de três níveis; Concluir volta direto à raiz.
func telaPrefs(nav *juigo.Navigator) juigo.Widget {
	return juigo.NewVBox(
		barra(nav, "Preferências"),
		juigo.NewCheckbox("Notificações"),
		juigo.NewCheckbox("Sons"),
		juigo.NewButton("Concluir", func() { nav.PopToRoot() }),
	).Pad(16).Gap(12)
}

// telaAjuda entra por baixo (TransitionSlideUp) e sai descendo — o Pop
// reverte a transição de entrada sem que a tela precise saber qual foi.
func telaAjuda(nav *juigo.Navigator) juigo.Widget {
	return juigo.NewVBox(
		juigo.NewText("Central de ajuda"),
		juigo.NewText("Esta tela subiu com TransitionSlideUp."),
		juigo.NewText("Feche e veja o Pop revertê-la: ela desce."),
		juigo.NewButton("Fechar", func() { nav.Pop() }),
	).Pad(16).Gap(12)
}
