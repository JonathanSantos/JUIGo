// Package render contém o backend de renderização do JUIGo.
//
// A renderização do JUIGo é feita por software (CPU) sobre um *image.RGBA;
// o OpenGL é usado exclusivamente como mecanismo de apresentação: o buffer
// vira uma textura que é desenhada em um quad cobrindo a janela inteira.
// Este pacote também oferece as primitivas de desenho (ver draw.go).
package render

import (
	"fmt"
	"image"

	"github.com/go-gl/gl/v3.3-core/gl"
)

const vertexShaderSrc = `#version 330 core
layout (location = 0) in vec2 inPos;
layout (location = 1) in vec2 inUV;
out vec2 uv;
void main() {
	uv = inUV;
	gl_Position = vec4(inPos, 0.0, 1.0);
}
` + "\x00"

const fragmentShaderSrc = `#version 330 core
in vec2 uv;
out vec4 outColor;
uniform sampler2D tex;
void main() {
	outColor = texture(tex, uv);
}
` + "\x00"

// Blitter apresenta um *image.RGBA na tela: mantém uma textura OpenGL do
// tamanho do buffer e a desenha em um quad que cobre a janela inteira.
// Deve ser criado e usado apenas na main thread, com um contexto GL corrente.
type Blitter struct {
	program uint32
	vao     uint32
	vbo     uint32
	texture uint32
	width   int
	height  int
}

// NewBlitter inicializa as funções OpenGL, compila o shader do quad e aloca
// a textura de apresentação com o tamanho dado (em pixels do buffer).
// Requer um contexto OpenGL 3.3 core corrente na thread chamadora.
func NewBlitter(width, height int) (*Blitter, error) {
	if err := gl.Init(); err != nil {
		return nil, fmt.Errorf("juigo/render: falha ao carregar funções OpenGL: %w", err)
	}

	program, err := buildProgram()
	if err != nil {
		return nil, err
	}

	// Quad fullscreen em TRIANGLE_STRIP. O V da textura é invertido porque
	// image.RGBA tem origem no canto superior esquerdo e o OpenGL amostra
	// texturas com origem no canto inferior esquerdo.
	vertices := [...]float32{
		// posX, posY, u, v
		-1, -1, 0, 1,
		1, -1, 1, 1,
		-1, 1, 0, 0,
		1, 1, 1, 0,
	}

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(&vertices[0]), gl.STATIC_DRAW)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)
	gl.EnableVertexAttribArray(1)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	// O buffer tem o tamanho exato do framebuffer (blit 1:1); NEAREST evita
	// qualquer filtragem.
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	b := &Blitter{program: program, vao: vao, vbo: vbo, texture: texture}
	b.resizeTexture(width, height)
	return b, nil
}

// resizeTexture realoca o armazenamento da textura para o novo tamanho.
func (b *Blitter) resizeTexture(width, height int) {
	b.width, b.height = width, height
	gl.BindTexture(gl.TEXTURE_2D, b.texture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0,
		gl.RGBA, gl.UNSIGNED_BYTE, nil)
}

// Upload envia o conteúdo do buffer para a textura de apresentação.
// Se o tamanho do buffer mudou (resize da janela), a textura é realocada.
func (b *Blitter) Upload(img *image.RGBA) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w == 0 || h == 0 {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, b.texture)
	if w != b.width || h != b.height {
		b.resizeTexture(w, h)
	}
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(w), int32(h),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
}

// Draw desenha o quad texturizado cobrindo o framebuffer inteiro.
// fbWidth e fbHeight são o tamanho do framebuffer da janela em pixels
// físicos (pode diferir do tamanho lógico em telas HiDPI).
func (b *Blitter) Draw(fbWidth, fbHeight int) {
	gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
	gl.UseProgram(b.program)
	gl.BindVertexArray(b.vao)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, b.texture)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
}

// Destroy libera os recursos OpenGL do Blitter.
func (b *Blitter) Destroy() {
	gl.DeleteTextures(1, &b.texture)
	gl.DeleteBuffers(1, &b.vbo)
	gl.DeleteVertexArrays(1, &b.vao)
	gl.DeleteProgram(b.program)
}

// buildProgram compila e linka o par de shaders do quad fullscreen.
func buildProgram() (uint32, error) {
	vs, err := compileShader(gl.VERTEX_SHADER, vertexShaderSrc)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(vs)

	fs, err := compileShader(gl.FRAGMENT_SHADER, fragmentShaderSrc)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(fs)

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		log := programInfoLog(program)
		gl.DeleteProgram(program)
		return 0, fmt.Errorf("juigo/render: falha ao linkar programa GL: %s", log)
	}
	return program, nil
}

// compileShader compila um shader e devolve erro com o log em caso de falha.
func compileShader(kind uint32, src string) (uint32, error) {
	shader := gl.CreateShader(kind)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := make([]byte, logLen+1)
		gl.GetShaderInfoLog(shader, logLen, nil, &log[0])
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("juigo/render: falha ao compilar shader: %s", string(log))
	}
	return shader, nil
}

// programInfoLog lê o log de link de um programa GL.
func programInfoLog(program uint32) string {
	var logLen int32
	gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLen)
	log := make([]byte, logLen+1)
	gl.GetProgramInfoLog(program, logLen, nil, &log[0])
	return string(log)
}
