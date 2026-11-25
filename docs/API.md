# Convenções da API

Como se escreve — e como se estende — a API do JUIGo. Tudo o que é exportado
segue estas regras; um widget novo que as respeite já "soa" como o resto da
biblioteca. Quem está migrando de uma versão anterior encontra o passo a
passo em [MIGRACAO-v0.19.md](MIGRACAO-v0.19.md).

## 1. Uma interface é uma expressão

Construtor com o essencial, o resto por métodos encadeáveis que devolvem o
tipo concreto:

```go
juigo.NewVBox(
	juigo.NewText("Cadastro").Subtitle(),
	juigo.NewInput("E-mail").BindValue(mail).OnSubmit(enviar).Grow(1),
	juigo.NewButton("Salvar", salvar).
		Tooltip("Grava o cadastro").
		BindDisabled(invalido),
).Pad(16).Gap(8)
```

- **Construtor** — `NewX(conteúdo essencial)`. Nada opcional entra aqui.
- **Identificadores em inglês**, godoc e comentários em português.

## 2. Três famílias de método

| Família | Forma | Exemplos |
|---|---|---|
| **Propriedade** (dá para ler de volta) | `X()` lê, `SetX(v)` escreve e redesenha | `Text()`/`SetText`, `Label()`/`SetLabel`, `Value()`/`SetValue`, `Checked()`/`SetChecked`, `Placeholder()`/`SetPlaceholder`, `Ratio()`/`SetRatio`, `ReadOnly()`/`SetReadOnly`, `Options()`/`SetOptions`, `Min()`/`Max()`/`SetRange` |
| **Opção** (só configura; não há getter) | verbo ou adjetivo, sem `Set` | `Pad`, `Gap`, `Step`, `MaxWidth`, `Center`, `Secondary`, `Ghost`, `Vertical`, `Horizontal`, `WrapLines(on)`, `CloseOnBackdrop(on)`, `StickToEnd(on)` |
| **Ação** (faz algo agora) | verbo, sem retorno | `Show`, `Close`, `Push`, `Pop`, `Select`, `Clear`, `Refresh`, `ScrollTo`, `ScrollToEnd` |

Propriedades e opções **devolvem o receptor** (encadeáveis). Ações não.

- Getter booleano é o adjetivo puro: `Disabled()`, `Loading()`, `Shown()`,
  `ReadOnly()`. Sem prefixo `Is`.
- Variantes de aparência não levam argumento (`Secondary()`, `Title()`);
  liga/desliga de comportamento leva `bool` (`WrapLines(true)`).
- Todo setter **redesenha sozinho**. Não há campo público de estado em
  widget: mudar algo é sempre por método.

## 3. Callbacks

- Registrados por `OnX(fn)`, encadeável.
- **`OnChange` é "o usuário mudou o valor"** — sempre com o valor novo:
  `func(string)` em Input/TextArea/Dropdown/Radio, `func(bool)` no Checkbox,
  `func(float64)` no Slider, `func(int)` em Tabs/List/Table,
  `func(time.Time)` no Calendar, `func(ID)` na Tree.
  Exceções documentadas: `CodeEditor.OnChange` e `Navigator.OnChange` não
  levam valor (montá-lo custaria O(n) por tecla / não há valor único).
- **Setter programático nunca dispara callback.** `SetText`, `SetValue`,
  `SetChecked`, `Select` mudam o widget (e o State vinculado) em silêncio;
  `OnChange` é reservado às mudanças vindas do usuário.
- Outros: `OnClick` (Button), `OnSubmit(func(text string))` (Enter no
  Input), `OnActivate` (duplo clique/Enter na Tree), `OnClose` (Modal),
  `OnReorder` (List/Table), `OnDone` (anim.Animation).
- **`OnFocus`/`OnBlur` existem em todo widget focável** (Button, Checkbox,
  CodeEditor, Dropdown, Input, Radio, Slider, SplitPane, Tabs, TextArea,
  Tree).

## 4. Reatividade

Cada vínculo tem o seu par de leitura/escrita:

