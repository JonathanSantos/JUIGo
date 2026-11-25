package widget

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// chainExempt são os widgets exportados que NÃO recebem os encadeáveis de
// layout: camadas de sobreposição nunca são filhas de um box.
var chainExempt = map[string]bool{"Modal": true, "Popup": true, "TooltipView": true}

// TestTodoWidgetTemEncadeaveis garante que todo widget exportado do pacote
// (struct que embute BaseWidget, direto ou via Container) tem os métodos
// gerados por internal/gen/chain. Widget novo sem eles? Acrescente o tipo
// à lista do gerador e rode `go generate ./...`.
func TestTodoWidgetTemEncadeaveis(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	embeds := map[string]bool{}  // tipos exportados que embutem BaseWidget/Container
	hasGrow := map[string]bool{} // tipos com o método Grow declarado
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, f := range st.Fields.List {
						if len(f.Names) != 0 {
							continue
						}
						if id, ok := f.Type.(*ast.Ident); ok && (id.Name == "BaseWidget" || id.Name == "Container") {
							embeds[ts.Name.Name] = true
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv != nil && d.Name.Name == "Grow" {
					hasGrow[receiverName(d.Recv.List[0].Type)] = true
				}
			}
		}
	}
	var missing []string
	for name := range embeds {
		if !chainExempt[name] && !hasGrow[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("widgets sem os encadeáveis gerados: %v — inclua em internal/gen/chain e rode go generate ./...", missing)
	}
}

// receiverName extrai o nome do tipo de um receptor (*T, *T[A], *T[A, B]).
func receiverName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return receiverName(x.X)
	case *ast.IndexExpr:
		return receiverName(x.X)
	case *ast.IndexListExpr:
		return receiverName(x.X)
	case *ast.Ident:
		return x.Name
	}
	return ""
}

// Os encadeáveis preservam o tipo concreto e produzem o mesmo efeito das
// funções genéricas.
func TestEncadeaveisEquivalemAsFuncoes(t *testing.T) {
	in := NewInput("x").Grow(2).AtEnd().Tooltip("dica").SetDisabled(true)
	if in.grow != 2 || in.cross != crossEnd || TooltipTextOf(in) != "dica" || !in.Disabled() {
		t.Fatalf("encadeáveis não aplicaram: grow=%d cross=%v tip=%q disabled=%v",
			in.grow, in.cross, TooltipTextOf(in), in.Disabled())
	}
	box := NewVBox().Centered()
	if box.cross != crossCenter {
		t.Fatal("VBox.Centered deveria marcar o alinhamento transversal")
	}
}
