// Package chart traz gráficos temáticos do JUIGo: widgets que desenham
// séries de float64 com as primitivas de antialiasing (render.StrokeLine)
// e as cores do DESIGN SYSTEM — série em Accent, eixos e grade em
// SurfaceBorder, rótulos em legenda. Nada de cor própria: troque o tema e
// o gráfico acompanha.
//
//	chart.NewSpark(vendas)              // sparkline embutível numa linha
//	chart.NewLine(vendas).Min(0)        // gráfico de linha com eixos
//	chart.NewBars(porMes)               // barras (negativos descem)
//
// Os dados são do chamador: SetData substitui a fatia e redesenha (o
// gráfico NÃO copia — não mute a fatia entregue sem chamar SetData de
// novo); BindData observa um State. Como todo o JUIGo, desenhar não aloca
// (o buffer de pontos é reutilizado).
package chart

import (
	"image"
	"strconv"

	"github.com/JonathanSantos/JUIGo/render"
	"github.com/JonathanSantos/JUIGo/state"
	"github.com/JonathanSantos/JUIGo/widget"
)

// serie é o miolo comum: dados, faixa (auto ou fixada) e o buffer de
// pontos reutilizado no desenho.
type serie struct {
	widget.BaseWidget
	data   []float64
	min    float64
	max    float64
	hasMin bool
	hasMax bool
	pts    []image.Point
}

// setData troca os dados e redesenha.
func (s *serie) setData(d []float64) {
	s.data = d
	s.Invalidate()
}

