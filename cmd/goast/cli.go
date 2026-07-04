package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

var logger = slog.Default()

func run(args []string) error {
	var (
		logLevel  string
		logFormat string
		logOutput string
	)

	cmd := &cli.Command{
		Name:    "goast",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "[debug|info|warn|error]",
				Aliases:     []string{"l"},
				Sources:     cli.EnvVars("GOAST_LOG_LEVEL"),
				Value:       "info",
				Destination: &logLevel,
			},
			&cli.StringFlag{
				Name:        "log-format",
				Usage:       "[text|json]",
				Sources:     cli.EnvVars("GOAST_LOG_FORMAT"),
				Value:       "text",
				Destination: &logFormat,
			},
			&cli.StringFlag{
				Name:        "log-output",
				Usage:       "[stdout|stderr|$FILENAME]",
				Sources:     cli.EnvVars("GOAST_LOG_OUTPUT"),
				Value:       "stderr",
				Destination: &logOutput,
			},
		},
		Commands: []*cli.Command{
			cmdEval(),
			cmdDump(),
			cmdSync(),
			cmdTest(),
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			level, err := parseLogLevel(logLevel)
			if err != nil {
				return ctx, err
			}

			var w io.Writer
			switch logOutput {
			case "stdout":
				w = os.Stdout
			case "stderr":
				w = os.Stderr
			default:
				// The log file must stay open for the whole command run, so it
				// is not closed here: Before returns before Action executes, and
				// closing it now would break logging during the command. The OS
				// reclaims the descriptor when the process exits.
				logFile, err := os.Create(filepath.Clean(logOutput))
				if err != nil {
					return ctx, goerr.Wrap(err, "failed to create log file", goerr.V("path", logOutput))
				}
				w = logFile
			}

			l, err := newLogger(w, level, logFormat)
			if err != nil {
				return ctx, err
			}

			logger = l
			goast.SetLogger(l)

			return ctx, nil
		},
	}

	if err := cmd.Run(context.Background(), args); err != nil {
		// Error carries the goerr values (expanded by the GoErr hook) for a
		// concise report; the full stack trace is reserved for debug level so
		// it does not clutter normal output.
		logger.Error("command failed", slog.Any("error", err))
		logger.Debug("error details", slog.String("stacktrace", fmt.Sprintf("%+v", err)))
		return err
	}

	return nil
}

