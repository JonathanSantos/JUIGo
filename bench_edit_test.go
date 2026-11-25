package juigo_test

import (
	"image"
	"strings"
	"testing"

	"github.com/JonathanSantos/JUIGo"
)

// BenchmarkInputArrastoSelecao mede o clique→índice do Input durante uma
// seleção por arraste num texto longo (1000 caracteres): cada movimento do
// mouse precisa descobrir a rune sob o cursor.
func BenchmarkInputArrastoSelecao(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatalf("DefaultTheme: %v", err)
	}
	campo := juigo.NewInput("")
	juigo.Mount(campo, th)
	campo.Layout(image.Rect(0, 0, 20000, 40))
	campo.SetText(strings.Repeat("texto longo ", 84)[:1000])
	campo.HandleEvent(juigo.FocusEvent{Gained: true})
	campo.HandleEvent(juigo.MouseEvent{Kind: juigo.MouseDown, Pos: image.Pt(20, 20)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := 2000 + i%3000
		campo.HandleEvent(juigo.MouseEvent{Kind: juigo.MouseMove, Pos: image.Pt(x, 20)})
	}
}

// BenchmarkTextAreaCursor mede um movimento de cursor (seta) numa TextArea
// de 250 linhas — não muda o texto, então não deveria reconstruir nada.
func BenchmarkTextAreaCursor(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatalf("DefaultTheme: %v", err)
	}
	area := juigo.NewTextArea("")
	juigo.Mount(area, th)
	area.Layout(image.Rect(0, 0, 600, 400))
	area.SetText(strings.Repeat("uma linha de texto comum\n", 250))
	area.HandleEvent(juigo.FocusEvent{Gained: true})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		area.HandleEvent(juigo.KeyEvent{Key: juigo.KeyLeft})
		area.HandleEvent(juigo.KeyEvent{Key: juigo.KeyRight})
	}
}

// BenchmarkDropdownPreferredSize mede o tamanho preferido de um Dropdown
// com 50 opções — consultado pelo layout a cada frame.
func BenchmarkDropdownPreferredSize(b *testing.B) {
	th, err := juigo.DefaultTheme()
	if err != nil {
		b.Fatalf("DefaultTheme: %v", err)
	}
	opts := make([]string, 50)
	for i := range opts {
		opts[i] = strings.Repeat("opção ", 1+i%4) + string(rune('A'+i%26))
	}
	d := juigo.NewDropdown(opts...)
	juigo.Mount(d, th)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.PreferredSize()
	}
}
