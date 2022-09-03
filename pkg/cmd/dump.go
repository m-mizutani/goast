package cmd

import (
	"os"

	"github.com/m-mizutani/goast/pkg/usecase"
	"github.com/urfave/cli/v2"
)

func cmdDump() *cli.Command {
	var (
		outDir    string
		lines     cli.IntSlice
		funcNames cli.StringSlice
	)

	return &cli.Command{
		Name:    "dump",
		Usage:   "output go codes as AST",
		Aliases: []string{"d"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Aliases:     []string{"d"},
				Destination: &outDir,
				Usage:       "Output directory to dump *.go as *.go.json",
			},
			&cli.IntSliceFlag{
				Name:        "line",
				Aliases:     []string{"l"},
				Destination: &lines,
				Usage:       "Line number condition for dump",
			},
			&cli.StringSliceFlag{
				Name:        "func",
				Aliases:     []string{"f"},
				Destination: &funcNames,
				Usage:       "Function name condition for dump",
			},
		},
		Action: func(c *cli.Context) error {
			codes := c.Args().Slice()
			if outDir != "" {
				if err := usecase.DumpDir(codes, outDir); err != nil {
					return err
				}
			} else {
				var options []usecase.DumpOption
				for _, line := range lines.Value() {
					options = append(options, usecase.WithDumpLine(line))
				}
				for _, funcName := range funcNames.Value() {
					options = append(options, usecase.WithDumpFuncName(funcName))
				}

				if err := usecase.DumpWriter(codes, os.Stdout, options...); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
