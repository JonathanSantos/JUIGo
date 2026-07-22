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
  acentuação e UTF-8 em geral funcionam.

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

Limitações conhecidas desta fase: sem suporte HiDPI (em telas retina o buffer
é esticado pelo blit, com leve borrão) e sem cache de glyphs (a rasterização
de texto aloca dentro de `x/image`).
