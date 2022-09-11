package goast

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"

	"github.com/m-mizutani/goerr"
)

func (x *Goast) Dump(filePath string, r io.Reader, w io.Writer) error {
	dump := func(data *Node) error {
		if x.dumpHook == nil {
			raw, err := json.Marshal(data)
			if err != nil {
				return goerr.Wrap(err)
			}

			fmt.Fprintln(w, string(raw))
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
