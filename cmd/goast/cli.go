package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/opac"
	"github.com/m-mizutani/zlog"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/urfave/cli/v2"
)

var logger = zlog.New()

func run(args []string) error {
	var (
		logLevel  string
		logFormat string
		logOutput string
	)

	app := cli.App{
		Name: "goast",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "[debug|info|warn|error]",
				Aliases:     []string{"l"},
				EnvVars:     []string{"GOAST_LOG_LEVEL"},
				Value:       "info",
				Destination: &logLevel,
			},
			&cli.StringFlag{
				Name:        "log-format",
				Usage:       "[text|json]",
				EnvVars:     []string{"GOAST_LOG_FORMAT"},
				Value:       "text",
				Destination: &logFormat,
			},
			&cli.StringFlag{
				Name:        "log-output",
				Usage:       "[stdout|stderr|$FILENAME]",
				EnvVars:     []string{"GOAST_LOG_OUTPUT"},
				Value:       "stderr",
				Destination: &logOutput,
			},
		},
		Commands: []*cli.Command{
			cmdEval(),
			cmdDump(),
		},
		Before: func(c *cli.Context) error {
			options := []zlog.Option{
				zlog.WithLogLevel(logLevel),
			}

			var w io.Writer
			switch logOutput {
			case "stdout":
				w = os.Stdout
			case "stderr":
				w = os.Stderr
			default:
				logFile, err := os.Create(filepath.Clean(logOutput))
				if err != nil {
					return goerr.Wrap(err)
				}
				defer func() {
					if err := logFile.Close(); err != nil {
						goast.Logger().Warn("failed to close a log file: %s", err)
					}
				}()
			}

			switch logFormat {
			case "text":
				options = append(options, zlog.WithEmitter(
					zlog.NewConsoleEmitter(zlog.ConsoleWriter(w)),
				))

			case "json":
				options = append(options, zlog.WithEmitter(
					zlog.NewJsonEmitter(zlog.JsonWriter(w)),
				))

			default:
				return goerr.New("unsupported log format: " + logFormat)
			}

			goast.RenewLogger(options)

			return nil
		},
	}

	if err := app.Run(args); err != nil {
		logger.Err(err).Error("exit with error")
		return err
	}

	return nil
}

func cmdDump() *cli.Command {
	var opt inspectOptions

	return &cli.Command{
		Name:    "dump",
		Usage:   "output go codes as AST",
		Aliases: []string{"d"},
		Flags:   opt.Flags(),
		Action: func(c *cli.Context) error {
			codes := c.Args().Slice()

			g := goast.New(
				goast.WithInspectOptions(opt.Options()...),
			)
			if err := walkCode(codes, func(filePath string, r io.Reader) error {
				return g.Dump(filePath, r, os.Stdout)
			}); err != nil {
				return err
			}

			return nil
		},
	}
}

func cmdEval() *cli.Command {
	var (
		policies cli.StringSlice
		format   string
		output   string
		opt      inspectOptions
	)

	return &cli.Command{
		Name:    "eval",
		Usage:   "inspect and check Go code with Rego policy",
		Aliases: []string{"e"},
		Flags: append([]cli.Flag{
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
		}, opt.Flags()...),
		Action: func(c *cli.Context) error {
			files := c.Args().Slice()

			// format
			f, ok := toOutputFormat(format)
			if !ok {
				return goerr.New("unsupported output format").With("format", format)
			}

			// output
			var w io.Writer
			if output == "-" {
				w = os.Stdout
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
				w = fd
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

			g := goast.New(
				goast.WithOpacClient(client),
				goast.WithInspectOptions(opt.Options()...),
			)

			var failCases []*goast.Fail

			if err := walkCode(files, func(filePath string, r io.Reader) error {
				resp, err := g.Eval(filePath, r)
				if err != nil {
					return err
				}
				failCases = append(failCases, resp...)
				return nil
			}); err != nil {
				return err
			}

			switch f {
			case outputText:
				for _, fail := range failCases {
					fmt.Fprintf(w, "[%s:%d] - %s\n", fail.Path, fail.Line, fail.Msg)
				}
				fmt.Fprintf(w, "\n\tDetected %d violations\n\n", len(failCases))

			case outputJSON:
				diagnosis := &rdf.DiagnosticResult{
					Source: &rdf.Source{
						Name: "goast",
						Url:  "https://github.com/m-mizutani/goast",
					},
				}

				for _, fail := range failCases {
					diagnosis.Diagnostics = append(diagnosis.Diagnostics, &rdf.Diagnostic{
						Message: fail.Msg,
						Location: &rdf.Location{
							Path: fail.Path,
							Range: &rdf.Range{
								Start: &rdf.Position{
									Line:   int32(fail.Line),
									Column: int32(fail.Column),
								},
							},
						},
					})
				}

				if err := json.NewEncoder(w).Encode(diagnosis); err != nil {
					return goerr.Wrap(err, "failed to convert DiagnosticResult")
				}
			}

			return nil
		},
	}
}
