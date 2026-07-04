package goast

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/m-mizutani/goerr/v2"
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

	// skipClone hands the raw ast.Node to the callback instead of a reflection
	// deep-copy. It is only safe when the file was parsed with
	// parser.SkipObjectResolution: clone() exists solely to strip the *ast.Object
	// graph (the one reference cycle in go/ast) so the node can be JSON-encoded,
	// and with object resolution skipped there is no such graph to strip.
	skipClone bool
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

	return inspect(f, fSet, cb, option)
}

// inspect is the shared body of Inspect and the fast path in Eval. Callers that
// pre-build inspectOptions (e.g. to decide the parser mode from ObjectDepth
// before parsing) use this directly to avoid rebuilding the options.
func inspect(f *ast.File, fSet *token.FileSet, cb callback, option *inspectOptions) error {
	ctx := newCloneContext()
	ctx.objectDepth = option.ObjectDepth

	pos := fSet.Position(f.Pos())
	filePath := pos.Filename

	// nodeFor produces the *Node handed to the callback: the raw node when
	// cloning is skipped, otherwise a stripped deep-copy.
	nodeFor := func(n ast.Node) (*Node, error) {
		if option.skipClone {
			return newNode(filePath, n, fSet), nil
		}
		obj := clone(ctx, reflect.ValueOf(n))
		output, ok := obj.Interface().(ast.Node)
		if !ok {
			return nil, goerr.New("consistency error, obj must be ast.Node")
		}
		return newNode(filePath, output, fSet), nil
	}

	if option.RootOnly {
		node, err := nodeFor(f)
		if err != nil {
			return err
		}
		return cb(node)
	}

	var cbErr error
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		if !option.shouldInspect(n, fSet) {
			return true
		}

		node, err := nodeFor(n)
		if err != nil {
			cbErr = err
			return false
		}

		if err := cb(node); err != nil {
			cbErr = err
			return false
		}

		return true
	})

	return cbErr
}
