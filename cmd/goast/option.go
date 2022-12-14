package main

import (
	"github.com/m-mizutani/goast"
	"github.com/urfave/cli/v2"
)

type inspectOptions struct {
	Lines       cli.IntSlice
	FuncNames   cli.StringSlice
	ObjectDepth int
	RootOnly    bool
	AllMatched  bool
}

func (x *inspectOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.IntSliceFlag{
			Name:        "line",
			Aliases:     []string{"l"},
			Usage:       "Line number condition for dump",
			Destination: &x.Lines,
		},
		&cli.StringSliceFlag{
			Name:        "func",
			Aliases:     []string{"n"},
			Usage:       "Function name condition for dump",
			Destination: &x.FuncNames,
		},
		&cli.IntFlag{
			Name:        "object-depth",
			Aliases:     []string{"d"},
			Usage:       "Recursion depth of *ast.Object",
			Destination: &x.ObjectDepth,
			Value:       0,
		},
		&cli.BoolFlag{
			Name:        "root-only",
			Aliases:     []string{"r"},
			Usage:       "Disable recursive inspection",
			Destination: &x.RootOnly,
		},
		&cli.BoolFlag{
			Name:        "all-matched",
			Aliases:     []string{"a"},
			Usage:       "Inspect all node matched with condition(s))",
			Destination: &x.AllMatched,
		},
	}
}

func (x *inspectOptions) Options() []goast.InspectOption {
	var opt []goast.InspectOption

	for _, v := range x.Lines.Value() {
		opt = append(opt, goast.WithLine(v))
	}
	for _, v := range x.FuncNames.Value() {
		opt = append(opt, goast.WithFuncName(v))
	}

	if x.ObjectDepth > 0 {
		opt = append(opt, goast.WithObjectDepth(x.ObjectDepth))
	}
	if x.RootOnly {
		opt = append(opt, goast.WithRootOnly())
	}
	if x.AllMatched {
		opt = append(opt, goast.WithAllMatched())
	}

	return opt
}