// faixa devolve o intervalo efetivo [lo, hi] da série: o fixado por
// Min/Max, senão o dos dados (com folga quando degenerado).
func (s *serie) faixa() (lo, hi float64) {
	lo, hi = s.min, s.max
	if !s.hasMin || !s.hasMax {
		dlo, dhi := lo, hi
		if len(s.data) > 0 {
			dlo, dhi = s.data[0], s.data[0]
			for _, v := range s.data {
				if v < dlo {
					dlo = v
				}
				if v > dhi {
					dhi = v
				}
			}
		}
		if !s.hasMin {
			lo = dlo
		}
		if !s.hasMax {
			hi = dhi
		}
	}
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

// plot converte os dados em pontos dentro de r (índice→x, valor→y),
// reutilizando o buffer.
func (s *serie) plot(r image.Rectangle) []image.Point {
	n := len(s.data)
	s.pts = s.pts[:0]
	if n == 0 || r.Dx() <= 0 || r.Dy() <= 0 {
		return s.pts
	}
	lo, hi := s.faixa()
	for i, v := range s.data {
		x := r.Min.X
		if n > 1 {
			x += i * (r.Dx() - 1) / (n - 1)
		}
		f := (v - lo) / (hi - lo)
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
		y := r.Max.Y - 1 - int(f*float64(r.Dy()-1)+0.5)
		s.pts = append(s.pts, image.Pt(x, y))
	}
	return s.pts
}

// Spark é a sparkline: só a linha da série, do tamanho de uma palavra —
// para dentro de células, cartões e rodapés.
type Spark struct {
	serie
}

// NewSpark cria uma sparkline com os dados dados. O tema é herdado no
// mount.
func NewSpark(data []float64) *Spark {
	s := &Spark{}
	s.data = data
	return s
}

// SetData substitui a série e redesenha.
func (s *Spark) SetData(d []float64) { s.setData(d) }

// BindData observa o State: cada Set troca a série. Encadeável.
func (s *Spark) BindData(st *state.State[[]float64]) *Spark {
	s.data = st.Get()
	st.Watch(func(d []float64) { s.setData(d) })
	return s
}

// PreferredSize devolve o tamanho de uma sparkline (96×28 lógicos).
func (s *Spark) PreferredSize() image.Point {
	th := s.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Pt(th.Px(96), th.Px(28))
}

// Draw desenha a linha da série em Accent.
func (s *Spark) Draw(dst *image.RGBA) {
	th := s.Theme()
	if th == nil || len(s.data) < 2 {
		return
	}
	pts := s.plot(s.Bounds().Inset(th.Px(2)))
	render.StrokePolyline(dst, pts, max(th.Px(2), 1), th.Accent)
}

// Line é o gráfico de linha: eixos, a série em Accent e os extremos da
// faixa em legenda.
type Line struct {
	serie
	// Rótulos dos extremos cacheados pela faixa corrente (Draw não aloca
	// enquanto a faixa não muda).
	loV, hiV float64
	loS, hiS string
	labelsOK bool
}

// NewLine cria um gráfico de linha com os dados dados. O tema é herdado no
// mount.
func NewLine(data []float64) *Line {
	l := &Line{}
	l.data = data
	return l
}

// Min fixa o piso da faixa (ex.: 0 para séries absolutas). Encadeável.
func (l *Line) Min(v float64) *Line {
	l.min, l.hasMin = v, true
	l.Invalidate()
	return l
}

// Max fixa o teto da faixa. Encadeável.
func (l *Line) Max(v float64) *Line {
	l.max, l.hasMax = v, true
	l.Invalidate()
	return l
}

// SetData substitui a série e redesenha.
func (l *Line) SetData(d []float64) { l.setData(d) }

// BindData observa o State: cada Set troca a série. Encadeável.
func (l *Line) BindData(st *state.State[[]float64]) *Line {
	l.data = st.Get()
	st.Watch(func(d []float64) { l.setData(d) })
	return l
}

// PreferredSize devolve o tamanho padrão de um gráfico (280×160 lógicos).
func (l *Line) PreferredSize() image.Point {
	th := l.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Pt(th.Px(280), th.Px(160))
}

// Draw desenha os eixos (esquerda e base), a série e os extremos da faixa.
func (l *Line) Draw(dst *image.RGBA) {
	th := l.Theme()
	if th == nil {
		return
	}
	b := l.Bounds()
	line := max(th.BorderPx(), 1)
	render.FillRect(dst, image.Rect(b.Min.X, b.Min.Y, b.Min.X+line, b.Max.Y), th.SurfaceBorder)
	render.FillRect(dst, image.Rect(b.Min.X, b.Max.Y-line, b.Max.X, b.Max.Y), th.SurfaceBorder)
	if len(l.data) >= 2 {
		plotR := b.Inset(th.Px(6))
		render.StrokePolyline(dst, l.plot(plotR), max(th.Px(2), 1), th.Accent)
	}
	// Extremos da faixa em legenda apagada, dentro do canto esquerdo.
	f := th.Caption()
	lo, hi := l.faixa()
	if !l.labelsOK || lo != l.loV || hi != l.hiV {
		l.loV, l.hiV = lo, hi
		l.loS, l.hiS = formatoCurto(lo), formatoCurto(hi)
		l.labelsOK = true
	}
	f.Draw(dst, l.hiS, image.Pt(b.Min.X+th.Px(6), b.Min.Y+f.Ascent()+th.Px(2)), th.Placeholder)
	f.Draw(dst, l.loS, image.Pt(b.Min.X+th.Px(6), b.Max.Y-th.Px(4)), th.Placeholder)
}

// Bars é o gráfico de barras verticais: positivos sobem, negativos descem
// da linha do zero.
type Bars struct {
	serie
}

// NewBars cria um gráfico de barras com os dados dados. O tema é herdado
// no mount.
func NewBars(data []float64) *Bars {
	b := &Bars{}
	b.data = data
	return b
}

// SetData substitui a série e redesenha.
func (b *Bars) SetData(d []float64) { b.setData(d) }

// BindData observa o State: cada Set troca a série. Encadeável.
func (b *Bars) BindData(st *state.State[[]float64]) *Bars {
	b.data = st.Get()
	st.Watch(func(d []float64) { b.setData(d) })
	return b
}

// Min fixa o piso da faixa. Encadeável.
func (b *Bars) Min(v float64) *Bars {
	b.min, b.hasMin = v, true
	b.Invalidate()
	return b
}

// Max fixa o teto da faixa. Encadeável.
func (b *Bars) Max(v float64) *Bars {
	b.max, b.hasMax = v, true
	b.Invalidate()
	return b
}

// PreferredSize devolve o tamanho padrão de um gráfico (280×160 lógicos).
func (b *Bars) PreferredSize() image.Point {
	th := b.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Pt(th.Px(280), th.Px(160))
}

// Draw desenha a linha do zero e as barras arredondadas em Accent.
func (b *Bars) Draw(dst *image.RGBA) {
	th := b.Theme()
	if th == nil || len(b.data) == 0 {
		return
	}
	r := b.Bounds().Inset(th.Px(6))
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	lo, hi := b.faixa()
	if lo > 0 && !b.hasMin {
		lo = 0 // barras absolutas nascem do zero
	}
	if hi < 0 && !b.hasMax {
		hi = 0
	}
	if hi <= lo {
		hi = lo + 1
	}
	zeroY := r.Max.Y - int((0-lo)/(hi-lo)*float64(r.Dy())+0.5)
	n := len(b.data)
	gap := th.Px(2)
	bw := max((r.Dx()-gap*(n-1))/n, 1)
	radius := min(th.Px(3), bw/2)
	for i, v := range b.data {
		x := r.Min.X + i*(bw+gap)
		f := (v - lo) / (hi - lo)
		y := r.Max.Y - int(f*float64(r.Dy())+0.5)
		var barra image.Rectangle
		if y <= zeroY {
			barra = image.Rect(x, y, x+bw, zeroY)
		} else {
			barra = image.Rect(x, zeroY, x+bw, y)
		}
		if barra.Dy() < 1 {
			barra.Max.Y = barra.Min.Y + 1
		}
		render.FillRoundRect(dst, barra, radius, th.Accent)
	}
	line := max(th.BorderPx(), 1)
	render.FillRect(dst, image.Rect(r.Min.X, zeroY-line/2, r.Max.X, zeroY-line/2+line), th.SurfaceBorder)
}

// formatoCurto devolve o número com poucos dígitos para os rótulos de
// extremos (inteiro quando é inteiro; senão uma casa decimal).
func formatoCurto(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}
