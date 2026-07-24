package widget

import "github.com/JonathanSantos/JUIGo/internal/hooks"

// OpenOverlay exibe w como camada de SOBREPOSIÇÃO da aplicação: desenhada
// por cima da árvore normal e com prioridade nos eventos de mouse/rolagem.
// As camadas formam uma PILHA — abrir outra a empilha por cima (um popup
// aberto de dentro de um modal fica sobre ele), e as regras de fechamento
// valem para a camada do TOPO: um clique fora dos bounds dela a fecha
// (engolindo o clique), assim como Tab/foco fora dela — é a base de popups
// como o do Dropdown. Se w for focável, recebe o foco ao abrir; ao fechar,
// o foco de quando ELA abriu é restaurado e w recebe
// FocusEvent{Gained: false}. Fora de uma aplicação, é um no-op.
func OpenOverlay(w Widget) {
	if hooks.OpenOverlay != nil {
		hooks.OpenOverlay(w)
	}
}

// CloseOverlay remove w da pilha de sobreposição, se ele estiver nela,
// junto com as camadas abertas por cima dele. Fora de uma aplicação, é um
// no-op.
func CloseOverlay(w Widget) {
	if hooks.CloseOverlay != nil {
		hooks.CloseOverlay(w)
	}
}

// Focus move o foco de teclado para w programaticamente — um campo de
// edição recém-criado, por exemplo. Focar fora da overlay do topo a fecha
// (mesma regra do Tab), camada por camada até a que contém w. Fora de uma
// aplicação, é um no-op.
func Focus(w Widget) {
	hooks.RequestFocus(w)
}

// Blur limpa o foco de teclado programaticamente.
func Blur() {
	hooks.RequestFocus(nil)
}

// Contains informa se w pertence à árvore enraizada em root (inclusive se
// w == root).
func Contains(root, w Widget) bool {
	if root == nil || w == nil {
		return false
	}
	if root == w {
		return true
	}
	if p, ok := root.(ParentWidget); ok {
		for _, ch := range p.Children() {
			if Contains(ch, w) {
				return true
			}
		}
	}
	return false
}
