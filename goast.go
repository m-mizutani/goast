package goast

import (
	"io"

	"github.com/m-mizutani/opac"
)

type Goast struct {
	opac       opac.Client
	inspectOpt []InspectOption

	dumpHook DumpHook
}

type Option func(g *Goast)

func New(options ...Option) *Goast {
	g := &Goast{}
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
