package goast_test

import (
	"go/ast"
	"io"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

const dumpExampleCode = `package main

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	d := a - b
	return d
}
`

func TestDumpFile(t *testing.T) {
	code := strings.NewReader(dumpExampleCode)

	var called int
	g := goast.New(
		goast.WithInspectOptions(
			goast.WithRootOnly(),
			goast.WithLine(7),         // does not work
			goast.WithFuncName("Add"), // does not work
		),
		goast.WithDumpHook(func(node *goast.Node, w io.Writer) error {
			called++
			gt.Value(t, node.Kind).Equal("File")
			return nil
		}),
	)

	gt.NoError(t, g.Dump("test.go", code, nil)).Must()
	gt.Value(t, called).Equal(1)
}

func TestDumpLine(t *testing.T) {
	code := strings.NewReader(dumpExampleCode)

	g := goast.New(
		goast.WithInspectOptions(
			goast.WithLine(7),
		),
		goast.WithDumpHook(func(node *goast.Node, w io.Writer) error {
			gt.V(t, node.Kind).Equal("FuncDecl")

			f := gt.Cast[*ast.FuncDecl](t, node.Node)
			gt.V(t, f.Name.Name).Equal("Sub")
			gt.A(t, f.Body.List).Length(2)
			return nil
		}),
	)
	gt.NoError(t, g.Dump("test.go", code, nil)).Must()
}

func TestDumpLineAllNode(t *testing.T) {
	code := strings.NewReader(dumpExampleCode)
	var cnt int
	g := goast.New(
		goast.WithInspectOptions(
			goast.WithLine(8),
			goast.WithAllMatched(),
		),
		goast.WithDumpHook(func(node *goast.Node, w io.Writer) error {
			// nodes of `d := a - b`
			kinds := []string{
				"AssignStmt",
				"Ident",
				"BinaryExpr",
				"Ident",
				"Ident",
			}
			gt.V(t, node.Kind).Equal(kinds[cnt])
			cnt++
			return nil
		}),
	)
	gt.NoError(t, g.Dump("test.go", code, nil)).Must()
	gt.V(t, cnt).Equal(5)
}

/*
func TestStructAccess(t *testing.T) {
	codePath := createTestCode(t, `package main

	import "model"

	func main() {
		var u0 *model.User
		u1 := &model.User{}
		u2 := model.User{}
	}
	`)

	var w bytes.Buffer
	require.NoError(t, usecase.DumpWriter([]string{codePath}, &w, usecase.WithDumpLine(7)))
}
*/
