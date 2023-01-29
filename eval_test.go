package goast_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opac"
)

const evalExampleCode = `package main

func Add(a, b int) int {
	return a + b
}
`

func TestEval(t *testing.T) {
	code := strings.NewReader(evalExampleCode)

	mock := opac.NewMock(func(in any) (any, error) {
		node := gt.Cast[*goast.Node](t, in)
		if node.Kind != "File" {
			return nil, nil
		}

		src := gt.Cast[*ast.File](t, node.Node)
		gt.A(t, src.Decls).Must().Length(1)
		f := gt.Cast[*ast.FuncDecl](t, src.Decls[0])
		gt.V(t, f.Name.Name).Equal("Add")

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

	fails := gt.R1(g.Eval("test.go", code)).NoError(t)

	gt.A(t, fails).Length(1)
	gt.V(t, fails[0].Line).Equal(3)
	gt.V(t, fails[0].Msg).Equal("eval_test")
}

func TestIgnoreAutoGeneratedFile(t *testing.T) {
	const code = `// Code generated by yo. DO NOT EDIT.
// Package model contains the types.
package main

func Add(a, b int) int {
	return a + b
}
`
	mock := opac.NewMock(func(in any) (any, error) {
		return &goast.EvalOutput{
			Fail: []*goast.FailCase{
				{
					Msg: "always fail",
					Pos: 15,
				},
			},
		}, nil
	})

	t.Run("with ignore option", func(t *testing.T) {
		g := goast.New(
			goast.WithOpacClient(mock),
			goast.WithIgnoreAutoGen(),
		)

		fails := gt.R1(
			g.Eval("test.go", strings.NewReader(code)),
		).NoError(t)
		gt.A(t, fails).Length(0)
	})

	t.Run("without ignore option", func(t *testing.T) {
		g := goast.New(
			goast.WithOpacClient(mock),
		)

		fails := gt.R1(g.Eval("test.go", strings.NewReader(code))).NoError(t)
		gt.A(t, fails).Longer(0)
	})
}
