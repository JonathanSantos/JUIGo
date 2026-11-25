# Migrando para a v0.20

A v0.20 troca a identidade visual: o tema padrão passa a ser o **Nanquim**
(tinta, neutros puros e um sinal âmbar; ver [DESIGN.md](DESIGN.md)) e os
temas nascem de sementes (`NewTheme(Palette{…})`). O compilador aponta as
remoções; os itens marcados com ⚠️ mudam a **aparência** sem erro de
compilação.

## Removidos

| Antes | Agora |
|---|---|
| `juigo.ClaudeTheme()`, `theme.Claude()` | `juigo.DefaultTheme()` — ou um tema seu com `NewTheme` |
| `juigo.ClaudeDarkTheme()`, `theme.ClaudeDark()` | `juigo.DarkTheme()` |
| `juigo.LoraFont()`, `juigo.LoraBoldFont()`, `theme.Lora()`, `theme.LoraBold()` | `juigo.InterSemiBoldFont()`, `juigo.GoBoldFont()` ou `juigo.GoMonoBoldFont()`; para uma serif, carregue o `.ttf` com `opentype.Parse` e passe a `Theme.UseDisplayFont` |

Quem gostava do "papel e tinta" chega perto com sementes:

```go
th, _ := juigo.NewTheme(juigo.Palette{
	Primary: color.RGBA{R: 0xD9, G: 0x77, B: 0x57, A: 0xFF},
	Neutral: color.RGBA{R: 0xB0, G: 0xA0, B: 0x80, A: 0xFF},
})
```

## Mudanças de aparência

- ⚠️ **`DefaultTheme`/`DarkTheme` mudaram de paleta e de métricas:** tinta
  em vez de azul, `Padding` 10 (era 8) e a fonte passou a ser a **Inter**
  (corpo Regular, títulos SemiBold; eram Go Regular e Go Bold). Textos
  medem diferente: confira larguras fixas. Layouts ficam um pouco mais espaçosos. Para o visual antigo, use
  `ClassicTheme()` (claro) ou o novo `ClassicDarkTheme()` (escuro).
- ⚠️ **`ClassicTheme` não é mais "Default com cantos retos"**: ele guarda
  a paleta antiga inteira. Os widgets, porém, ganharam os ajustes de
  desenho abaixo em todos os temas, então não é mais idêntico pixel a pixel
  à v0.19.
- ⚠️ **Checkbox marcado** agora é a caixa cheia de `Accent` com um check
  em `AccentText` (antes, um quadrado interno).
- ⚠️ **Listas selecionáveis** recuam o conteúdo da linha para dentro da
  pílula de realce (`RowPad` + `Padding`). Linhas que posicionavam algo
  pela borda da lista ficam deslocadas para dentro.
- Trilho de `Slider`/`ProgressBar` agora usa `Theme.Track`; cabeçalho da
  `Table` e calha do `CodeEditor`, `Theme.HeaderBackground`; linhas da
  `Table`, o fio leve `SurfaceBorder`.

## Novidades

- `juigo.Palette`, `juigo.NewTheme`, `juigo.NanquimPalette`,
  `Theme.ApplyPalette`: temas a partir de sementes, com contraste WCAG
  garantido.
- `Theme.Audit()` e `juigo.Contrast(a, b)`: conferência de contraste.
- `Theme.UseBodyFont` troca a fonte do corpo; `theme.Inter()`,
  `theme.InterSemiBold()` e `theme.GoRegular()` dão as embutidas.
- Tokens novos: `Success`, `Warning`, `Track`, `HeaderBackground`,
  `AccentText`. Se você monta um `Theme` campo a campo, preencha-os (ou
  derive com `ApplyPalette` e sobrescreva o que quiser).
- `Text.Success()`/`Warning()` e `Label.Success()`/`Warning()`.
