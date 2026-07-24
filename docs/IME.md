# IME no JUIGo — avaliação e plano (jul/2026)

Entrada de texto por composição (japonês, chinês, coreano; "dead keys"
complexos) exige três peças que hoje o JUIGo não tem: receber o texto de
**pré-edição** (preedit) do sistema, desenhá-lo no caret com o cursor de
composição, e informar ao sistema **onde** o caret está para a janela de
candidatos aparecer no lugar certo.

## Onde o ecossistema está

- O GLFW upstream **não tem IME**: é o issue mais antigo do projeto
  ([#41](https://github.com/glfw/glfw/issues/41)) e a implementação
  completa vive num PR aberto há anos
  ([#2130](https://github.com/glfw/glfw/pull/2130), redo do
  [#2117](https://github.com/glfw/glfw/pull/2117), discussão em
  [#2097](https://github.com/glfw/glfw/issues/2097)). No estado atual do
  PR, Windows é o alvo mais completo; no macOS o posicionamento da janela
  de candidatos (`glfwSetPreeditCursorPos`) não funciona.
- Os bindings `go-gl/glfw` que usamos seguem o upstream — sem o merge,
  **não há evento de preedit para receber**
  ([go-gl/glfw#392](https://github.com/go-gl/glfw/issues/392)).
- Existem saídas fora do upstream: bindings de forks com o patch de IME e
  projetos como o [mado](https://pkg.go.dev/github.com/kanryu/mado), que
  nasceu exatamente dessa lacuna.

## Opções

| Opção | Custo | Risco |
| --- | --- | --- |
| **A. Esperar o upstream** | zero | prazo indefinido (o PR é de 2022) |
| **B. Fork do GLFW com o patch de IME** | quebra a regra "só glfw/gl/x/image"; manter fork | o patch muda; macOS incompleto |
| **C. Shim nativo próprio (cgo/`NSTextInputClient` no macOS)** | profundo e por plataforma | manutenção alta; foge do escopo de estudo |
| **D. Preparar a LIB primeiro** | médio, todo testável headless | nenhum — vale para qualquer rota |

## Plano recomendado

1. **Fase D (lib, sem dependência nova) — ✅ FEITA (jul/2026)** — o
   protocolo de composição vive na lib, moldado pela API do PR do GLFW:
   `event.PreeditEvent{Text, Caret, Blocks, FocusedBlock}` roteado por foco
   (`Session.Preedit`); Input e TextArea desenham a composição INLINE no
   cursor, sublinhada (blocos com o em conversão destacado), sem entrar no
   texto — o commit chega pelos `CharEvent` normais; compor sobre seleção a
   substitui e o blur descarta; `CaretRect()` (contrato `widget.TextCaret`
   + `Session.CaretRect`) é a âncora da janela de candidatos. Tudo
   dirigível headless por `uitest.Preedit`. Limitação aceita: na TextArea a
   composição não recalcula o soft wrap (composições são curtas; o excesso
   recorta à direita).
2. **Rota de plataforma** — decisão em aberto (é uma dependência nova):
   adotar o fork com o patch (opção B), ou esperar o merge (opção A).
   Reavaliar a cada release do GLFW. Quando existir, a fiação é: preedit
   callback → `Session.Preedit`; caret/foco mudou → `Session.CaretRect` →
   `glfwSetPreeditCursorPos`.
3. **Fonte** — pré-requisito paralelo da rota de plataforma: a Go Regular
   embutida NÃO cobre CJK (os glifos da composição real sairiam como
   quadrados). Exibir japonês/chinês/coreano pede fallback de fonte no
   tema (uma segunda face consultada quando o glifo falta) — independente
   do IME em si.

Enquanto isso, o que já funciona hoje: entrada Latin completa (acentos e
dead keys simples chegam compostos pelo `CharCallback`), índices em runes
em todos os campos, e toda a mecânica de composição pronta e testada na
lib.
