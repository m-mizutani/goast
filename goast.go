package goast

import (
	"io"
	"io/fs"
	"os"

	"github.com/m-mizutani/opac"
)

type Goast struct {
	opac       opac.Client
	inspectOpt []InspectOption

	create func(path string) (io.WriteCloser, error)
	mkdir  func(path string, perm fs.FileMode) error
	walk   func(src string, cb func(fpath string, r io.Reader) error) error

	dumpHook DumpHook
}

type Option func(g *Goast)

func New(options ...Option) *Goast {
	g := &Goast{
		create: func(path string) (io.WriteCloser, error) {
			return os.Create(path)
		},
		mkdir: os.MkdirAll,
		walk:  walkGoCode,
	}
	for _, opt := range options {
		opt(g)
	}
	return g
}

func WithOpacClient(client opac.Client) Option {
	return func(g *Goast) {
		g.opac = client
	}
}

func WithInspectOptions(options ...InspectOption) Option {
	return func(g *Goast) {
		g.inspectOpt = options
	}
}

func WithDumpHook(hook DumpHook) Option {
	return func(g *Goast) {
		g.dumpHook = hook
	}
}

type DumpHook func(node *Node, w io.Writer) error
