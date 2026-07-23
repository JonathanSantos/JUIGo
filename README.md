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
                             Container/VBox/HBox, Scroll (clipping),
                             overlay (OpenOverlay/CloseOverlay), CursorShape,
                             Tooltip, Text, Button, Input, TextArea, Checkbox,
                             Slider, ProgressBar, Radio, Image, Dropdown, Modal
  offscreen/                 Render/SavePNG: árvore → *image.RGBA sem janela
                             (golden tests, screenshots, depuração)
  uitest/                    harness de testes de UI: dirige a Session real
                             com cliques/teclas/hover sintéticos, seletores,
                             relógio virtual e screenshots determinísticos
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
- **Overlay**: uma camada de sobreposição (popup do Dropdown) desenhada por
  cima da árvore, com prioridade nos eventos: clique/rolagem fora fecham e
  são engolidos, Tab/foco fora fecham, e o foco anterior é restaurado. O
  tooltip (`Tooltip(w, texto)`) é uma camada passiva acima de tudo, fora do
  hit-test.
- **Tema**: nenhuma cor ou tamanho hardcoded nos widgets — tudo vem de
  `Theme`. `Theme.MeasureString` é a única fonte de verdade para largura de
  texto (layout e posicionamento de cursor).
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

Três ideias sustentam essa DX:

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

Testes da arquitetura (rodam sem janela):

```bash
go test ./...
```

## Fora de escopo (por enquanto)

Animações, dirty regions, temas alternáveis, acessibilidade, IME, quebra
automática de linha na TextArea (soft wrap), tabelas/árvores. A arquitetura
foi pensada para recebê-los depois: eventos são tipos abertos, o tema é
injetado, containers são aninháveis e o desenho é isolado em `render/`.

Limitação conhecida desta fase: o cache de glyphs cresce sob demanda e não é
descartado por LRU (irrelevante para textos de UI; cada glyph ocupa poucos
bytes).
