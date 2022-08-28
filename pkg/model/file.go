package model

import (
	"go/ast"
	"go/token"
)

type File struct {
	FilePath string
	Source   *ast.File

	fileSet *token.FileSet
}

func NewFile(fpath string, src *ast.File, fset *token.FileSet) *File {
	return &File{
		FilePath: fpath,
		Source:   src,
		fileSet:  fset,
	}
}

func (x *File) Pos(p token.Pos) token.Position {
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
