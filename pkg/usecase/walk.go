package usecase

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/utils"
	"github.com/m-mizutani/goerr"
)

type walkCallback func(path string, data *model.Target) error

func walkCode(codes []string, callback walkCallback) error {
	for _, codePath := range codes {
		if err := filepath.WalkDir(codePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return goerr.Wrap(err)
			}
			if d.IsDir() {
				return nil
			}

			fpath := filepath.Clean(path)
			if filepath.Ext(fpath) != ".go" {
				return nil
			}

			utils.Logger().With("file", fpath).Debug("loading file")

			fileSet := token.NewFileSet()
			f, err := parser.ParseFile(fileSet, fpath, nil, parser.ParseComments)
			if err != nil {
				return goerr.Wrap(err)
			}

			/*
				obj := clone(reflect.ValueOf(f))
				output, ok := obj.Interface().(ast.Node)
				if !ok {
					return goerr.New("consistency error, obj must be *ast.File")
				}

				node := model.NewTarget(fpath, output, fileSet)

				if err := callback(fpath, node); err != nil {
					return err
				}
			*/

			var cbErr error
			ast.Inspect(f, func(n ast.Node) bool {
				if n == nil {
					return true
				}

				obj := clone(reflect.ValueOf(n))
				output, ok := obj.Interface().(ast.Node)
				if !ok {
					cbErr = goerr.New("consistency error, obj must be *ast.File")
					return false
				}

				node := model.NewTarget(fpath, output, fileSet)

				if err := callback(fpath, node); err != nil {
					cbErr = err
					return false
				}

				return true
			})

			if cbErr != nil {
				return cbErr
			}

			return nil
		}); err != nil {
			return goerr.Wrap(err)
		}
	}

	return nil
}
