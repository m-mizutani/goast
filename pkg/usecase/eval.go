package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"io"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/opac"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func Eval(client opac.Client, codes []string, w io.Writer, format model.OutputFormat) error {
	diagnosis := &rdf.DiagnosticResult{
		Source: &rdf.Source{
			Name: "goast",
			Url:  "https://github.com/m-mizutani/goast",
		},
	}

	type failCase struct {
		path   string
		line   int
		column int
		msg    string
		sev    string
	}
	var failCases []failCase

	callback := func(data *model.Target) error {
		ctx := context.Background()
		var out model.EvalOutput
		if err := client.Query(ctx, data, &out); err != nil {
			return goerr.Wrap(err)
		}

		for _, fail := range out.Fail {
			pos := data.Pos(token.Pos(fail.Pos))

			failCases = append(failCases, failCase{
				path:   data.Path,
				line:   pos.Line,
				column: pos.Column,
				msg:    fail.Msg,
				sev:    fail.Sev,
			})
		}
		return nil
	}

	if err := walkCode(codes, callback); err != nil {
		return err
	}

	switch format {
	case model.OutputText:
		for _, fail := range failCases {
			fmt.Fprintf(w, "[%s:%d] - %s\n", fail.path, fail.line, fail.msg)
		}

		fmt.Fprintf(w, "\n\tDetected %d violations\n\n", len(failCases))

	case model.OutputJSON:
		for _, fail := range failCases {
			diagnosis.Diagnostics = append(diagnosis.Diagnostics, &rdf.Diagnostic{
				Message: fail.msg,
				Location: &rdf.Location{
					Path: fail.path,
					Range: &rdf.Range{
						Start: &rdf.Position{
							Line:   int32(fail.line),
							Column: int32(fail.column),
						},
					},
				},
			})
		}

		if err := json.NewEncoder(w).Encode(diagnosis); err != nil {
			return goerr.Wrap(err, "failed to convert DiagnosticResult")
		}
	}

	return nil
}
