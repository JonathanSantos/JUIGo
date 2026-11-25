# Migrando para a v0.19

A v0.19 padroniza a API (ver [API.md](API.md)). O compilador aponta quase
tudo: troque conforme a tabela e recompile. Os dois itens marcados com ⚠️
mudam **comportamento** sem erro de compilação — confira-os à mão.

## Renomeações

| Antes | Agora |
|---|---|
| `btn.Label`, `check.Label`, `radio.Label` (campos) | `Label()` / `SetLabel(s)` |
| `radio.Value` (campo) | `Value()` |
| `input.Placeholder`, `area.Placeholder` (campos) | `Placeholder()` / `SetPlaceholder(s)` |
| `dropdown.Placeholder(s)` | `dropdown.SetPlaceholder(s)` |
| `dropdown.Options` (campo) | `Options()` / `SetOptions(...)` |
| `dropdown.SelectedIndex()` | `dropdown.Selected()` |
| `slider.Min`, `slider.Max`, `bar.Min`, `bar.Max` (campos) | `Min()`, `Max()`, `SetRange(min, max)` |
| `text.Align` (campo) | `text.Align(a)` — ou `Left()`, `Center()`, `Right()` |
| `box.Padding`, `box.Spacing` (campos) | `Pad(n)`, `Gap(n)` |
| `split.Ratio(f)` | `split.SetRatio(f)` |
| `split.CurrentRatio()` | `split.Ratio()` |
| `editor.ReadOnly(on)` | `editor.SetReadOnly(on)` |
| `editor.IsReadOnly()` | `editor.ReadOnly()` |
| `calendar.OnPick(fn)` | `calendar.OnChange(fn)` |
| `calendar.Selected()` | `calendar.Value()` |
| `animation.OnDone = fn` (campo) | `animation.OnDone(fn)` |
| `quick.Text(…).Disabled(state)` (e demais campos do quick) | `.BindDisabled(state)` |

## Assinaturas

| Antes | Agora |
|---|---|
| `input.OnSubmit(func())` | `input.OnSubmit(func(text string))` |
| `state.Watch(fn)` sem retorno | devolve `cancel func()` (ignorar é válido) |
| `Observable.WatchChange(fn)` sem retorno | devolve `cancel func()` — quem implementa a interface precisa acompanhar |

Setters que não devolviam nada agora devolvem o receptor (`SetText`,
`SetValue`, `SetChecked`, `SetInvalid`, `SetLoading`, `SetCount`,
`SetCursor`, `SetImage`, `SetData`). Código existente segue compilando.

## ⚠️ Mudanças de comportamento

1. **`Tabs.Select(i)` não dispara mais `OnChange`.** Vale a regra geral:
   setter programático é silencioso; `OnChange` só reage ao usuário. Se você
   dependia do callback numa troca feita por código, chame a função
   diretamente ou observe o State de `BindSelected`.
2. **Os textos da biblioteca agora são inglês por padrão, inclusive o
   Calendar e o formato de data do `quick.Date`** (`YYYY-MM-DD`). Para
   manter português (`DD/MM/AAAA`, meses em português, "Cancelar"):

   ```go
   juigo.SetLocale(juigo.LocalePTBR())
   ```

Também mudou, sem exigir ação:

- Um widget focado que sai da árvore (removido, ou numa aba que deixou de
  ser a ativa) **perde o foco** no frame seguinte.
- `Bind*` não mantém mais o widget vivo: widgets descartados são coletados.
- O `Navigator` libera os retratos da transição quando ela termina.

## O que dá para simplificar

Nada disto é obrigatório — as formas antigas continuam valendo.

```go
// antes
juigo.BindDisabled(juigo.Tooltip(juigo.NewButton("Enviar", enviar), "Envia"), vazio)
juigo.Grow(juigo.NewScroll(lista), 1)

// agora
juigo.NewButton("Enviar", enviar).Tooltip("Envia").BindDisabled(vazio)
juigo.NewScroll(lista).Grow(1)
```

| Em vez de | Use |
|---|---|
| `NewList(n, func() *Text {…}, func(t *Text, i int) {…})` | `NewTextList(n, func(i int) string {…})` |
| `BindSelected(state)` + `state.Watch(fn)` só para reagir | `OnChange(fn)` e `Selected()` |
| `scroll.ScrollTo(1 << 30)` | `scroll.ScrollToEnd()` ou `StickToEnd(true)` |
| `strconv` em volta de um Dropdown | `juigo.BindChoice(dropdown, itens, rótulo, state)` |
| guardar o `*Modal` para chamar `Show` | `modal.BindOpen(state)` |
| `campo.Text()` dentro do `OnSubmit` | o parâmetro `text` do callback |

**Atenção ao decorar widgets prontos**: num tipo seu que embute
`*juigo.VBox` ou `*juigo.Table`, use `juigo.Grow(w, 1)` (função), não
`w.Grow(1)` — o método promovido devolve o widget de dentro. Detalhes em
[API.md](API.md#5-layout-e-os-encadeáveis-comuns).