| Vínculo | Lê / escreve | Onde |
|---|---|---|
| `BindText` | `Text()` / `SetText` | Text, Label (uma via) |
| `BindValue` | `Value()` / `SetValue` (`Text()`/`SetText` nos campos de texto) | Input, TextArea, Slider, ProgressBar, Dropdown, Radio, Calendar |
| `BindChecked` | `Checked()` / `SetChecked` | Checkbox |
| `BindSelected` | `Selected()` / `Select` | Tabs, List, Table, Tree |
| `BindRatio` | `Ratio()` / `SetRatio` | SplitPane |
| `BindOpen` | `Shown()` / `Show`, `Close` | Modal |
| `BindDisabled` | `Disabled()` / `SetDisabled` | todo widget |
| `BindInvalid`, `BindLoading`, `BindData` | `SetInvalid`, `SetLoading`, `SetData` | Input/TextArea, Button, gráficos |

- `juigo.BindChoice(dropdown, itens, rótulo, state)` liga um Dropdown a um
  `State[T]` de qualquer tipo comparável.
- List, Table e Tree também funcionam **sem State**: `OnChange(fn)` (ou
  `Selectable()`) habilita a seleção e `Selected()` a lê.
- **Vínculos não prendem o widget.** Todo `Bind*` assina o State com
  referência fraca (`state.WatchWeak`): um widget descartado numa
  reconstrução (`Clear` + `Add`) é coletado e a assinatura some sozinha.
- `State.Watch(fn)` devolve a função que cancela a assinatura. Ignorá-la é
  o certo para observadores que vivem tanto quanto o State.
- `Map` e `Combine` são mantidos vivos pela fonte: crie derivados junto dos
  estados, uma vez — não a cada reconstrução de interface.

## 5. Layout e os encadeáveis comuns

Todo widget da biblioteca tem, com o próprio tipo preservado:

`Grow(peso)` · `Centered()` · `AtStart()` · `AtEnd()` · `Tooltip(texto)` ·
`SetDisabled(v)` · `BindDisabled(state)`

Eles são **gerados** (`internal/gen/chain`, `go generate ./...`) — Go não
deixa um método do `BaseWidget` embutido devolver o tipo que o embute. Um
teste do pacote `widget` falha se um widget novo ficar de fora da lista.

**Widgets próprios** — um tipo seu que embute `juigo.BaseWidget`, ou que
*decora* um widget pronto embutindo `*juigo.Table`, `*juigo.VBox`… — usam as
funções de mesmo nome, que preservam o **seu** tipo:

```go
juigo.Grow(minhaColuna, 1)          // certo: entra a coluna na árvore
minhaColuna.Grow(1)                 // ERRADO se coluna embute *juigo.VBox:
                                    // o método promovido devolve o *VBox de
                                    // dentro, e é ele que entra na árvore
```

## 6. Unidades

- **Unidades lógicas** (escaladas pelo tema/HiDPI): tudo o que se declara —
  `Pad`, `Gap`, `NewSized(w, h)`, `MaxWidth`, `SplitPane.Min`,
  `Table.Widths`, campos do `Theme`.
- **Pixels**: tudo o que é geometria em execução — `Bounds()`, `Layout`,
  eventos, `Scroll.Offset()`/`ScrollTo`, `Popup.ShowAt`, `Table.RowRect`.
- **Pontos tipográficos**: `Theme.FontSize`, `CodeEditor.FontSize`.
- Índices de texto são sempre em **runes**.

## 7. Textos da própria biblioteca

Rótulos de diálogos, meses do Calendar e formato de data vêm de um `Locale`
(inglês por padrão):

```go
juigo.SetLocale(juigo.LocalePTBR()) // uma vez, antes de montar a interface
```

## 8. Pacotes

- **`juigo`** (raiz) — fachada: App, janelas, todos os widgets, State, tema,
  eventos. É o único import de uma aplicação comum.
- **Camadas opcionais**, importadas pelo nome quando usadas: `quick`
  (formulários e diálogos prontos), `form` (validação), `anim`, `chart`,
  `syntax`, `uitest`, `offscreen`.
- `widget`, `theme`, `state`, `event`, `render` — os subpacotes por trás da
  fachada; importe direto para construir shells ou widgets próprios.

## 9. Regras de implementação

- `Draw`, `Layout` e `PreferredSize` **não alocam** com os caches aquecidos.
  `Children()` devolve fatia reutilizada.
- Medida de texto passa por `Theme.MeasureString` (memorizada por string);
  posição de cursor e clique→índice usam a tabela `render.PrefixWidths`.
- Nenhuma cor ou tamanho fixo no widget: tudo vem do `Theme`.
- Sem `panic` em código de biblioteca; consultas antes do mount devolvem o
  valor zero.
- Timers próprios (piscada, spinner) param quando o widget sai da árvore.
