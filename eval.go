package goast

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"regexp"

	"github.com/m-mizutani/goerr/v2"
)

// evalQuery is the Rego query Eval runs against the loaded policies. It targets
// the "fail" rule of the "goast" package directly, rather than the whole
// "data.goast" document. Querying the package document would force OPA to
// materialize every rule in the package on every AST node — helper rules,
// constant sets, and, crucially, the test rules that live in same-package
// *_test.rego files. On a large tree that is millions of pointless rule
// evaluations. Since Eval only ever consumes the "fail" set, asking for exactly
// that keeps behavior identical while cutting the per-node cost by orders of
// magnitude.
const evalQuery = "data.goast.fail"

type failCase struct {
	Msg string `json:"msg"`
	Pos int    `json:"pos"`
	Sev string `json:"sev"`
}

var generatedCodePattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGeneratedFile(file *ast.File) bool {
	for _, comment := range file.Comments {
		for _, row := range comment.List {
			if generatedCodePattern.MatchString(row.Text) {
				return true
			}
		}
	}
	return false
}

func (x *Goast) Eval(filePath string, r io.Reader) ([]*Fail, error) {
	var fails []*Fail

	callback := func(data *Node) error {
		ctx := context.Background()
		var out []*failCase
		if err := x.engine.Query(ctx, evalQuery, data, &out); err != nil {
			return goerr.Wrap(err, "failed to evaluate policy")
		}

		for _, fail := range out {
			pos := data.Pos(token.Pos(fail.Pos))

			fails = append(fails, &Fail{
				Path:   data.Path,
				Line:   pos.Line,
				Column: pos.Column,
				Msg:    fail.Msg,
				Sev:    fail.Sev,
			})
		}
		return nil
	}

	option := newInspectOptions()
	for _, o := range x.inspectOpt {
		o(option)
	}

	mode := parser.Mode(parser.ParseComments)
	if option.ObjectDepth <= 0 {
		// Nothing downstream needs the *ast.Object graph (that is exactly what
		// ObjectDepth > 0 would request), so skip object resolution. This removes
		// go/ast's only reference cycle, which in turn lets inspect hand raw nodes
		// to the policy instead of a per-node reflection deep-copy. On a large
		// tree the copy is a meaningful share of the work.
		mode |= parser.SkipObjectResolution
		option.skipClone = true
	}

	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, filePath, r, mode)
	if err != nil {
		return nil, err
	}

	if x.ignoreAutoGen && isGeneratedFile(f) {
		return nil, nil
	}

	if err := inspect(f, fSet, callback, option); err != nil {
		return nil, err
	}

	return fails, nil
}
