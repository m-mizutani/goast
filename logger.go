package goast

import "log/slog"

var logger = slog.Default()

// SetLogger replaces the package-level logger used by goast internals. The
// caller (typically the CLI) is responsible for constructing the slog.Logger
// with the desired handler, level and output.
func SetLogger(l *slog.Logger) {
	logger = l
}

func Logger() *slog.Logger {
	return logger
}
