package goast

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io"

	"github.com/m-mizutani/goerr"
)

func (x *Goast) Dump(filePath string, r io.Reader, w io.Writer) error {
	encoder := json.NewEncoder(w)
	if !x.dumpCompact {
		encoder.SetIndent("", "  ")
	}

	dump := func(data *Node) error {
		if x.dumpHook == nil {
			if err := encoder.Encode(data); err != nil {
				return goerr.Wrap(err, "failed to encode dump data")
			}
			return nil
		} else {
			return x.dumpHook(data, w)
		}
	}

	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, filePath, r, parser.ParseComments)
	if err != nil {
		return goerr.Wrap(err)
	}

	if err := Inspect(f, fSet, dump, x.inspectOpt...); err != nil {
		return err
	}

	return nil
}
