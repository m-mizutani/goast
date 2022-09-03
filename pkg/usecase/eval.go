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

	callback := func(path string, data *model.Target) error {
		ctx := context.Background()
		var out model.EvalOutput
		if err := client.Query(ctx, data, &out); err != nil {
			return goerr.Wrap(err)
		}

		for _, fail := range out.Fail {
			pos := data.Pos(token.Pos(fail.Pos))

			switch format {
			case model.OutputText:
				fmt.Fprintf(w, "[%s:%d] - %s\n", path, pos.Line, fail.Msg)

			case model.OutputJSON:
				diagnosis.Diagnostics = append(diagnosis.Diagnostics, &rdf.Diagnostic{
					Message: fail.Msg,
					Location: &rdf.Location{
						Path: path,
						Range: &rdf.Range{
							Start: &rdf.Position{
								Line:   int32(pos.Line),
								Column: int32(pos.Column),
							},
						},
					},
				})
			}
		}
		return nil
	}

	if err := walkCode(codes, callback); err != nil {
		return err
	}

	switch format {
	case model.OutputText: // nothing to do
	case model.OutputJSON:
		if err := json.NewEncoder(w).Encode(diagnosis); err != nil {
			return goerr.Wrap(err, "failed to convert DiagnosticResult")
		}
	}

	return nil
}
