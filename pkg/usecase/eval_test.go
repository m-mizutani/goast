package usecase_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"os"
	"testing"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/usecase"
	"github.com/m-mizutani/opac"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCode(t *testing.T, code string) string {
	fd, err := os.CreateTemp("", "*.go")
	require.NoError(t, err)
	_, err = fd.WriteString(code)
	require.NoError(t, err)
	require.NoError(t, fd.Close())

	t.Cleanup(func() {
		require.NoError(t, os.Remove(fd.Name()))
	})

	return fd.Name()
}

func TestEval(t *testing.T) {
	codePath := createTestCode(t, `package main

	func Add(a, b int) int {
		return a + b
	}`)

	mock := opac.NewMock(func(in any) (any, error) {
		node, ok := in.(*model.Target)
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

		return &model.EvalOutput{
			Fail: []*model.EvalFail{
				{
					Msg: "eval_test",
					Pos: 15,
				},
			},
		}, nil
	})

	type testCase struct {
		format   model.OutputFormat
		testFunc func(t *testing.T, output []byte)
	}

	testCases := map[string]testCase{
		"json": {
			format: model.OutputJSON,
			testFunc: func(t *testing.T, output []byte) {
				var result rdf.DiagnosticResult
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, "goast", result.Source.Name)
				assert.Equal(t, "https://github.com/m-mizutani/goast", result.Source.Url)

				require.Len(t, result.Diagnostics, 1)
				d := result.Diagnostics[0]
				assert.Equal(t, "eval_test", d.Message)
				assert.Contains(t, d.Location.Path, ".go")
				assert.Equal(t, d.Location.Range.Start.Column, int32(1))
				assert.Equal(t, d.Location.Range.Start.Line, int32(3))
			},
		},
		"text": {
			format: model.OutputText,
			testFunc: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "["+codePath+":3] - eval_test")
			},
		},
	}

	for title, tc := range testCases {
		t.Run(title, func(t *testing.T) {
			var w bytes.Buffer
			require.NoError(t, usecase.Eval(mock, []string{codePath}, &w, tc.format))
			tc.testFunc(t, w.Bytes())
		})
	}
}

func TestEvalPolicy(t *testing.T) {
	client, err := opac.NewLocal(opac.WithDir("../../policy/"))
	require.NoError(t, err)

	var w bytes.Buffer
	require.NoError(t, usecase.Eval(client, []string{"../../examples/main.go"}, &w, model.OutputText))
}

func TestNestedCode(t *testing.T) {
	codePath := createTestCode(t, `package main
	import "fmt"

	func Add(a, b int) {
		for i := 0; i < a; i++ {
			for j := 0; j < b; j++ {
				fmt.Println(i * j)
			}
		}
	}`)

	mock := opac.NewMock(func(in any) (any, error) {
		return &model.EvalOutput{
			Fail: []*model.EvalFail{},
		}, nil
	})

	var w bytes.Buffer
	require.NoError(t, usecase.Eval(mock, []string{codePath}, &w, model.OutputJSON))
}
