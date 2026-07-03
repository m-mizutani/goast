package goast

import (
	"context"
	"io"
	"io/fs"
	"os"
)

// policyEngine abstracts the Rego evaluation so that Eval stays testable without
// standing up a real OPA evaluation. *PolicyEngine is the production implementation.
type policyEngine interface {
	Query(ctx context.Context, query string, input, output any) error
}

type Goast struct {
	engine     policyEngine
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

func WithPolicyEngine(engine policyEngine) Option {
	return func(g *Goast) {
		g.engine = engine
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
