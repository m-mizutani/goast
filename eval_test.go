package goast_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/opac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const evalExampleCode = `package main

func Add(a, b int) int {
	return a + b
}
`

func TestEval(t *testing.T) {
	code := strings.NewReader(evalExampleCode)

	mock := opac.NewMock(func(in any) (any, error) {
		node, ok := in.(*goast.Node)
		require.True(t, ok)
		if node.Kind != "File" {
			return nil, nil
		}

		src, ok := node.Node.(*ast.File)
		require.True(t, ok)
		require.Len(t, src.Decls, 1)
		f, ok := src.Decls[0].(*ast.FuncDecl)
		require.True(t, ok)

		assert.Equal(t, "Add", f.Name.Name)

		return &goast.EvalOutput{
			Fail: []*goast.FailCase{
				{
					Msg: "eval_test",
					Pos: 15,
				},
			},
		}, nil
	})

	g := goast.New(
		goast.WithOpacClient(mock),
	)

	fails, err := g.Eval("test.go", code)
	require.NoError(t, err)

	require.Len(t, fails, 1)
	assert.Equal(t, 3, fails[0].Line)
	assert.Equal(t, "eval_test", fails[0].Msg)
}
