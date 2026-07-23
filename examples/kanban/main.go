// Kanban — o exemplo de DRAG-AND-DROP do JUIGo: três colunas e cartões que
// se arrastam entre elas. Cada cartão é um widget FONTE (inicia o arrasto
// ao passar de um pequeno limiar com o botão pressionado) e cada coluna é
// um DropTarget: a sessão cuida do fantasma que segue o cursor, do realce
// da coluna sob ele, do soltar e do Escape para cancelar. As linhas são
// reconstruídas da projeção do modelo, como no TodoMVC.
package main

import (
	"image"
	"log"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/render"
)

// cartao é o registro do modelo.
type cartao struct {
	id     int
	titulo string
}

// quadro é o modelo: os cartões de cada coluna, na ordem do quadro.
type quadro struct {
	colunas [3][]cartao
}

// mover leva o cartão id para o fim da coluna de destino (na própria
// coluna, vai para o fim — reordenação simples).
func (q *quadro) mover(id, destino int) {
	for c := range q.colunas {
		for i, k := range q.colunas[c] {
			if k.id == id {
				q.colunas[c] = append(q.colunas[c][:i], q.colunas[c][i+1:]...)
				q.colunas[destino] = append(q.colunas[destino], k)
				return
			}
		}
	}
}

// nomes das colunas do quadro.
var nomes = [3]string{"A fazer", "Fazendo", "Feito"}

// vista projeta o quadro em três colunas de cartões.
type vista struct {
	q       *quadro
	colunas [3]*coluna

	// Raiz é a árvore de widgets pronta para App.SetRoot.
	Raiz juigo.Widget
}

// nova monta a vista sobre o quadro dado.
func nova(q *quadro) *vista {
	v := &vista{q: q}
	linha := juigo.NewHBox().Pad(12).Gap(12)
	for i := range v.colunas {
		v.colunas[i] = &coluna{VBox: juigo.NewVBox().Pad(8).Gap(8), indice: i, aoSoltar: v.mover}
		linha.Add(juigo.Grow(v.colunas[i], 1))
	}
	v.Raiz = linha
	v.reprojeta()
	return v
}

// mover aplica a mudança no modelo e reprojeta.
func (v *vista) mover(id, destino int) {
	v.q.mover(id, destino)
	v.reprojeta()
}

// reprojeta reconstrói as colunas a partir do modelo (Clear + Add).
func (v *vista) reprojeta() {
	for i, col := range v.colunas {
		col.Clear()
		col.Add(juigo.NewText(nomes[i]).Center())
		for _, k := range v.q.colunas[i] {
			col.Add(&cartaoView{c: k})
		}
	}
}

// coluna decora um VBox como DropTarget: aceita o id de qualquer cartão e
// repassa o soltar à vista. O fundo pintado marca a área da coluna.
type coluna struct {
	*juigo.VBox
	indice   int
	aoSoltar func(id, destino int)
}

// CanDrop aceita payloads de cartão (o id).
func (c *coluna) CanDrop(payload any) bool {
	_, ok := payload.(int)
	return ok
}

// Drop leva o cartão solto para esta coluna.
func (c *coluna) Drop(payload any, _ image.Point) {
	c.aoSoltar(payload.(int), c.indice)
}

// Draw pinta o fundo da coluna e deixa o VBox desenhar os filhos.
func (c *coluna) Draw(dst *image.RGBA) {
	if th := c.Theme(); th != nil {
		render.FillRoundRect(dst, c.Bounds(), th.RadiusPx(), th.HoverBackground)
	}
	c.VBox.Draw(dst)
}

// cartaoView é o widget de um cartão: fonte de arrasto com limiar.
type cartaoView struct {
	juigo.BaseWidget
	c           cartao
	pressionado bool
	origem      image.Point
}

// CursorShape devolve a mãozinha: cartões se arrastam.
func (k *cartaoView) CursorShape() juigo.CursorShape {
	return juigo.CursorHand
}

// PreferredSize acomoda o título com respiro de cartão.
func (k *cartaoView) PreferredSize() image.Point {
	th := k.Theme()
	if th == nil {
		return image.Point{}
	}
	return image.Pt(th.MeasureString(k.c.titulo)+2*th.PaddingPx(), th.LineHeight()+2*th.Px(6))
}

// HandleEvent inicia o arrasto quando o ponteiro passa do limiar com o
// botão pressionado — clique simples não arrasta.
func (k *cartaoView) HandleEvent(ev juigo.Event) bool {
	e, ok := ev.(juigo.MouseEvent)
	if !ok {
		return false
	}
	switch e.Kind {
	case juigo.MouseDown:
		if e.Button != juigo.MouseButtonLeft {
			return false
		}
		k.pressionado = true
		k.origem = e.Pos
		return true
	case juigo.MouseMove:
		if !k.pressionado {
			return false
		}
		limiar := 4
		if th := k.Theme(); th != nil {
			limiar = th.Px(4)
		}
		d := e.Pos.Sub(k.origem)
		if d.X*d.X+d.Y*d.Y >= limiar*limiar {
			k.pressionado = false
			juigo.StartDrag(k.c.id, k.c.titulo)
		}
		return true
	case juigo.MouseUp, juigo.MouseLeave:
		k.pressionado = false
		return true
	}
	return false
}

// Draw desenha o cartão: fundo, borda e título.
func (k *cartaoView) Draw(dst *image.RGBA) {
	th := k.Theme()
	if th == nil {
		return
	}
	b := k.Bounds()
	radius := th.RadiusPx()
	render.FillRoundRect(dst, b, radius, th.InputBackground)
	render.StrokeRoundRect(dst, b, radius, th.BorderPx(), th.InputBorder)
	baseline := b.Min.Y + (b.Dy()-th.LineHeight())/2 + th.Ascent()
	th.DrawText(dst, k.c.titulo, image.Pt(b.Min.X+th.PaddingPx(), baseline), th.Text)
}

// exemplo devolve um quadro com cartões de amostra.
func exemplo() *quadro {
	return &quadro{colunas: [3][]cartao{
		{{0, "Rascunhar a proposta"}, {1, "Revisar o texto"}, {2, "Separar as imagens"}},
		{{3, "Ajustar o layout"}},
		{{4, "Publicar o site"}},
	}}
}

func main() {
	v := nova(exemplo())
	app, err := juigo.New("Kanban — drag-and-drop em JUIGo", 640, 360)
	if err != nil {
		log.Fatal(err)
	}
	app.SetRoot(v.Raiz)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
