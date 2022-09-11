package goast

import "github.com/m-mizutani/zlog"

var logger = zlog.New()

func RenewLogger(options []zlog.Option) {
	logger = logger.Clone(options...)
}

func Logger() *zlog.Logger {
	return logger
}
