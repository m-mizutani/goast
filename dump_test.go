package goast_test

import (
	"go/ast"
	"io"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			goast.WithLine(7),         // does not work
			goast.WithFuncName("Add"), // does not work
		),
		goast.WithDumpHook(func(node *goast.Node, w io.Writer) error {
			called++
			assert.Equal(t, "File", node.Kind)
			return nil
		}),
	)

	require.NoError(t, g.Dump("test.go", code, nil))
	assert.Equal(t, 1, called)
}

func TestDumpLine(t *testing.T) {
	code := strings.NewReader(dumpExampleCode)

	g := goast.New(
		goast.WithInspectOptions(
			goast.WithLine(7),
			goast.WithWalk(),
		),
		goast.WithDumpHook(func(node *goast.Node, w io.Writer) error {
			assert.Equal(t, "FuncDecl", node.Kind)
			f, ok := node.Node.(*ast.FuncDecl)
			require.True(t, ok)
			assert.Equal(t, "Sub", f.Name.Name)
			assert.Len(t, f.Body.List, 2)
			return nil
		}),
	)
	require.NoError(t, g.Dump("test.go", code, nil))
}

func TestDumpLineAllNode(t *testing.T) {
	code := strings.NewReader(dumpExampleCode)
	var cnt int
	g := goast.New(
		goast.WithInspectOptions(
			goast.WithLine(8),
			goast.WithAllMatched(),
			goast.WithWalk(),
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
			assert.Equal(t, kinds[cnt], node.Kind)
			cnt++
			return nil
		}),
	)
	require.NoError(t, g.Dump("test.go", code, nil))
	assert.Equal(t, 5, cnt)
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
