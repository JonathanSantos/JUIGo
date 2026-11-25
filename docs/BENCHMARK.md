# JUIGo — Benchmark e análise de viabilidade

*Medições de 24/jul/2026, na `v0.17.0` (+ correção de alocação do Label
pega por este próprio relatório). Números novos substituem os históricos
citados em docs/ARQUITETURA.md.*

## Veredito executivo

O JUIGo é **viável hoje** para aplicações desktop reais de pequeno e médio
porte — ferramentas internas, utilitários de desenvolvimento, painéis,
front-ends de serviços próprios — com uma ressalva de plataforma (macOS é
o único ambiente verificado) e três limites estruturais conhecidos
(acessibilidade, IME/CJK e shaping/RTL). Os números: um frame completo
numa tela retina inteira custa **0,55 ms de CPU** (3% do orçamento de
60 Hz), o caminho quente **não aloca**, o binário sai em **3,7 MB**, a
cobertura de testes real é **80%** e a dependência externa é só o trio
GLFW/GL/x-image. Para produto de consumo internacional ou com requisitos
de acessibilidade, ainda não.

## Ambiente e metodologia

- **Máquina:** Apple M2, 24 GB, macOS 26.5.1, Go 1.26.4 (arm64).
- **Benchmarks:** `go test . -run '^$' -bench . -benchtime 1s` —
  [bench_test.go](../bench_test.go) (a régua original) +
  [bench_extra_test.go](../bench_extra_test.go) (componentes novos).
  Frames medem CPU pura (fundo + layout + desenho); upload de textura e
  present exigem GL e usam a referência histórica medida no App real.
- **Cobertura:** `go test -coverpkg=<pacotes da lib> ./...` — cruzada,
  porque os testes de widget moram majoritariamente no harness `uitest`.
- **Heap:** programa headless com `runtime.ReadMemStats` (GC forçado
  antes/depois), sem janela.

## Desempenho de frame

