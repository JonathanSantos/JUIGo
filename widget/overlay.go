package widget

import "juigo/internal/hooks"

// OpenOverlay exibe w como a camada de SOBREPOSIÇÃO da aplicação: desenhada
// por cima da árvore normal e com prioridade nos eventos de mouse/rolagem.
// Um clique fora dos bounds de w fecha a camada (engolindo o clique), assim
// como Tab/foco fora dela — é a base de popups como o do Dropdown. Se w for
// focável, recebe o foco ao abrir; ao fechar, o foco anterior é restaurado
// e w recebe FocusEvent{Gained: false}. Há uma única camada por vez: abrir
// outra substitui a atual. Fora de uma aplicação, é um no-op.
func OpenOverlay(w Widget) {
	if hooks.OpenOverlay != nil {
		hooks.OpenOverlay(w)
	}
}

// CloseOverlay remove w da camada de sobreposição, se ele for a camada
// atual. Fora de uma aplicação, é um no-op.
func CloseOverlay(w Widget) {
	if hooks.CloseOverlay != nil {
		hooks.CloseOverlay(w)
	}
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
