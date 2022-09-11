package goast

import (
	"context"
	"go/parser"
	"go/token"
	"io"

	"github.com/m-mizutani/goerr"
)

type failCase struct {
	Msg string `json:"msg"`
	Pos int    `json:"pos"`
	Sev string `json:"sev"`
}

type evalOutput struct {
	Fail []*failCase `json:"fail"`
}

func (x *Goast) Eval(filePath string, r io.Reader) ([]*Fail, error) {
	var fails []*Fail

	callback := func(data *Node) error {
		ctx := context.Background()
		var out evalOutput
		if err := x.opac.Query(ctx, data, &out); err != nil {
			return goerr.Wrap(err)
		}

		for _, fail := range out.Fail {
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

	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, filePath, r, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	if err := Inspect(f, fSet, callback, x.inspectOpt...); err != nil {
		return nil, err
	}

	return fails, nil
}
