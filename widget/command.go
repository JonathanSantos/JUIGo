package widget

import (
	"strings"

	"github.com/JonathanSantos/JUIGo/event"
)

// Command é um comando GLOBAL da aplicação: um título (para menus e para a
// paleta de comandos), um atalho opcional e a ação. Registre com
// App.AddCommand (ou Session.AddCommand); os itens de um MenuBar são
// Commands e se registram sozinhos no mount.
//
//	app.AddCommand(juigo.Command{
//	    Title:  "Salvar",
//	    Key:    juigo.LetterKey('s'),
//	    Mods:   juigo.ModControl, // casa Ctrl E Cmd (a convenção da lib)
//	    Action: salvar,
//	})
//
// O atalho dispara quando o widget focado NÃO consome a tecla — um campo de
// texto continua dono do próprio teclado. Com um ou mais comandos
// registrados, Ctrl/Cmd+K abre a paleta de comandos embutida (busca pelo
// título, Enter executa) — a menos que você registre o seu próprio
// comando nesse atalho.
type Command struct {
	// Title é o nome exibido em menus e na paleta.
	Title string
	// Key e Mods formam o atalho global; KeyUnknown = sem atalho. No
	// casamento, ModControl e ModSuper são equivalentes (Ctrl no
	// Linux/Windows, Cmd no macOS).
	Key  event.Key
	Mods event.Modifiers
	// Action é executada ao acionar o comando (atalho, menu ou paleta).
	Action func()
}

// ShortcutLabel devolve o atalho para exibição ("Cmd+S"; vazio sem atalho).
// "Cmd" representa o modificador de comando — no casamento vale Ctrl também.
func (c Command) ShortcutLabel() string {
	if c.Key == event.KeyUnknown {
		return ""
	}
	var b strings.Builder
	if c.Mods.Command() {
		b.WriteString("Cmd+")
	}
	if c.Mods.Shift() {
		b.WriteString("Shift+")
	}
	if c.Mods.Alt() {
		b.WriteString("Alt+")
	}
	b.WriteString(c.Key.Label())
	return b.String()
}

// matches informa se o evento de tecla aciona o atalho do comando
// (ModControl≡ModSuper; Shift e Alt exatos).
func (c Command) matches(k event.Key, mods event.Modifiers) bool {
	if c.Key == event.KeyUnknown || c.Key != k {
		return false
	}
	return c.Mods.Command() == mods.Command() &&
		c.Mods.Shift() == mods.Shift() &&
		c.Mods.Alt() == mods.Alt()
}

// AddCommand registra (ou SUBSTITUI, pelo título) um comando global da
// sessão — a substituição torna o registro idempotente para quem registra a
// cada mount, como o MenuBar.
func (s *Session) AddCommand(c Command) {
	if c.Title == "" {
		return
	}
	for i := range s.commands {
		if s.commands[i].Title == c.Title {
			s.commands[i] = c
			return
		}
	}
	s.commands = append(s.commands, c)
}

// Commands devolve uma cópia dos comandos registrados, na ordem de
// registro (a paleta e menus de aplicação leem daqui).
func (s *Session) Commands() []Command {
	if len(s.commands) == 0 {
		return nil
	}
	out := make([]Command, len(s.commands))
	copy(out, s.commands)
	return out
}

// runCommand aciona o primeiro comando cujo atalho casa com a tecla.
// Devolve true se algum rodou.
func (s *Session) runCommand(k event.Key, mods event.Modifiers) bool {
	for i := range s.commands {
		if s.commands[i].matches(k, mods) {
			if a := s.commands[i].Action; a != nil {
				a()
			}
			return true
		}
	}
	return false
}
