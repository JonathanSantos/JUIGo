# JUIGo

Biblioteca minimalista de interface gráfica em Go (**J**onathan + **UI** + **Go**).

Renderização por software (CPU) sobre um `*image.RGBA`; GLFW cuida da janela e
dos eventos do sistema operacional, e o OpenGL é usado **apenas** para
apresentar o buffer na tela (upload de textura + quad fullscreen). O foco desta
fase é a arquitetura dos widgets básicos, não features.

## Arquitetura

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
                             Container/VBox/HBox, Grid, Scroll (clipping),
                             List virtualizada, overlay, CursorShape, Tooltip,
                             Text, Button, Input, TextArea (soft wrap),
                             Checkbox, Slider, ProgressBar, Radio, Image,
                             Dropdown, Modal
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
  examples/basic/            demo reativa: input, contador, checkbox, slider
```

Dependências entre pacotes (sempre acíclicas): `widget` → `theme`, `event`,
`state`, `render`; `theme` → `render`; `state` e `widget` alcançam o App em
execução só por `internal/hooks`, registrado na inicialização.

### Contratos centrais

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
- **Tema**: nenhuma cor ou tamanho hardcoded nos widgets — tudo vem de
  `Theme`. `Theme.MeasureString` é a única fonte de verdade para largura de
  texto (layout e posicionamento de cursor). Há dois temas prontos —
  `DefaultTheme` (claro) e `DarkTheme` (escuro, mesmas métricas) — e
  `App.SetTheme` troca em RUNTIME: a nova paleta se propaga pela árvore no
  próximo frame (widgets com `SetTheme` explícito mantêm o próprio).
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

## Dependências

- `github.com/go-gl/glfw/v3.3/glfw` — janela e eventos
- `github.com/go-gl/gl/v3.3-core/gl` — upload de textura + quad fullscreen
- `golang.org/x/image` — fonte (opentype), medidas em fixed point

A fonte **Go Regular** (https://go.dev/blog/go-fonts, licença BSD do projeto
Go) é embutida via `go:embed` — binários JUIGo não dependem de fontes do
sistema.

### Dependências de sistema (cgo)

O GLFW é compilado via cgo e precisa de toolchain C e headers de janela/GL:

- **Linux**:
  `libgl1-mesa-dev libx11-dev libxrandr-dev libxi-dev libxcursor-dev libxinerama-dev`
- **Windows**: toolchain gcc (MSYS2 ou TDM-GCC)
- **macOS**: Xcode Command Line Tools (`xcode-select --install`)

## DX: como se escreve uma aplicação

```go
valor := juigo.NewState("")
eco := juigo.NewState("Digite algo e clique em Enviar")

contador := juigo.Map(valor, func(s string) string {
    return fmt.Sprintf("%d caracteres", len([]rune(s)))
})

ui := juigo.NewVBox(
    juigo.NewText("").BindText(eco).Center(),
    juigo.NewHBox( // campo expande; botão fica na largura natural
        juigo.Grow(juigo.NewInput("Digite aqui…").BindValue(valor), 1),
        juigo.NewButton("Enviar", func() {
            eco.Set("Você digitou: " + valor.Get())
        }),
    ),
    juigo.NewText("").BindText(contador).Right(),
).Pad(16)

