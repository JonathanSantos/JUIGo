// Arquivos — o exemplo do trio "app de verdade": um SplitPane com a Tree de
// pastas à esquerda (filhos lidos preguiçosamente do disco, virtualizada), os
// arquivos da pasta selecionada à direita e os diálogos quick.OpenFile e
// quick.SaveFile — tudo 100% JUIGo, sem dependência de plataforma. Arraste o
// divisor (o cursor vira seta dupla), navegue a árvore pelo teclado (setas
// expandem/recolhem) e abra/salve pelo seletor de arquivos.
package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JonathanSantos/JUIGo"
	"github.com/JonathanSantos/JUIGo/quick"
)

func main() {
	raiz, err := os.UserHomeDir()
	if err != nil || raiz == "" {
		raiz = "."
	}
	app, err := juigo.New("Arquivos — JUIGo", 820, 520)
	if err != nil {
		log.Fatal(err)
	}
	// O exemplo roda no design system "papel e tinta" (docs/DESIGN.md).
	if th, err := juigo.ClaudeTheme(); err == nil {
		if err := app.SetTheme(th); err != nil {
			log.Fatal(err)
		}
	}
	app.SetRoot(ui(raiz))
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// ui monta o navegador de arquivos enraizado em raiz.
func ui(raiz string) juigo.Widget {
	// Cache do modelo da árvore: os filhos de cada pasta são lidos do disco
	// uma vez (a Tree consulta children a cada achatamento).
	cache := map[string][]string{}
	subpastas := func(dir string) []string {
		if got, ok := cache[dir]; ok {
			return got
		}
		var out []string
		if itens, err := os.ReadDir(dir); err == nil {
			for _, it := range itens {
				if it.IsDir() && !strings.HasPrefix(it.Name(), ".") {
					out = append(out, filepath.Join(dir, it.Name()))
				}
			}
		}
		cache[dir] = out
		return out
	}

	selDir := juigo.NewState(raiz)
	arvore := juigo.NewTree(
		func() []string { return []string{raiz} },
		subpastas,
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, dir string) { t.SetText(filepath.Base(dir)) },
	).BindSelected(selDir)
	arvore.Expand(raiz)

	// Painel direito: os ARQUIVOS da pasta selecionada na árvore.
	var arqs []string
	lista := juigo.NewList(0,
		func() *juigo.Text { return juigo.NewText("") },
		func(t *juigo.Text, i int) {
			if i >= 0 && i < len(arqs) {
				t.SetText(arqs[i])
			}
		},
	)
	recarrega := func(dir string) {
		arqs = arqs[:0]
		if itens, err := os.ReadDir(dir); err == nil {
			for _, it := range itens {
				if !it.IsDir() && !strings.HasPrefix(it.Name(), ".") {
					arqs = append(arqs, it.Name())
				}
			}
		}
		sort.Slice(arqs, func(i, j int) bool {
			return strings.ToLower(arqs[i]) < strings.ToLower(arqs[j])
		})
		lista.SetCount(len(arqs))
		lista.Refresh()
	}
	recarrega(raiz)

	caminho := juigo.Map(selDir, func(dir string) string {
		if dir == "" {
			return raiz
		}
		return dir
	})
	selDir.Watch(func(dir string) {
		if dir != "" {
			recarrega(dir)
		}
	})

	// Última ação dos diálogos, exibida no rodapé do painel.
	resultado := juigo.NewState("Selecione uma pasta; Abrir…/Salvar… usam o seletor.")
	dirAtual := func() string {
		if d := selDir.Get(); d != "" {
			return d
		}
		return raiz
	}
	abrir := juigo.NewButton("Abrir…", func() {
		quick.OpenFileIn(dirAtual(), "Abrir arquivo", func(p string, ok bool) {
			if ok {
				resultado.Set("Abriu: " + p)
			} else {
				resultado.Set("Abrir cancelado.")
			}
		})
	})
	salvar := juigo.NewButton("Salvar…", func() {
		quick.SaveFileIn(dirAtual(), "Salvar como", "sem-nome.txt", func(p string, ok bool) {
			if ok {
				resultado.Set("Salvaria em: " + p)
			} else {
				resultado.Set("Salvar cancelado.")
			}
		})
	})

	direita := juigo.NewVBox(
		juigo.NewText("Arquivos").Subtitle(),
		juigo.NewText("").BindText(caminho).Caption(),
		juigo.Grow(juigo.NewCard(juigo.NewScroll(lista)).Pad(4), 1),
		juigo.NewText("").BindText(resultado).Caption(),
		juigo.NewDivider(),
		juigo.NewHBox(juigo.NewSpacer(), salvar.Secondary(), abrir).Gap(8),
	).Pad(12).Gap(8)

	return juigo.NewSplitPane(
		juigo.NewScroll(arvore),
		direita,
	).Ratio(0.35).Min(120)
}
