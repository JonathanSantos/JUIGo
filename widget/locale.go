package widget

// Locale reúne os textos que a própria biblioteca exibe — rótulos dos
// diálogos prontos, nomes de meses do Calendar, formato de data. O padrão é
// inglês (LocaleEN); aplicações em português chamam, uma vez, antes de
// montar a interface:
//
//	juigo.SetLocale(juigo.LocalePTBR())
//
// Para outro idioma, parta de um dos prontos e troque os campos. O locale é
// global ao processo (como o relógio e a área de transferência): defina-o
// na inicialização, na main thread.
type Locale struct {
	// OK, Cancel, Open e Save rotulam os botões dos diálogos de quick.
	OK, Cancel, Open, Save string
	// FileName e FileNamePlaceholder rotulam o campo de nome do diálogo de
	// salvar; FolderUp é a dica do botão que sobe uma pasta e OpenError o
	// prefixo da mensagem de erro de leitura de diretório.
	FileName, FileNamePlaceholder string
	FolderUp, OpenError           string
	// CommandPlaceholder é o texto do campo da paleta de comandos.
	CommandPlaceholder string
	// PickDate é a dica do botão que abre o calendário de quick.Date.
	PickDate string
	// Months são os nomes dos meses (janeiro primeiro) do cabeçalho do
	// Calendar; Weekdays, as iniciais dos dias (domingo primeiro).
	Months   [12]string
	Weekdays [7]string
	// DateLayout é o formato (pacote time) dos campos de data de quick;
	// DatePlaceholder é a máscara exibida no campo vazio.
	DateLayout, DatePlaceholder string
}

// LocaleEN devolve os textos em inglês — o padrão da biblioteca.
func LocaleEN() Locale {
	return Locale{
		OK: "OK", Cancel: "Cancel", Open: "Open", Save: "Save",
		FileName: "Name:", FileNamePlaceholder: "file name…",
		FolderUp: "Parent folder", OpenError: "Could not open: ",
		CommandPlaceholder: "Command…",
		PickDate:           "Calendar",
		Months: [12]string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
		Weekdays:   [7]string{"S", "M", "T", "W", "T", "F", "S"},
		DateLayout: "2006-01-02", DatePlaceholder: "YYYY-MM-DD",
	}
}

// LocalePTBR devolve os textos em português do Brasil.
func LocalePTBR() Locale {
	return Locale{
		OK: "OK", Cancel: "Cancelar", Open: "Abrir", Save: "Salvar",
		FileName: "Nome:", FileNamePlaceholder: "nome do arquivo…",
		FolderUp: "Pasta acima", OpenError: "Não deu para abrir: ",
		CommandPlaceholder: "Comando…",
		PickDate:           "Calendário",
		Months: [12]string{
			"janeiro", "fevereiro", "março", "abril", "maio", "junho",
			"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
		},
		Weekdays:   [7]string{"D", "S", "T", "Q", "Q", "S", "S"},
		DateLayout: "02/01/2006", DatePlaceholder: "DD/MM/AAAA",
	}
}

// locale é o conjunto de textos em vigor; localeGen muda a cada troca e
// invalida caches de texto derivados (ex.: o título do Calendar).
var (
	locale    = LocaleEN()
	localeGen uint64
)

// SetLocale troca os textos exibidos pela biblioteca e devolve os
// anteriores (útil para restaurar em testes). Widgets já montados passam a
// usar os novos textos no próximo redesenho; diálogos já abertos mantêm os
// rótulos com que foram criados.
func SetLocale(l Locale) (previous Locale) {
	previous = locale
	locale = l
	localeGen++
	return previous
}

// CurrentLocale devolve os textos em vigor.
func CurrentLocale() Locale {
	return locale
}