| Cenário | CPU/frame | Alocações | % do orçamento 60 Hz |
| --- | ---: | ---: | ---: |
| Frame completo 480×320, tema Default (Nanquim, Inter) | 52 µs | 0 | 0,3% |
| Frame completo 480×320, tema Classic (cantos retos, Go) | 44 µs | 0 | 0,3% |
| Frame completo 960×640 (escala 2×) | 142 µs | 0 | 0,9% |
| Frame completo **2560×1600** (retina 13" inteira) | 553 µs | 0 | 3,3% |
| Frame **incremental** (dirty region de um campo) | 15 µs | 0 | 0,1% |
| Rolagem em List de **10.000 linhas** (evento + rebind + frame) | 40 µs | 3 (64 B) | 0,2% |
| Tecla digitada no Input (inserir + apagar) | 5,3 µs | 6 (288 B) | — |

Leituras:

- **Nanquim e Classic custam quase o mesmo**: 52 µs contra 44 µs fica
  dentro da oscilação de ±30% entre rodadas descrita abaixo (o que sobra
  vem do raio com antialiasing e dos glifos da Inter, mais detalhados). (O
  antigo tema "papel e tinta", removido na v0.20, custava ~1,9× por causa
  do raio 10 e da serif.) Só estas duas linhas foram medidas de novo
  (2026-11-24, `-count=3`); as demais são da v0.19.
- **Dirty regions entregam 3,3×** sobre o frame completo na cena pequena;
  a vantagem cresce com a janela (o incremental é proporcional ao dano,
  não à tela).
- O custo que a CPU não vê: na medição histórica do App real (jul/2026,
  janela 1400×850 retina), upload de textura + present somavam **~2,1 ms**
  — o teto de tela grande é o transporte para a GPU, não o desenho. O
  upload parcial (`TexSubImage2D`) acompanha o dano.

## Primitivas e componentes

| Operação | Custo | Alocações |
| --- | ---: | ---: |
| Polilinha AA de 12 segmentos, espessura 2, 480×320 | 93 µs | 0 |
| CrossFade 480×320 (um quadro de transição Fade) | 521 µs | 0 |
| Label de ~60 palavras, desenho com wrap cacheado | 70 µs | 0 |
| Label, **reflow completo** por mudança de largura | 19 µs | 0 |
| chart.Line completo (eixos + série AA + rótulos), 300×180 | 28 µs | 0 |

- O **CrossFade é a operação mais cara por quadro** da lib (mistura por
  byte da área inteira). Numa janela 900×560 projeta ~1,7 ms/quadro — ok a
  60 Hz, e só dura os 280 ms da transição; os slides do Navigator usam
  cópia (`draw.Src`) e não pagam isso. É o primeiro candidato a SIMD/
  otimização se transições em 4K virarem rotina.
- Este relatório **pagou o próprio custo**: o benchmark do Label revelou 2
  alocações por frame (method values virando closures) — corrigidas; o
  caminho quente voltou a zero.

## Memória

| Métrica | Valor |
| --- | ---: |
| Binário do chat (exemplo mais completo), `-s -w` | 3,8 MiB (4,0 MB) |
| Binário com símbolos | 5,9 MiB |
| — dos quais fontes embutidas (Inter ×2, Go Regular/Mono/Bold/Mono Bold, Fira) | ~1,8 MB |
| Tema (fontes rasterizadas + caches), heap — medido na v0.19 | ~0,3 MB |
| UI densa 900×560 (SplitPane + List 10k + sessão + framebuffer), heap | ~2,2 MB |
| RSS no App real com GL (referência histórica, 1400×850) | ~101 MB (88 MB são a stack OpenGL do macOS) |

O heap da interface em si é desprezível — o framebuffer domina (900×560×4
≈ 2 MB) e cresce com a janela. O RSS "alto" é a pilha GL da Apple, comum a
qualquer app OpenGL no macOS; o que a lib controla está em dígitos únicos
de MB.

## Base de código e qualidade

| Métrica | Valor |
| --- | ---: |
| Biblioteca (sem testes/exemplos) | 19.585 LOC |
| Testes | 7.893 LOC · **234 testes** · 12 benchmarks |
| Exemplos (10 apps) | 6.071 LOC |
| Cobertura real (cruzada) da lib | **80,1%** das instruções |
| Suíte completa, do zero | 9,6 s |
| Dependências diretas | **3** (go-gl/gl, go-gl/glfw, x/image) |

A razão teste/código (0,40) é sustentada pelo harness `uitest`: os testes
dirigem o mesmo núcleo de interação do App real, headless e sem sleeps —
foi o que permitiu, nesta base, mudar a altura de todas as linhas de todas
as listas sem quebrar um teste, e é o que mantém a suíte em ~10 s. Os
golden tests "incremental == completo" são a rede de segurança das dirty
regions em cada widget novo.

## Posição no ecossistema (qualitativa)

| | JUIGo | Fyne | Gio |
| --- | --- | --- | --- |
| Renderização | software + blit GL | GPU (OpenGL/ES) | GPU (Vulkan/Metal/GL) |
| Plataformas | macOS verificado; Linux/Windows mapeados | desktop + mobile | desktop + mobile + web |
| Acessibilidade | não | parcial | parcial |
| IME/CJK | plumbing pronto; sem rota de plataforma | sim | sim |
| Testes de UI | **harness próprio, determinístico** | básico | manual |
| Design system | **completo (tokens/tipos/regras)** | temas | por conta do app |
| Dependências | 3 | dezenas | poucas |

O nicho honesto do JUIGo não é competir em matriz de plataforma — é a
combinação de **DX declarativa + testabilidade real + identidade visual
pronta** num binário pequeno com dependências que cabem numa linha.

## Limites estruturais e riscos

1. **Plataforma:** só macOS foi verificado de ponta a ponta; Linux/Windows
   compilam com as dependências mapeadas no CI (headless), sem validação
   visual. **Risco correlato:** a Apple mantém o OpenGL deprecado — funciona
   hoje, mas a rota de longo prazo no macOS é Metal (via ANGLE ou troca do
   blit; o desenho por software isola bem essa fronteira: só o blitter
   precisaria mudar).
2. **Acessibilidade:** sem ponte para leitores de tela (NSAccessibility/
   UIA/AT-SPI). Navegação por teclado é quase completa — é o que dá para
   ter barato.
3. **Texto internacional:** sem fallback de fonte (CJK/emoji viram tofu),
   sem shaping (ligaduras, árabe, índico) e sem RTL. O IME tem o plumbing
   pronto (fase D), aguardando a rota de plataforma do GLFW.
4. **Fator ônibus:** projeto de estudo de um mantenedor; a mitigação real
   é a suíte de testes e a documentação de arquitetura.

## Casos de uso

**Recomendado hoje:** ferramentas internas e utilitários de dev (o
mini-proxy é a prova), painéis/dashboards (chart + design system),
front-ends de serviços próprios, protótipos de produto com identidade,
ensino de GUI.

**Ainda não:** produto de consumo internacionalizado (CJK/RTL), qualquer
contexto com requisito legal de acessibilidade, mobile/web.

## Gatilhos de otimização mapeados

| Sintoma | Resposta preparada |
| --- | --- |
| Transições Fade pesadas em janelas 4K | ✅ feito na v0.19: CrossFade por palavra de 64 bits (ver abaixo) |
| Upload dominando em telas grandes | dano em N retângulos (fase 3, plano no ARQUITETURA.md) |
| Glyph cache crescendo sem teto | LRU (limitação aceita, ainda sem dor) |
| OpenGL removido do macOS | trocar só o blitter (Metal/ANGLE) |

## Rodada de otimização da v0.19

Uma auditoria apontou trabalho repetido por frame e por tecla; esta rodada o
removeu. Medição **antes/depois na mesma máquina** (M2, Go 1.26.4), binários
de benchmark do commit anterior e do atual rodados intercalados, seis
rodadas cada, menor tempo de cada lado. A máquina estava sob carga durante a
medição — os tempos absolutos ficaram acima dos das tabelas acima (medidos
em repouso) e oscilaram na casa de ±30% entre rodadas; **o que vale daqui é
a razão**, e só onde ela supera essa oscilação. As contagens de alocação são
exatas.

| Cenário | Antes | Depois | Ganho | Alocações |
| --- | ---: | ---: | ---: | ---: |
| Arraste de seleção no Input, 1000 caracteres (por movimento) | 1.779 µs | 5,5 µs | **323×** | 263 → 0 |
| Movimento de cursor numa TextArea de 250 linhas | 152 µs | 2,8 µs | **54×** | 506 → 0 |
| `PreferredSize` de um Dropdown com 50 opções | 34,6 µs | 0,68 µs | **51×** | 0 → 0 |
| Reflow de Label (largura mudou) | 30,3 µs | 1,6 µs | **18×** | 0 → 0 |
| CrossFade 480×320 | 854 µs | 163 µs | **5,3×** | 0 → 0 |
| Tecla digitada no Input (inserir + apagar) | 7,8 µs | 2,6 µs | **3,0×** | 6 → 2 |
| Frame completo 480×320, papel e tinta (tema removido na v0.20) | 157 µs | 83 µs | 1,9× | 0 → 0 |
| Frame completo 480×320, Default | 83 µs | 58 µs | 1,4× | 0 → 0 |
| Rolagem em List de 10.000 linhas | 66 µs | 47 µs | 1,4× | 3 → 1 |
| Frame completo 2560×1600 | 677 µs | 705 µs | sem mudança | 0 → 0 |

O que mudou:

- **Medida de texto memorizada** (`render.Measurer`, por trás de
  `Theme.MeasureString`): o layout media os mesmos rótulos várias vezes por
  frame; agora é uma consulta de mapa. É o que explica os ganhos de frame,
  do Dropdown e do Label.
- **Tabela de larguras de prefixo** (`render.PrefixWidths` + busca binária):
  clique→índice no Input deixou de medir prefixo por prefixo (era O(n²) com
  uma string alocada por caractere).
- **Movimento de cursor não reconstrói o texto** da TextArea.
- **CrossFade em palavras de 64 bits**: oito canais por iteração.
- **`Children()` sem alocação** em Scroll, Modal, Popup e paleta; a piscada
  do cursor não cria closure por tique.
- O frame em tela cheia não muda: lá domina o preenchimento de pixels, não a
  medição.

Benchmarks novos em `bench_edit_test.go`.

## Reproduzindo

```bash
go test . -run '^$' -bench . -benchtime 1s   # todos os benchmarks
go test -count=1 ./...                        # suíte completa (~10 s)
go build -ldflags="-s -w" ./examples/chat     # binário de referência
```
