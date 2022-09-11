package goast

import (
	"go/ast"
	"go/token"
	"reflect"
)

type Node struct {
	Path string
	Node ast.Node
	Kind string

	fileSet *token.FileSet
}

func newNode(fpath string, node ast.Node, fset *token.FileSet) *Node {
	return &Node{
		Path:    fpath,
		Node:    node,
		Kind:    reflect.ValueOf(node).Elem().Type().Name(),
		fileSet: fset,
	}
}

func (x *Node) Pos(p token.Pos) token.Position {
	return x.fileSet.Position(p)
}

type Fail struct {
	Path   string
	Line   int
	Column int
	Msg    string
	Sev    string
}
