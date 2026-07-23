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
  doc.go, app.go, alias.go   fachada: App (janela, buffer, foco, captura,
                             loop dirty) + reexports dos subpacotes
  widget/                    contrato Widget, BaseWidget, roteamento
                             (DispatchAt/DispatchMouse/DispatchScroll,
                             DeepestAt/FocusableAt/Focusables), Mount,
                             flex (Grow/Spacer/Centered/AtStart/AtEnd),
                             Container/VBox/HBox, Scroll (clipping),
                             CursorShape, Text, Button, Input, Checkbox,
                             Slider
  offscreen/                 Render/SavePNG: árvore → *image.RGBA sem janela
                             (golden tests, screenshots, depuração)
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
  frames e só é realocado no resize.
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
- **Boot em uma linha** — `juigo.Run(título, w, h, raiz)`. `juigo.New`
  continua disponível para quem precisa do `*App`.
- **Renderização offscreen de primeira classe** — `juigo/offscreen` desenha
  qualquer árvore em um `*image.RGBA` sem janela nem GL, deterministicamente
  (mesma árvore ⇒ mesmos bytes): golden tests, screenshots de documentação e
  depuração de layout com `offscreen.Render` + `offscreen.SavePNG`.

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

Animações, dirty regions, temas alternáveis, acessibilidade, IME, outros
widgets (radio/dropdown/modal), overlay (pré-requisito de dropdown/modal/
tooltip). A arquitetura
foi pensada para recebê-los depois: eventos são tipos abertos, o tema é
injetado, containers são aninháveis e o desenho é isolado em `render/`.

Limitação conhecida desta fase: o cache de glyphs cresce sob demanda e não é
descartado por LRU (irrelevante para textos de UI; cada glyph ocupa poucos
bytes).
