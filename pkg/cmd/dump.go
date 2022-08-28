package cmd

import (
	"os"

	"github.com/m-mizutani/goast/pkg/usecase"
	"github.com/urfave/cli/v2"
)

func cmdDump() *cli.Command {
	var (
		outDir string
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
		},
		Action: func(c *cli.Context) error {
			codes := c.Args().Slice()
			if outDir != "" {
				if err := usecase.DumpDir(codes, outDir); err != nil {
					return err
				}
			} else {
				if err := usecase.DumpWriter(codes, os.Stdout); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
