package main

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr/v2"
)

type callback func(filePath string, r io.Reader) error

func walkCode(codes []string, cb callback) error {
	for _, codePath := range codes {
		if err := filepath.WalkDir(codePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return goerr.Wrap(err, "failed to walk code path", goerr.V("path", path))
			}
			if d.IsDir() {
				return nil
			}

			fpath := filepath.Clean(path)
			if filepath.Ext(fpath) != ".go" {
				return nil
			}

			goast.Logger().Debug("loading file", slog.String("file", fpath))

			fd, err := os.Open(fpath)
			if err != nil {
				return goerr.Wrap(err, "failed to open go file", goerr.V("path", fpath))
			}
			defer func() {
				if err := fd.Close(); err != nil {
					goast.Logger().Warn("failed to close file", slog.String("file", fpath), slog.Any("error", err))
				}
			}()

			if err := cb(fpath, fd); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return goerr.Wrap(err, "failed to walk code path", goerr.V("path", codePath))
		}
	}

	return nil
}