func cmdDump() *cli.Command {
	var (
		opt     inspectOptions
		compact bool
	)

	return &cli.Command{
		Name:    "dump",
		Usage:   "output go codes as AST",
		Aliases: []string{"d"},
		Flags: append(opt.Flags(), []cli.Flag{
			&cli.BoolFlag{
				Name:        "compact",
				Aliases:     []string{"c"},
				Usage:       "Compact instead of pretty-printed output",
				Destination: &compact,
			},
		}...),
		Action: func(ctx context.Context, c *cli.Command) error {
			codes := c.Args().Slice()

			g := goast.New(
				goast.WithInspectOptions(opt.Options()...),
				goast.WithCompact(compact),
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

func cmdSync() *cli.Command {
	var compact bool
	return &cli.Command{
		Name:    "sync",
		Usage:   "sync dump of go codes to files",
		Aliases: []string{"s"},
		Flags: []cli.Flag{&cli.BoolFlag{
			Name:        "compact",
			Aliases:     []string{"c"},
			Usage:       "Compact instead of pretty-printed output",
			Destination: &compact,
		},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			g := goast.New(
				goast.WithCompact(compact),
			)

			for _, src := range c.Args().Slice() {
				if err := g.Sync(src); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

// evalFlagInput is the CLI side of eval configuration: each value plus whether
// the operator set it explicitly. Splitting it out keeps resolveEval a pure
// function that is easy to unit-test without spinning up a cli.Command.
type evalFlagInput struct {
	policies         []string
	policiesSet      bool
	args             []string
	format           string
	formatSet        bool
	output           string
	outputSet        bool
	fail             bool
	failSet          bool
	ignoreAutoGen    bool
	ignoreAutoGenSet bool
	workers          int
	workersSet       bool
	objectDepth      int
	objectDepthSet   bool
}

// resolvedEval is the effective configuration after merging, in precedence
// order, CLI flags over the [eval] section over built-in defaults. An empty
// output means stdout.
type resolvedEval struct {
	policies      []string
	paths         []string
	format        string
	output        string
	fail          bool
	ignoreAutoGen bool
	workers       int
	objectDepth   int
}

// resolveEval applies the precedence "CLI (explicitly set) > [eval] config >
// built-in default". An explicitly-set CLI flag always wins; otherwise the
// config value fills in. For fields whose built-in default equals the zero value
// (bools, ints, empty slices/strings) an unset config field naturally yields the
// default; only format has a non-zero default, handled explicitly.
func resolveEval(in evalFlagInput, sec evalSection) resolvedEval {
	var r resolvedEval

	if in.policiesSet {
		r.policies = in.policies
	} else {
		r.policies = sec.Policy
	}

	// Positional args are the primary way to name targets; the config default is
	// used only when none are given.
	if len(in.args) > 0 {
		r.paths = in.args
	} else {
		r.paths = sec.Paths
	}

	switch {
	case in.formatSet:
		r.format = in.format
	case sec.Format != "":
		r.format = sec.Format
	default:
		r.format = "text"
	}

	// output "" means stdout. On the CLI, '-' is the stdout sentinel; the config
	// uses a plain path (and an empty/omitted value for stdout), so there is no
	// sentinel to translate on the config side.
	if in.outputSet {
		if in.output != "-" {
			r.output = in.output
		}
	} else {
		r.output = sec.Output
	}

	if in.failSet {
		r.fail = in.fail
	} else {
		r.fail = sec.Fail
	}

	if in.ignoreAutoGenSet {
		r.ignoreAutoGen = in.ignoreAutoGen
	} else {
		r.ignoreAutoGen = sec.IgnoreAutoGenerated
	}

	if in.workersSet {
		r.workers = in.workers
	} else {
		r.workers = sec.Workers
	}

	if in.objectDepthSet {
		r.objectDepth = in.objectDepth
	} else {
		r.objectDepth = sec.ObjectDepth
	}

	return r
}

func cmdEval() *cli.Command {
	var (
		policies      []string
		format        string
		output        string
		fail          bool
		opt           inspectOptions
		ignoreAutoGen bool
		workers       int
		config        string
	)

	return &cli.Command{
		Name:    "eval",
		Usage:   "inspect and check Go code with Rego policy",
		Aliases: []string{"e"},
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				// Shared with `goast test`: the [eval] section supplies defaults.
				Sources:     cli.EnvVars("GOAST_CONFIG"),
				Destination: &config,
				Value:       ".goast.toml",
				Usage:       "Path to the config TOML file; the [eval] section supplies defaults",
			},
			&cli.StringSliceFlag{
				Name:        "policy",
				Aliases:     []string{"p"},
				Destination: &policies,
				// Not Required: it may instead come from [eval].policy. The
				// requirement is enforced after config resolution.
				Usage: "Policy directory or file (or set [eval].policy)",
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
			&cli.BoolFlag{
				Name:        "fail",
				Destination: &fail,
				Usage:       "Exit with non-zero code when detecting violation",
			},
			&cli.BoolFlag{
				Name:        "ignore-auto-generated",
				Destination: &ignoreAutoGen,
				Usage:       "Ignore auto generated go code file",
			},
			&cli.IntFlag{
				Name:        "workers",
				Aliases:     []string{"w"},
				Sources:     cli.EnvVars("GOAST_WORKERS"),
				Destination: &workers,
				Value:       0,
				Usage:       "Number of files evaluated in parallel (0 = number of CPUs)",
			},
		}, opt.Flags()...),
		Action: func(ctx context.Context, c *cli.Command) error {
			// Load the [eval] config. A missing *default* config is not an error —
			// eval then runs from CLI flags alone ("no config -> CLI-only"). But an
			// explicitly requested --config that is absent IS an error, so the
			// operator's intent is never silently ignored.
			var evalSec evalSection
			cfgPath := c.String("config")
			switch cfg, err := loadConfig(cfgPath); {
			case err == nil:
				if verr := cfg.Eval.validate(); verr != nil {
					return goerr.Wrap(verr, "invalid [eval] config", goerr.V("path", cfgPath))
				}
				evalSec = cfg.Eval
			case errors.Is(err, os.ErrNotExist) && !c.IsSet("config"):
				// default .goast.toml absent -> CLI-only; evalSec stays zero.
			case errors.Is(err, os.ErrNotExist):
				return goerr.Wrap(err, "config file not found", goerr.V("path", cfgPath))
			default:
				return err
			}

			r := resolveEval(evalFlagInput{
				policies:         policies,
				policiesSet:      c.IsSet("policy"),
				args:             c.Args().Slice(),
				format:           format,
				formatSet:        c.IsSet("format"),
				output:           output,
				outputSet:        c.IsSet("output"),
				fail:             fail,
				failSet:          c.IsSet("fail"),
				ignoreAutoGen:    ignoreAutoGen,
				ignoreAutoGenSet: c.IsSet("ignore-auto-generated"),
				workers:          workers,
				workersSet:       c.IsSet("workers"),
				objectDepth:      opt.ObjectDepth,
				objectDepthSet:   c.IsSet("object-depth"),
			}, evalSec)

			if len(r.policies) == 0 {
				return goerr.New("policy is required: pass --policy or set [eval].policy")
			}
			if len(r.paths) == 0 {
				return goerr.New("no evaluation target: pass paths as arguments or set [eval].paths")
			}

			f, ok := toOutputFormat(r.format)
			if !ok {
				return goerr.New("unsupported output format", goerr.V("format", r.format))
			}
			if r.workers < 0 {
				return goerr.New("workers must be >= 0", goerr.V("workers", r.workers))
			}

			var w io.Writer
			if r.output == "" {
				w = os.Stdout
			} else {
				fd, err := os.Create(filepath.Clean(r.output))
				if err != nil {
					return goerr.Wrap(err, "failed to open output file", goerr.V("output", r.output))
				}
				defer func() {
					if err := fd.Close(); err != nil {
						logger.Warn("failed to close output file", slog.Any("error", err))
					}
				}()
				w = fd
			}

			engine, err := goast.NewPolicyEngine(r.policies...)
			if err != nil {
				return goerr.Wrap(err, "failed to load policy", goerr.V("policies", r.policies))
			}

			// The resolved object depth may come from config; feed it into the
			// inspect options the same way the CLI flag would.
			opt.ObjectDepth = r.objectDepth
			goastOptions := []goast.Option{
				goast.WithPolicyEngine(engine),
				goast.WithInspectOptions(opt.Options()...),
			}
			if r.ignoreAutoGen {
				goastOptions = append(goastOptions, goast.WithIgnoreAutoGen())
			}

			g := goast.New(goastOptions...)

			fileList, err := listGoFiles(r.paths)
			if err != nil {
				return err
			}

			failCases, err := evalFilesParallel(fileList, r.workers, g.Eval)
			if err != nil {
				return err
			}

			switch f {
			case outputText:
				for _, fail := range failCases {
					if err := outputFailAsText(w, fail); err != nil {
						return err
					}
				}

				fmt.Fprintf(w, "\n\tDetected %d violations\n\n", len(failCases))

			case outputJSON:
				diagnosis := newRDJSONResult(failCases)

				if err := json.NewEncoder(w).Encode(diagnosis); err != nil {
					return goerr.Wrap(err, "failed to convert DiagnosticResult")
				}
			}

			if r.fail && len(failCases) > 0 {
				return goerr.Wrap(errEvalFail, "violations detected", goerr.V("count", len(failCases)))
			}

			return nil
		},
	}
}
