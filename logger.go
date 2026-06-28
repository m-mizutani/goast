package goast

import (
	"log/slog"
	"sync/atomic"
)

// logger is stored atomically so that SetLogger and Logger are safe for
// concurrent use: goast is a library and callers may replace the logger at any
// time while internals are emitting logs.
var logger atomic.Pointer[slog.Logger]

func init() {
	logger.Store(slog.Default())
}

// SetLogger replaces the package-level logger used by goast internals. The
// caller (typically the CLI) is responsible for constructing the slog.Logger
// with the desired handler, level and output.
func SetLogger(l *slog.Logger) {
	logger.Store(l)
}

func Logger() *slog.Logger {
	return logger.Load()
}
