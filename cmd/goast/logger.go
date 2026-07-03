package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/fatih/color"
	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/clog/hooks"
	"github.com/m-mizutani/goerr/v2"
	"github.com/mattn/go-isatty"
)

// parseLogLevel converts a human readable log level into slog.Level.
func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, goerr.New("unsupported log level", goerr.V("level", level))
	}
}

// newLogger builds a *slog.Logger for the given writer, level and format.
// "text" uses clog for a colorized, human friendly console output and "json"
// uses the standard slog JSON handler. goerr values are expanded by clog's
// GoErr attr hook so that errors carry their diagnostic context to the console.
func newLogger(w io.Writer, level slog.Level, format string) (*slog.Logger, error) {
	var handler slog.Handler
	switch format {
	case "text":
		// clog decides coloring from $TERM by default, ignoring the actual
		// writer. That would leak ANSI escapes into a file when --log-output
		// points to one while running under a color-capable terminal. Gate
		// coloring on whether the writer itself is a TTY.
		enableColor := false
		if f, ok := w.(*os.File); ok {
			enableColor = isatty.IsTerminal(f.Fd())
		}

		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithColor(enableColor),
			clog.WithAttrHook(hooks.GoErr()),
			clog.WithColorMap(&clog.ColorMap{
				Level: map[slog.Level]*color.Color{
					slog.LevelDebug: color.New(color.FgGreen, color.Bold),
					slog.LevelInfo:  color.New(color.FgCyan, color.Bold),
					slog.LevelWarn:  color.New(color.FgYellow, color.Bold),
					slog.LevelError: color.New(color.FgRed, color.Bold),
				},
				LevelDefault: color.New(color.FgBlue, color.Bold),
				Time:         color.New(color.FgWhite),
				Message:      color.New(color.FgHiWhite),
				AttrKey:      color.New(color.FgHiCyan),
				AttrValue:    color.New(color.FgHiWhite),
			}),
		)

	case "json":
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: level,
		})

	default:
		return nil, goerr.New("unsupported log format", goerr.V("format", format))
	}

	return slog.New(handler), nil
}
