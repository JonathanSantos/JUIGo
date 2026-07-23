# 7GUIs — gaps encontrados e RESOLVIDOS

O exercício dos 7GUIs foi feito em duas rodadas: a primeira implementou as
sete GUIs até onde a lib alcançava e registrou aqui cada limitação; a
segunda transformou o registro em componentes da lib e reescreveu os
exemplos por cima deles. Estado atual: **todos os gaps acionáveis foram
resolvidos** — este arquivo vira o changelog do que a lib ganhou e de onde
cada peça é demonstrada.

## O que a lib ganhou (e onde ver funcionando)

| Componente | O que é | Demonstrado em |
| --- | --- | --- |
| `quick.Date` | Campo de data DD/MM/AAAA: máscara por `Input.Filter`, valor `time.Time`, `Required` e regra multi-fonte `.Rule` | Flight Booker |
| `Field.Disabled(state)` | Desabilitado reativo em qualquer campo do quick | Flight Booker (volta presa ao "só ida") |
| `Input/TextArea.BindInvalid` | Borda `Theme.Danger` reativa; o quick liga sozinho ao `ErrorOf` do campo | Flight Booker (datas inválidas ficam vermelhas) |
| `juigo.After` / `juigo.Every` | Timers públicos na main thread (relógio virtual no uitest) | `uitest/timers_test.go` |
| `uitest.NewLazy` | Harness que constrói a raiz DEPOIS de ligar os hooks (UIs que agendam timers na construção) | Timer |
| `Table` | Tabela de texto com colunas, cabeçalho FIXO na viewport, virtualização de desenho e seleção | CRUD |
| `List.BindSelected` / `Table.BindSelected` | Seleção de linha como State (clique seleciona, Set externo move o realce, `Theme.Selection` de fundo) | CRUD |
| `Popup` | Painel ancorado num ponto, sem escurecer o fundo — a base de menus de contexto | Circle Drawer (ajuste de diâmetro no ponto do clique) |
| `uitest.RightClickAt/RightClick` | Clique direito sintético | Circle Drawer |
| `render.StrokeCircle` | Anel de círculo por varredura (sem o truque dos dois FillCircle) | Circle Drawer |
| `juigo.History` + `juigo.Not` | Undo/redo genérico (pilhas + `CanUndo`/`CanRedo` prontos para `BindDisabled`) | Circle Drawer |
| `Scroll.Horizontal()` | Eixo X na rolagem (delta horizontal do trackpad, indicador inferior) | Cells |
| `uitest.FindAll` | Todos os widgets que casam com um seletor | Flight Booker, CRUD |

Tudo coberto pelo segundo golden test de dirty regions
(`uitest/damage_test.go`, `TestIncrementalNovosComponentes`): seleção de
tabela, cabeçalho fixo sob rolagem, popup abrindo/reposicionando/fechando,
rolagem horizontal e o realce de erro — frame incremental byte a byte igual
ao completo.

## Simplificações assumidas (deliberadas, não gaps)

- **Cells recalcula tudo a cada edição** — a grade é A–H × 1–12; o 7GUIs
  original propaga por grafo de dependências. Vira interessante só com
  planilhas grandes.
- **Cells edita pela barra de fórmulas** (fiel ao Excel) — edição in-place
  exigiria um Input flutuando sobre a célula selecionada.
- **Table é de células de TEXTO** — conteúdo rico em linhas fica com a
  List (widgets de verdade por linha) ou widgets custom.
- **Seleção de lista/tabela é única e por clique** — multi-seleção e
  navegação por teclado ficam para quando um app real pedir.
