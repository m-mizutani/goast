package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/usecase"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/opac"
	"github.com/urfave/cli/v2"
)

func cmdEval() *cli.Command {
	var (
		policies cli.StringSlice
		format   string
		output   string
	)

	return &cli.Command{
		Name:    "eval",
		Usage:   "inspect and check Go code with Rego policy",
		Aliases: []string{"e"},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "policy",
				Aliases:     []string{"p"},
				Destination: &policies,
				Usage:       "Policy directory or file",
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "format",
				Aliases:     []string{"f"},
				Destination: &format,
				Usage:       "Output format [text|json]",
				Value:       "text",
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Destination: &output,
				Usage:       "Output file. '-' means stdout",
				Value:       "-",
			},
		},
		Action: func(c *cli.Context) error {
			files := c.Args().Slice()

			// format
			f, ok := model.ToOutputFormat(format)
			if !ok {
				return goerr.New("unsupported output format").With("format", format)
			}

			// output
			var out io.Writer
			if output == "-" {
				out = os.Stdout
			} else {
				fd, err := os.Create(filepath.Clean(output))
				if err != nil {
					return goerr.Wrap(err, "failed to open output file")
				}
				defer func() {
					if err := fd.Close(); err != nil {
						logger.Err(err).Warn("failed to close output file")
					}
				}()
				out = fd
			}

			// policy
			opacOpt := []opac.LocalOption{opac.WithPackage("goast")}
			for _, policy := range policies.Value() {
				opacOpt = append(opacOpt, opac.WithDir(policy))
			}
			client, err := opac.NewLocal(opacOpt...)
			if err != nil {
				return goerr.Wrap(err)
			}

			// validate
			if err := usecase.Eval(client, files, out, f); err != nil {
				return err
			}
			return nil
		},
	}
}