if err := juigo.Run("JUIGo — demo", 480, 240, ui); err != nil {
    log.Fatal(err)
}
```

As ideias que sustentam essa DX:

- **Tudo declarável inline** — callbacks e opções são métodos encadeáveis
  que devolvem o tipo concreto, no estilo dos `Bind*`:
  `NewInput("E-mail").BindValue(mail).OnSubmit(enviar)`,
  `NewSlider(0, 1).Step(0.1).OnChange(f)`,
  `NewModal(conteúdo).CloseOnBackdrop(false).OnClose(f)`,
  `NewText("").BindText(erro).Danger()`. Qualquer interface cabe em uma
  única expressão, sem variáveis temporárias; dados simples (`Label`,
  `Options`, `Min`/`Max`) continuam campos públicos.
- **Tema ambiente** — construtores não pedem tema: o App o injeta na árvore
  no *mount* (e a cada render, cobrindo widgets adicionados dinamicamente).
  `SetTheme` dá a um widget — ou a uma subárvore inteira, quando aplicado a
  um container — um tema próprio. Gap/Pad e os campos métricos são unidades
  lógicas, escaladas pelo tema (HiDPI transparente para o usuário).
- **Reatividade com `State[T]`** — `NewState`/`Get`/`Set`/`Watch` + `Map`
  para derivados. `Set` notifica os observadores sincronamente e redesenha a
  interface sozinho; setters de widgets (`SetText`) também. Bindings:
  `Text.BindText` (uma via) e `Input.BindValue` (duas vias). Eventos
  continuam callbacks (`OnClick`) — estado é para dados.
- **Grid e lista virtualizada** — `NewGrid(2, rótulo, campo, …)` alinha
  formulários (colunas na largura do filho mais largo; sobra vai às colunas
  com `Grow`); `NewList(n, criar, vincular)` mostra milhares de linhas com
  um pool de meia dúzia de widgets reciclados na rolagem, dentro de um
  `Scroll` comum. A TextArea quebra linha automaticamente (soft wrap) na
  largura do campo, preferindo espaços, com navegação por linhas visuais.
- **Layout flex sem cerimônia** — `Grow(w, peso)` expande um filho no eixo
  principal do box (proporcional ao peso); `NewSpacer()` empurra irmãos;
  `Centered`/`AtStart`/`AtEnd` controlam o eixo transversal (padrão:
  esticar). As funções devolvem o próprio widget com o tipo concreto — uso
  inline na árvore, sem nó extra, metadados internos no `BaseWidget`.
- **Disabled e loading** — `SetDisabled`/`BindDisabled(w, estado)` tiram o
  widget (e a subárvore) da interação de forma CENTRAL, no roteamento: sem
  clique, foco, Tab, hover ou tooltip, com esmaecimento visual
  (`Theme.DisabledWash`). `Button.SetLoading`/`BindLoading` mostram um
  spinner animado e implicam desabilitado.
- **Validação declarativa (`juigo/form`)** — campos são os mesmos States dos
  bindings; `form.Field(nome, form.Required(…), form.MinRunes(3, …))` deriva
  erros e validade como States: `BindDisabled(salvar, f.Invalid())`,
  `NewText("").BindText(f.ErrorOf(nome)).Danger()`. Erros aparecem no blur
  (`campo.OnBlur(func(){ f.Touch(nome) })`) ou no primeiro `Submit`, e daí
  acompanham a digitação ao vivo. `Rule` cobre regras multi-fonte (senhas
  coincidem) e `Check`, booleanas (aceite os termos).
- **Camada rápida (`juigo/quick`)** — os rituais mais comuns em uma
  expressão, compondo widget+form por convenção. Cada campo é um **handle
  tipado** dono do próprio State — a variável é a "chave" do formulário,
  verificada pelo compilador (nada de `Get("nome")` por string):

  ```go
  nome  := quick.Text("Nome:").Required("Informe o nome").Min(3, "Mínimo de 3")
  email := quick.Text("E-mail:").Email("E-mail inválido")
  idade := quick.Number("Idade:", 18).Max(120, "Confere a idade?")

  quick.Form(nome, email, idade).Submit("Salvar", func() {
      salvar(nome.Value(), email.Value(), idade.Value()) // string, string, int
  })
  ```

  monta a grade alinhada com erros por campo e Enter enviando. `Number` só
  aceita dígitos (filtro de digitação; vazio vale o valor inicial) e entrega
  `int`; `Notes`/`Options`/`Check` cobrem TextArea, Dropdown e Checkbox;
  `quick.Section("Endereço")` divide formulários longos; `Gap`/`Pad`
  ajustam o respiro. Quando o valor precisa viver fora do formulário,
  `campo.State()` expõe o State (Map, Watch, bindings) e `.Bind(estado)`
  liga o campo a um State externo. `quick.Confirm/Alert/Prompt` abrem
  diálogos de uma chamada — o campo do `Prompt` (e o botão do `Alert`) já
  nascem focados, porque overlays focam o primeiro focável do conteúdo;
  `quick.Labeled` e `quick.Buttons` cobrem rótulo+campo e a barra de ações.
  A regra da rampa: tudo aceita e devolve widgets comuns — quando o padrão
  pronto não servir, desça um nível naquele ponto (`Model()` dá o form;
  monte o Modal à mão) sem reescrever a tela.
- **Animações (`juigo/anim`)** — `anim.Tween(estado, alvo, duração, easing)`
  interpola qualquer `State[float64]` sobre os timers da aplicação, com
  retarget automático; compõe com os bindings (barra de progresso, rolagem
  suave via `Watch`+`ScrollTo`) e avança deterministicamente no relógio
  virtual do uitest.
- **Trabalho assíncrono com `App.Post`** — o ÚNICO método seguro para chamar
  de outras goroutines: entrega um callback à main thread na próxima volta
  do loop. Padrão: `SetLoading(true)` → goroutine faz o trabalho →
  `app.Post(func(){ SetLoading(false); estado.Set(resultado) })`.
- **Boot em uma linha** — `juigo.Run(título, w, h, raiz)`. `juigo.New`
  continua disponível para quem precisa do `*App` (tema, `Post`, `Bus`).
- **Renderização offscreen de primeira classe** — `juigo/offscreen` desenha
  qualquer árvore em um `*image.RGBA` sem janela nem GL, deterministicamente
  (mesma árvore ⇒ mesmos bytes): golden tests, screenshots de documentação e
  depuração de layout com `offscreen.Render` + `offscreen.SavePNG`.

## Inspector de depuração

**Ctrl/Cmd+I** em qualquer app JUIGo abre o inspector: contornos de todos os
widgets, realce do widget sob o ponteiro (inclusive desabilitados) e um
crachá com tipo, tamanho, posição, tamanho preferido e flags. A interface
continua interativa; aperte de novo para fechar. Funciona também no uitest
(`h.Key(juigo.KeyI, juigo.ModControl)`) e headless via
`Session.SetInspector`.

## Testando sua aplicação

Toda a lógica de interação (roteamento, foco, captura, hover, overlay,
tooltip) vive na `widget.Session`, sem GLFW — o App real é uma casca fina
sobre ela. O `juigo/uitest` dirige a MESMA Session sinteticamente, então
testar pelo harness é testar o comportamento real, headless e sem sleeps:

```go
func TestFormulario(t *testing.T) {
    valor := juigo.NewState("")
    ui := juigo.NewVBox(
        juigo.NewInput("Digite…").BindValue(valor),
        juigo.Tooltip(juigo.NewButton("Enviar", enviar), "Envia o formulário"),
    )

    h := uitest.New(t, ui, 480, 240)
    h.Click(uitest.Placeholder("Digite…")) // foca de verdade (Session)
    h.Type("olá, ação")                    // runes, com acentos
    h.Key(juigo.KeyTab)                    // navegação de foco real
    h.Hover(uitest.Text("Enviar"))
    h.Advance(600 * time.Millisecond)      // relógio virtual: tooltip aparece
    img := h.Screenshot()                  // determinístico → golden tests
}
```

Seletores: `uitest.Text`, `uitest.Placeholder`, `uitest.OfType[*juigo.Button]()`
e `uitest.Where(desc, predicado)`. `h.Session()` expõe foco, overlay, cursor e
tooltip para asserções; `h.Advance` dispara timers (piscada de cursor, atraso
de tooltip) na ordem, sem dormir.

## Rodando a demo

```bash
go run ./examples/basic
```

Na janela: digite no campo (acentos funcionam; o contador reage a cada
tecla), use setas/Home/End para mover o cursor, Tab para alternar o foco e
Enter/Espaço para acionar o botão focado. Clicar em **Enviar** atualiza o
título com o texto digitado.

E o clássico [7GUIs](https://eugenkiss.github.io/7guis/) — as sete GUIs de
referência (contador, conversor, reserva de voo, cronômetro, CRUD, desenho
de círculos com undo/redo e mini-planilha com fórmulas), cada uma num
pacote com testes de uitest; as limitações encontradas no exercício estão
registradas em [examples/7guis/GAPS.md](examples/7guis/GAPS.md):

```bash
go run ./examples/7guis
```

Testes da arquitetura (rodam sem janela):

```bash
go test ./...
```

## Fora de escopo (por enquanto)

Acessibilidade, IME (composição de texto asiático), multi-janela, edição
rica (negrito/itálico), drag-and-drop, tabelas com colunas e árvores. A
arquitetura foi pensada para recebê-los depois: eventos são tipos abertos, o
tema é injetado, containers são aninháveis e o desenho é isolado em
`render/`.

Limitação conhecida desta fase: o cache de glyphs cresce sob demanda e não é
descartado por LRU (irrelevante para textos de UI; cada glyph ocupa poucos
bytes).
