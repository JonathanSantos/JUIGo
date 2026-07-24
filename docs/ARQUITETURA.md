# Arquitetura do JUIGo

Este documento descreve a organização interna da lib — pacotes,
contratos e as decisões de arquitetura. Para o uso da API, veja o
[README](../README.md).

## Pacotes

O código é organizado em pacotes coesos; o pacote raiz `juigo` é uma
**fachada**: contém o `App` (janela + loop) e reexporta os tipos e
construtores dos subpacotes por alias. Aplicações comuns importam **apenas
`"juigo"`**; os subpacotes existem para casos avançados (widgets próprios,
shells alternativos, renderização offscreen).

```
juigo/
  doc.go, app.go, alias.go   fachada: App (janela GLFW, buffer, timers, loop
                             dirty — casca fina sobre a widget.Session) +
                             reexports dos subpacotes
  widget/                    contrato Widget, BaseWidget, roteamento
                             (DispatchAt/DispatchMouse/DispatchScroll,
                             DeepestAt/FocusableAt/Focusables), Mount,
                             flex (Grow/Spacer/Centered/AtStart/AtEnd),
                             Container/VBox/HBox, Grid, Scroll (clipping,
                             eixo horizontal opcional), List virtualizada
                             (com seleção), Table (cabeçalho fixo, seleção),
                             overlay, CursorShape, Tooltip, Text, Button,
                             Input, TextArea (soft wrap), Checkbox, Slider,
                             ProgressBar, Radio, Image, Dropdown, Modal,
                             Popup ancorado
  offscreen/                 Render/SavePNG: árvore → *image.RGBA sem janela
                             (golden tests, screenshots, depuração)
  uitest/                    harness de testes de UI: dirige a Session real
                             com cliques/teclas/hover sintéticos, seletores,
                             relógio virtual e screenshots determinísticos
  form/                      validação declarativa sobre estados: Field/
                             Check/Rule, validadores, Valid/Invalid,
                             ErrorOf com semântica touched e Submit
  quick/                     camada de conveniência: Form validado com
                             handles tipados (Text/Notes/Options/Check/
                             Number com estado interno, Value/State/Bind),
                             Section, diálogos Confirm/Alert/Prompt,
                             Labeled e Buttons — compõe widget+form, aceita
                             e devolve widgets comuns (sem segundo dialeto)
  anim/                      Tween de State[float64] com easing sobre os
                             timers — determinístico no uitest (Advance)
  theme/                     Theme: cores, métricas, escala HiDPI, cache de
                             glyphs e a fonte embutida (theme/assets/)
  event/                     tipos de evento, modificadores e o Bus síncrono
  state/                     State[T] (Get/Set/Watch) e Map — reatividade
  render/                    Blitter GL, primitivas de desenho, GlyphCache
  internal/hooks/            fiação App↔widgets: repaint e clipboard (fora
                             da API pública)
  examples/                  basic (demo reativa), 7guis (o benchmark
                             completo) e todo (TodoMVC em camadas)
```

Dependências entre pacotes (sempre acíclicas): `widget` → `theme`, `event`,
`state`, `render`; `theme` → `render`; `state` e `widget` alcançam o App em
execução só por `internal/hooks`, registrado na inicialização.

## Contratos centrais

- **Widget**: `Layout(bounds)`, `Draw(dst)`, `HandleEvent(ev) bool`,
  `Bounds()`, `Focusable()`, `PreferredSize()`. `BaseWidget` fornece os
  defaults para embutir nos widgets concretos.
