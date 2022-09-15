package goast

import (
	"io"
	"io/fs"
)

type FailCase failCase
type EvalOutput struct {
	Fail []*FailCase `json:"fail"`
}

func WithCreateFunc(f func(path string) (io.WriteCloser, error)) Option {
	return func(g *Goast) {
		g.create = f
	}
}

func WithMkDirFunc(f func(path string, perm fs.FileMode) error) Option {
	return func(g *Goast) {
		g.mkdir = f
	}
}

func WithWalkFunc(f func(src string, cb func(fpath string, r io.Reader) error) error) Option {
	return func(g *Goast) {
		g.walk = f
	}
}
