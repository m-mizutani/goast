package model

import (
	"go/ast"
	"go/token"
	"reflect"
)

type Target struct {
	Path string
	Node ast.Node
	Kind string

	fileSet *token.FileSet
}

func NewTarget(fpath string, node ast.Node, fset *token.FileSet) *Target {
	return &Target{
		Path:    fpath,
		Node:    node,
		Kind:    reflect.ValueOf(node).Elem().Type().Name(),
		fileSet: fset,
	}
}

func (x *Target) Pos(p token.Pos) token.Position {
	return x.fileSet.Position(p)
}

type EvalFail struct {
	Msg string `json:"msg"`
	Pos int    `json:"pos"`
	Sev string `json:"sev"`
}

type EvalOutput struct {
	Fail []*EvalFail `json:"fail"`
}
