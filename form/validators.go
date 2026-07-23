package form

import (
	"regexp"
	"strings"
)

// Required exige valor não-vazio (espaços não contam).
func Required(msg string) Validator {
	return func(v string) string {
		if strings.TrimSpace(v) == "" {
			return msg
		}
		return ""
	}
}

// MinRunes exige pelo menos n runes (acentos contam como 1).
func MinRunes(n int, msg string) Validator {
	return func(v string) string {
		if v != "" && len([]rune(v)) < n {
			return msg
		}
		return ""
	}
}

// MaxRunes exige no máximo n runes.
func MaxRunes(n int, msg string) Validator {
	return func(v string) string {
		if len([]rune(v)) > n {
			return msg
		}
		return ""
	}
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email exige o formato básico local@domínio.tld. Campos vazios passam —
// combine com Required para obrigatoriedade.
func Email(msg string) Validator {
	return Pattern(emailRe, msg)
}

// Pattern exige que o valor case com a expressão regular. Campos vazios
// passam — combine com Required para obrigatoriedade.
func Pattern(re *regexp.Regexp, msg string) Validator {
	return func(v string) string {
		if v != "" && !re.MatchString(v) {
			return msg
		}
		return ""
	}
}
