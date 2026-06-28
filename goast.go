package goast

import (
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/m-mizutani/opac"
)

// policyEngine is the subset of *opac.Client that Goast depends on. opac v0.2+
// exposes Client as a concrete struct, so an interface is declared here to keep
// Eval testable without standing up a real OPA evaluation.
type policyEngine interface {
	Query(ctx context.Context, query string, input, output any, options ...opac.QueryOption) error
}

type Goast struct {
	opac       policyEngine
	inspectOpt []InspectOption

	create func(path string) (io.WriteCloser, error)
	mkdir  func(path string, perm fs.FileMode) error
	walk   func(src string, cb func(fpath string, r io.Reader) error) error

	dumpCompact bool
	dumpHook    DumpHook

	ignoreAutoGen bool
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

func WithOpacClient(client policyEngine) Option {
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

func WithCompact(enable bool) Option {
	return func(g *Goast) {
		g.dumpCompact = enable
	}
}

func WithIgnoreAutoGen() Option {
	return func(g *Goast) {
		g.ignoreAutoGen = true
	}
}
