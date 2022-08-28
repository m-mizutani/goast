package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goerr"
)

func Import(filePath string) (*model.File, error) {
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, goerr.Wrap(err)
	}

	obj := clone(reflect.ValueOf(f))
	output, ok := obj.Interface().(*ast.File)
	if !ok {
		return nil, goerr.New("consistency error")
	}

	return model.NewFile(filePath, output, fileSet), nil
}
