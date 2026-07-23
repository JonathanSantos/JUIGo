// Package juigo é uma biblioteca minimalista de interface gráfica para Go.
//
// A renderização é feita por software (CPU) sobre um buffer *image.RGBA;
// GLFW fornece a janela e os eventos do sistema operacional, e o OpenGL é
// usado apenas para apresentar o buffer na tela (blit de textura em um quad
// fullscreen). Toda a biblioteca é single-threaded: janela, eventos e
// desenho vivem na main thread do processo.
//
// Este pacote é a FACHADA da biblioteca: contém o App (janela + loop) e
// reexporta os tipos e construtores dos subpacotes, para que uma aplicação
// comum importe apenas "github.com/JonathanSantos/JUIGo":
//
//	juigo/widget — Widget, BaseWidget, roteamento, Mount, containers e controles
//	juigo/theme  — Theme: cores, fonte embutida, métricas, escala HiDPI
//	juigo/event  — tipos de evento, modificadores e o EventBus síncrono
//	juigo/state  — State[T]: reatividade (Get/Set/Watch) e Map
//	juigo/render — primitivas de desenho por software e o blit OpenGL
//
// Importe os subpacotes diretamente para casos avançados: widgets próprios,
// shells alternativos ou renderização offscreen.
package juigo
