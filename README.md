# JUIGo

Biblioteca minimalista de interface gráfica em Go (**J**onathan + **UI** + **Go**).

Renderização por software (CPU) sobre um `*image.RGBA`; GLFW cuida da janela e
dos eventos do sistema operacional, e o OpenGL é usado **apenas** para
apresentar o buffer na tela (upload de textura + quad fullscreen). O foco desta
fase é a arquitetura dos widgets básicos, não features.

## Arquitetura

```
juigo/
  juigo.go         App: janela, buffer RGBA, árvore de widgets, foco, loop
  widget.go        interface Widget, BaseWidget embutível, roteamento
  container.go     Container (absoluto), VBox, HBox
  button.go        Button (normal|hover|pressed, OnClick)
  input.go         Input (edição por runes, cursor, placeholder, OnChange)
  text.go          Text (alinhamento left/center/right)
  theme.go         Theme: cores, fonte embutida, métricas, espaçamentos
  event.go         eventos internos (mouse/tecla/char/foco) + EventBus síncrono
  render/
    gl.go          Blitter: init GL, shader do quad, upload de textura
    draw.go        primitivas: FillRect, StrokeRect, DrawText, MeasureText
  examples/
    basic/main.go  demo: título + input + botão que ecoa o texto
```

### Contratos centrais

- **Widget**: `Layout(bounds)`, `Draw(dst)`, `HandleEvent(ev) bool`,
  `Bounds()`, `Focusable()`, `PreferredSize()`. `BaseWidget` fornece os
  defaults para embutir nos widgets concretos.
- **Roteamento de eventos**:
  - *Mouse* roteia por **geometria**: hit-test da raiz até o widget mais
    profundo que contém o ponto; propaga para cima se não consumido.
  - *Teclado/char* roteia por **foco**: direto ao widget focado, sem hit-test.
  - Clique em widget focável muda o foco; **Tab** avança o foco na ordem da
    árvore. O App também entrega `MouseEnter`/`MouseLeave` (hover) e
    `FocusEvent` diretamente aos widgets afetados.
- **Threading**: tudo single-threaded na main thread
  (`runtime.LockOSThread`). Callbacks do GLFW mutam estado diretamente — sem
  mutex, sem goroutines. O `Publish` do EventBus é síncrono.
- **Loop**: `glfw.WaitEvents()` (sem busy loop) + flag `dirty`; só renderiza
  e faz `SwapBuffers` quando algo mudou. O buffer RGBA é reutilizado entre
  frames e só é realocado no resize.
- **Tema**: nenhuma cor ou tamanho hardcoded nos widgets — tudo vem de
  `Theme`. `Theme.MeasureString` é a única fonte de verdade para largura de
  texto (layout e posicionamento de cursor).
- **Texto**: o Input opera sobre `[]rune` (cursor em runes, nunca bytes);
  acentuação e UTF-8 em geral funcionam. O desenho de texto passa por
  `Theme.DrawText`, que usa um **cache de glyphs** (`render.GlyphCache`):
  cada glyph é rasterizado uma única vez e o caminho quente não aloca —
  pixel a pixel idêntico ao `font.Drawer`, garantido por teste.
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

## Rodando a demo

```bash
go run ./examples/basic
```

Na janela: digite no campo (acentos funcionam), use setas/Home/End para mover
o cursor, Tab para alternar o foco e Enter/Espaço para acionar o botão focado.
Clicar em **Enviar** atualiza o título com o texto digitado.

Testes da arquitetura (rodam sem janela):

```bash
go test ./...
```

## Fora de escopo (por enquanto)

Animações, scroll, clipping hierárquico, dirty regions, temas alternáveis,
acessibilidade, IME, seleção de texto com mouse, clipboard, outros widgets
(checkbox/radio/slider/dropdown/modal). A arquitetura foi pensada para
recebê-los depois: eventos são tipos abertos, o tema é injetado, containers
são aninháveis e o desenho é isolado em `render/`.

Limitação conhecida desta fase: o cache de glyphs cresce sob demanda e não é
descartado por LRU (irrelevante para textos de UI; cada glyph ocupa poucos
bytes).
