package widget

import (
	"image"
	"time"

	"github.com/JonathanSantos/JUIGo/internal/hooks"
)

// ShowToast exibe um aviso transitório na base da janela — confirmações
// leves ("Contato salvo") que não pedem interação. O toast é uma camada
// PASSIVA, como o tooltip: fora do hit-test e da cadeia de foco, some
// sozinho após d (d <= 0 usa Theme.ToastDuration) e um novo toast substitui
// o atual. Fora de uma aplicação, é um no-op. Ver também quick.Toast.
func ShowToast(text string, d time.Duration) {
	hooks.ShowToast(text, d)
}

// HideToast esconde o toast atual, se houver. Fora de uma aplicação, é um
// no-op.
func HideToast() {
	hooks.ShowToast("", 0)
}

// ShowToast exibe o toast na sessão (ver a função de pacote ShowToast).
// Texto vazio esconde o atual.
func (s *Session) ShowToast(text string, d time.Duration) {
	if s.theme == nil || text == "" {
		s.HideToast()
		return
	}
	if s.toastCancel != nil {
		s.toastCancel()
		s.toastCancel = nil
	}
	if s.toastView == nil {
		s.toastView = NewTooltipView()
	}
	Mount(s.toastView, s.theme)
	s.toastView.SetText(text)
	s.toastShown = true
	// Posiciona já: o diff de bounds do Layout danifica posição antiga e
	// nova; o AddDamage cobre o caso de reexibir no mesmo lugar.
	s.layoutToast(image.Rectangle{Max: s.size})
	s.AddDamage(s.toastView.Bounds())
	if d <= 0 {
		d = s.theme.ToastDuration
	}
	if d > 0 {
		s.toastCancel = hooks.ScheduleAfter(d, func() {
			s.toastCancel = nil
			s.HideToast()
		})
	}
}

// HideToast esconde o toast atual e cancela o sumiço agendado.
func (s *Session) HideToast() {
	if s.toastCancel != nil {
		s.toastCancel()
		s.toastCancel = nil
	}
	if !s.toastShown {
		return
	}
	s.toastShown = false
	if s.toastView != nil {
		s.AddDamage(s.toastView.Bounds())
	}
}

// ToastVisible informa se há um toast na tela.
func (s *Session) ToastVisible() bool {
	return s.toastShown
}

// layoutToast ancora o toast no centro da base da área dada, com um respiro
// do tema. Chamado ao exibir e a cada Render — assim ele acompanha resize e
// troca de escala, e o diff de bounds gera o dano do movimento.
func (s *Session) layoutToast(b image.Rectangle) {
	pref := s.toastView.PreferredSize()
	pad := s.theme.PaddingPx()
	pos := image.Pt(b.Min.X+(b.Dx()-pref.X)/2, b.Max.Y-pref.Y-2*pad)
	if pos.X < b.Min.X {
		pos.X = b.Min.X
	}
	if pos.Y < b.Min.Y {
		pos.Y = b.Min.Y
	}
	s.toastView.Layout(image.Rectangle{Min: pos, Max: pos.Add(pref)})
}
