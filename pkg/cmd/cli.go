// Package ast declares the types used to represent syntax trees for Go
// packages.
package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast/pkg/utils"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zlog"
	"github.com/urfave/cli/v2"
)

var logger = zlog.New()

func Run(args []string) error {
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
						utils.Logger().Warn("failed to close a log file: %s", err)
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

			utils.InitLogger(options...)

			return nil
		},
	}

	if err := app.Run(args); err != nil {
		logger.Err(err).Error("exit with error")
		return err
	}

	return nil
}
