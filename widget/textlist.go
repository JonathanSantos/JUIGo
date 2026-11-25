package widget

// NewTextList cria uma List virtualizada de linhas de TEXTO: count linhas,
// cada uma exibindo text(i). É o atalho para o caso mais comum de lista —
// sem escrever a fábrica e o vínculo de linha à mão:
//
//	juigo.NewTextList(len(nomes), func(i int) string { return nomes[i] }).
//		OnChange(func(i int) { abrir(nomes[i]) })
//
// Para linhas com outros widgets (ícones, botões), use NewList.
func NewTextList(count int, text func(index int) string) *List[*Text] {
	return NewList(count,
		func() *Text { return NewText("") },
		func(row *Text, i int) { row.SetText(text(i)) },
	)
}

// NewTextTree cria uma Tree virtualizada de nós de TEXTO: o modelo vem de
// roots/children (como em NewTree) e cada nó exibe label(id).
//
//	juigo.NewTextTree(raizes, filhosDe, func(id string) string { return nome[id] })
func NewTextTree[ID comparable](
	roots func() []ID,
	children func(ID) []ID,
	label func(ID) string,
) *Tree[ID, *Text] {
	return NewTree(roots, children,
		func() *Text { return NewText("") },
		func(row *Text, id ID) { row.SetText(label(id)) },
	)
}
