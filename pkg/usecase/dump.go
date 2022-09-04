package usecase

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/utils"
	"github.com/m-mizutani/goerr"
)

func DumpDir(codes []string, outDir string) error {
	callback := func(data *model.Target) error {
		if data.Kind != "File" {
			return nil
		}

		codeDir := filepath.Dir(data.Path)
		dir := filepath.Join(outDir, codeDir)

		// #nosec
		if err := os.MkdirAll(dir, 0755); err != nil {
			return goerr.Wrap(err)
		}

		outPath := filepath.Join(dir, filepath.Base(data.Path)+".json")
		fd, err := os.Create(filepath.Clean(outPath))
		if err != nil {
			return err
		}
		defer func() {
			if err := fd.Close(); err != nil {
				utils.Logger().Warn("failed to close dump file: %s", err)
			}
		}()

		enc := json.NewEncoder(fd)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			return goerr.Wrap(err)
		}

		return nil
	}

	return walkCode(codes, callback)
}

type DumpOption func(opt *dumpOption)

type dumpOption struct {
	Lines     map[int]struct{}
	FuncNames map[string]struct{}
}

func WithDumpLine(line int) DumpOption {
	return func(opt *dumpOption) {
		opt.Lines[line] = struct{}{}
	}
}

func WithDumpFuncName(funcName string) DumpOption {
	return func(opt *dumpOption) {
		opt.FuncNames[funcName] = struct{}{}
	}
}

func DumpWriter(codes []string, w io.Writer, options ...DumpOption) error {
	opt := &dumpOption{
		Lines:     make(map[int]struct{}),
		FuncNames: make(map[string]struct{}),
	}
	for _, f := range options {
		f(opt)
	}

	dump := func(path string, data *model.Target) error {
		raw, err := json.Marshal(data)
		if err != nil {
			return goerr.Wrap(err)
		}

		fmt.Fprintln(w, string(raw))
		return nil
	}

	if len(opt.Lines) == 0 && len(opt.FuncNames) == 0 {
		return walkCode(codes, func(data *model.Target) error {
			if data.Kind != "File" {
				return nil
			}

			return dump(data.Path, data)
		})
	} else {
		visitedLine := map[int]struct{}{} // For avoiding to display children
		return walkCode(codes, func(data *model.Target) error {
			pos := data.Pos(data.Node.Pos())
			if _, ok := opt.Lines[pos.Line]; ok {
				if _, visited := visitedLine[pos.Line]; !visited {
					visitedLine[pos.Line] = struct{}{}
					return dump(data.Path, data)
				}
			}

			if decl, ok := data.Node.(*ast.FuncDecl); ok {
				if _, ok := opt.FuncNames[decl.Name.Name]; ok {
					return dump(data.Path, data)
				}
			}

			return nil
		})
	}
}
