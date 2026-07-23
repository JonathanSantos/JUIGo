# 7GUIs — limitações do JUIGo encontradas e melhorias de DX candidatas

Registro do exercício: cada GUI foi implementada até onde a lib alcança;
o que faltou (ou saiu deselegante) está aqui, por app, com candidatos de
melhoria. Itens marcados ✅ já foram resolvidos durante o próprio exercício.

## 3 · Flight Booker

- **Sem campo de data** — usei `Input` + validador de formato (`DD/MM/AAAA`),
  sem máscara de digitação nem calendário. Candidato: `quick.Date` sobre o
  molde `Field[T]` (com `Input.Filter` de dígitos/barras já dá uma máscara
  razoável).
- **Sem cor de fundo por instância no Input** — o 7GUIs original pinta o
  FUNDO do campo inválido de vermelho; nosso `Input` não expõe isso, então o
  erro aparece como texto `Danger` (o padrão da lib). Candidato: variação
  visual de erro no próprio Input (`Invalid(state)`?).
- **quick.Form não expõe o controle** — o campo de volta precisa de
  `BindDisabled` próprio (desabilitado quando "só ida"), o que forçou montar
  o formulário com `form` + `Grid` à mão. Candidato: `.Disabled(state)` no
  Field do quick, ou expor o controle do handle.

## 4 · Timer

- **Aplicações não têm timers públicos** — `internal/hooks` é inacessível;
  `anim.Tween` cobriu este caso com elegância (1s de animação = 1s de
  relógio), mas um timer que NÃO é interpolação (polling, relógio de parede)
  não tem API. Candidato: `App.After(d, fn)` / `App.Every(d, fn)` públicos,
  espelhados no relógio virtual do uitest.
- **UI que agenda timers na construção × uitest** — `uitest.New(t, UI(), …)`
  avalia `UI()` ANTES de ligar os hooks: o tween nasce sem scheduler e salta
  ao alvo. O contorno é `New` com raiz vazia + `Session().SetRoot(UI())` +
  `h.Layout()`. Candidato: `uitest.NewLazy(t, func() Widget, w, h)`.

## 5 · CRUD

- **Seleção de lista não é nativa** — a `List` virtualizada não tem noção de
  linha selecionada; fiz um widget custom de linha (desenha o realce, trata
  o clique) e o modelo chama `lista.Refresh()` a cada mudança. Funcionou bem
  (e provou o roteamento de eventos nas linhas do pool), mas é cerimônia
  para um padrão comum. Candidato: `SelectableList` no quick, ou seleção na
  própria `List`.
- **Escrever widget custom foi BOM** — `BaseWidget` + `Draw` + `HandleEvent`
  + `Invalidate` bastaram; tema via `Theme()`, primitivas via `juigo/render`.
  Nenhum gap aqui — registrado como ponto forte.

## 6 · Circle Drawer

- **Sem menu de contexto** — o ajuste de diâmetro abriu num `Modal` (que
  funcionou bem, com o slider ao vivo). Candidato: popup ancorado no ponto
  do clique (a infra de overlay já existe — o Dropdown faz isso).
- **Sem clique direito no uitest** — `Click`/`ClickAt` são só botão esquerdo;
  o teste usou `Session().PointerDown/Up` direto. Candidato:
  `h.RightClickAt(pos)`.
- **Sem `StrokeCircle` no render** — o anel do círculo são dois `FillCircle`
  (borda + miolo). Funciona, mas um traço de círculo evitaria o truque.
- **Sem infra de undo/redo** — snapshots à mão no modelo (30 linhas,
  aceitável). Candidato: `state.History` genérico (pilhas + estados
  `CanUndo`/`CanRedo` prontos para `BindDisabled`).

## 7 · Cells

- **Sem widget de tabela** — a planilha inteira é um widget custom que
  desenha cabeçalhos, grade, valores e seleção, e resolve cliques por
  coordenada. Viável (e rápido), mas é o maior gap da lib para apps de
  dados. Candidato: `Table` com colunas, cabeçalho e células viewportadas
  (a `List` já dá o modelo de virtualização).
- **Scroll é só vertical** — a grade A–H cabe na janela, mas uma planilha
  real precisaria de rolagem horizontal. Candidato: eixo X no `Scroll`.
- **Edição in-place não rola** — editar a célula NELA (em vez da barra de
  fórmulas) exigiria um Input posicionado sobre a célula selecionada; a
  barra de fórmulas foi a saída idiomática (e fiel ao Excel).
- Simplificação assumida: recálculo TOTAL a cada edição (grade pequena); o
  original propaga por grafo de dependências.

## Resolvidos durante o exercício ✅

- **`uitest.FindAll`** — não existia; necessário para "os 2 campos de data"
  e "quantas linhas a lista mostra". Adicionado à lib.
- **`h.Layout()` após `SetRoot`** — trocar a raiz sem renderizar deixa a
  árvore sem geometria e cliques caem no vazio; o helper já existia e os
  testes documentam o padrão.
