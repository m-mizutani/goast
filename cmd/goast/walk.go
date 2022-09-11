package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr"
)

type callback func(filePath string, r io.Reader) error

func walkCode(codes []string, cb callback) error {
	for _, codePath := range codes {
		if err := filepath.WalkDir(codePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return goerr.Wrap(err)
			}
			if d.IsDir() {
				return nil
			}

			fpath := filepath.Clean(path)
			if filepath.Ext(fpath) != ".go" {
				return nil
			}

			goast.Logger().With("file", fpath).Debug("loading file")

			fd, err := os.Open(fpath)
			if err != nil {
				return goerr.Wrap(err)
			}
			defer func() {
				if err := fd.Close(); err != nil {
					goast.Logger().Err(err).With("file", fpath).Warn("failed to close file")
				}
			}()

			if err := cb(fpath, fd); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return goerr.Wrap(err)
		}
	}

	return nil
}