- **Roteamento de eventos**:
  - *Mouse* e *rolagem* (`ScrollEvent`) roteiam por **geometria**: hit-test
    da raiz até o widget mais profundo que contém o ponto; propagam para
    cima se não consumidos (um `Scroll` no limite deixa o ancestral rolar).
    Quem consome um `MouseDown` **captura o mouse**: recebe
    `MouseMove`/`MouseUp` diretamente até o botão ser solto, mesmo fora dos
    próprios bounds — é o que permite arrastar o Slider e selecionar texto
    com fluidez. O cursor do sistema segue o widget sob o ponteiro (I-beam
    em campos, mãozinha em clicáveis), via `CursorShape`.
  - *Teclado/char* roteia por **foco**: direto ao widget focado, sem
    hit-test. `KeyEvent` carrega os modificadores (Shift, Ctrl, Alt, Super).
  - Clique em widget focável muda o foco; **Tab**/**Shift+Tab** avançam e
    recuam o foco na ordem da árvore. O App também entrega
    `MouseEnter`/`MouseLeave` (hover) e `FocusEvent` diretamente aos widgets
    afetados.
- **Threading**: tudo single-threaded na main thread
  (`runtime.LockOSThread`). Callbacks do GLFW mutam estado diretamente — sem
  mutex, sem goroutines. O `Publish` do EventBus é síncrono.
- **Loop**: `glfw.WaitEvents()` (sem busy loop) + flag `dirty`; só renderiza
  e faz `SwapBuffers` quando algo mudou. O buffer RGBA é reutilizado entre
  frames e só é realocado no resize. Com timers pendentes (cursor piscante,
  atraso do tooltip), o loop usa `WaitEventsTimeout` para acordar no
  vencimento — sem goroutines: os callbacks executam na main thread.
- **Dirty regions**: o dano nasce nos widgets (setters invalidam os próprios
  bounds; o diff de bounds no Layout cobre cascatas de geometria sozinho) e
  a Session acumula a união: cada frame repinta SÓ a região suja (visão
  `render.Clip`) e sobe só esse retângulo à GPU (`TexSubImage2D` parcial).
  Um golden test garante que o frame incremental é byte a byte igual a uma
  repintura completa após dezenas de interações. Regra prática: mude a
  interface por setters/States; mutação direta de campos públicos exige
  `App.Invalidate()`.
- **Overlay**: uma camada de sobreposição (popup do Dropdown) desenhada por
  cima da árvore, com prioridade nos eventos: clique/rolagem fora fecham e
  são engolidos, Tab/foco fora fecham, e o foco anterior é restaurado. O
  tooltip (`Tooltip(w, texto)`) é uma camada passiva acima de tudo, fora do
  hit-test.
- **Drag-and-drop**: `widget.StartDrag(payload, rótulo)` é chamado por um
  widget FONTE durante uma captura de mouse (tipicamente ao passar de um
  limiar de movimento); a partir daí a Session mantém um fantasma seguindo
  o cursor (camada passiva, como o tooltip), realça o `DropTarget` mais
  profundo sob o ponteiro cujo `CanDrop(payload)` aceite, entrega
  `Drop(payload, pos)` ao soltar e cancela no Escape ou na abertura de uma
  overlay. A fonte segue recebendo os eventos da própria captura — o gesto
  dela termina normalmente no MouseUp. `List.OnReorder`/`Table.OnReorder`
  usam essa infra para reordenar linhas sem widget custom: o próprio widget
  vira alvo do próprio payload e implementa `DropIndicator` — a Session
  troca o contorno por uma linha de inserção na fronteira mais próxima do
  cursor; soltar chama `fn(de, para)` com o índice final (mesmo lugar não
  chama). Arrastar entre widgets diferentes continua sendo
  `StartDrag`/`DropTarget` da aplicação.
- **Tema**: nenhuma cor ou tamanho hardcoded nos widgets — tudo vem de
  `Theme`. `Theme.MeasureString` é a única fonte de verdade para largura de
  texto (layout e posicionamento de cursor). Há dois temas prontos —
  `DefaultTheme` (claro) e `DarkTheme` (escuro, mesmas métricas) — e
  `App.SetTheme` troca em RUNTIME: a nova paleta se propaga pela árvore no
  próximo frame (widgets com `SetTheme` explícito mantêm o próprio).
  Cantos arredondados vêm de `Theme.Radius` (unidades lógicas, padrão 4):
  fundo, borda, anel de foco e a lavagem de desabilitado seguem o raio via
  `render.FillRoundRect`/`StrokeRoundRect`/`FillRoundRectOver` — cobertura
  por pixel só nos blocos de canto (antialiasing de rampa de 1px), faixas
  cheias no miolo, zero alocação, custo de 0–6% no frame completo. Com
  `Radius` zero as primitivas degradam byte a byte para as retas — o visual
  clássico (`ClassicTheme`) é o mesmo de antes, verificado por render
  idêntico; `ButtonBorder` com alfa zero desliga a borda do botão. Círculos
  (`FillCircle`/`StrokeCircle` — Radio, spinner) têm o mesmo antialiasing
  em todos os temas: qualidade de desenho, não uma escolha de visual.
- **Texto**: o Input opera sobre `[]rune` (cursor e âncora de seleção em
  runes, nunca bytes); acentuação e UTF-8 em geral funcionam. Suporta
  **seleção** (arraste do mouse ou Shift+setas/Home/End), **clipboard** do
  sistema com Ctrl/Cmd+A/C/X/V (colar filtra quebras de linha: campo de
  linha única) e **rolagem horizontal**: texto maior que o campo rola para
  manter o cursor sempre visível, recortado aos bounds (nada vaza). O desenho de texto passa por `Theme.DrawText`, que usa um
  **cache de glyphs** (`render.GlyphCache`): cada glyph é rasterizado uma
  única vez e o caminho quente não aloca — pixel a pixel idêntico ao
  `font.Drawer`, garantido por teste.
- **HiDPI**: o buffer RGBA tem o tamanho do *framebuffer* (pixels físicos,
  blit 1:1 com filtro NEAREST). O tema carrega uma escala
  (`Theme.SetScale`, aplicada pelo App a partir da escala de conteúdo do
  monitor, inclusive ao trocar de monitor): a fonte é re-rasterizada — texto
  nítido em retina — e os campos métricos do tema são LÓGICOS, convertidos
  por `Px`/`PaddingPx`/`SpacingPx`/`BorderPx`/`InputMinWidthPx`. O mouse é
  convertido de coordenadas lógicas para pixels antes do roteamento; widgets
  só veem pixels.

## Números de referência (M2, escala 1)

- Frame completo 480×320: ~92µs; incremental (dirty regions): ~14µs — zero
  alocações no caminho quente de desenho (jul/2026, cena do bench_test.go
  com os cantos arredondados do tema padrão; o antialiasing custa 0–6%).
- Binário da demo: 3,3MB com strip (≈1,1MB acima de uma janela GLFW crua).
- RSS ~101MB no macOS (≈88MB são a stack de GL do sistema).

## Limitações conhecidas

- O cache de glyphs cresce sob demanda e não é descartado por LRU
  (irrelevante para textos de UI; cada glyph ocupa poucos bytes).
- Os itens do popup do Dropdown não são widgets (desenhados à mão) — o
  seletor do uitest não os encontra; use `ClickAt`/setas.
