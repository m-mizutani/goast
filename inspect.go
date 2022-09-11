package goast

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/m-mizutani/goerr"
)

type InspectOption func(options *inspectOptions)

func newInspectOptions() *inspectOptions {
	return &inspectOptions{
		Lines:     map[int]struct{}{},
		FuncNames: map[string]struct{}{},

		viewedLine: map[int]struct{}{},
	}
}

type inspectOptions struct {
	Lines       map[int]struct{}
	FuncNames   map[string]struct{}
	ObjectDepth int
	AllMatched  bool
	RootOnly    bool

	viewedLine map[int]struct{}
}

func (x *inspectOptions) shouldInspect(n ast.Node, fs *token.FileSet) bool {
	pos := fs.Position(n.Pos())
	if len(x.Lines) > 0 || len(x.FuncNames) > 0 {
		if _, ok := x.Lines[pos.Line]; ok {
			if _, ok := x.viewedLine[pos.Line]; !x.AllMatched && ok {
				return false
			}
			x.viewedLine[pos.Line] = struct{}{}
			return true
		}

		if f, ok := n.(*ast.FuncDecl); ok {
			if _, ok := x.FuncNames[f.Name.Name]; ok {
				return true
			}
		}

		return false
	}

	return true
}

func WithLine(line int) InspectOption {
	return func(opt *inspectOptions) {
		opt.Lines[line] = struct{}{}
	}
}

func WithFuncName(funcName string) InspectOption {
	return func(opt *inspectOptions) {
		opt.FuncNames[funcName] = struct{}{}
	}
}

func WithObjectDepth(depth int) InspectOption {
	return func(opt *inspectOptions) {
		opt.ObjectDepth = depth
	}
}

func WithRootOnly() InspectOption {
	return func(opt *inspectOptions) {
		opt.RootOnly = true
	}
}

func WithAllMatched() InspectOption {
	return func(opt *inspectOptions) {
		opt.AllMatched = true
	}
}

type callback func(data *Node) error

func Inspect(f *ast.File, fSet *token.FileSet, cb callback, options ...InspectOption) error {
	option := newInspectOptions()
	for _, f := range options {
		f(option)
	}

	ctx := newCloneContext()
	ctx.objectDepth = option.ObjectDepth

	pos := fSet.Position(f.Pos())
	filePath := pos.Filename

	if option.RootOnly {
		obj := clone(ctx, reflect.ValueOf(f))
		output, ok := obj.Interface().(ast.Node)
		if !ok {
			return goerr.New("consistency error, obj must be *ast.File")
		}

		node := newNode(filePath, output, fSet)

		if err := cb(node); err != nil {
			return err
		}
	} else {
		var cbErr error
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			if !option.shouldInspect(n, fSet) {
				return true
			}

			obj := clone(ctx, reflect.ValueOf(n))
			output, ok := obj.Interface().(ast.Node)
			if !ok {
				cbErr = goerr.New("consistency error, obj must be *ast.File")
				return false
			}

			node := newNode(filePath, output, fSet)

			if err := cb(node); err != nil {
				cbErr = err
				return false
			}

			return true
		})

		if cbErr != nil {
			return cbErr
		}
	}

	return nil
}
