package usecase

import (
	"io/fs"
	"path/filepath"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/source"
	"github.com/m-mizutani/goast/pkg/utils"
	"github.com/m-mizutani/goerr"
)

func walkCode(codes []string, callback func(path string, data *model.File) error) error {
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

			utils.Logger().With("file", fpath).Debug("loading file")

			f, err := source.Import(fpath)
			if err != nil {
				return goerr.Wrap(err)
			}

			if err := callback(fpath, f); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return goerr.Wrap(err)
		}
	}

	return nil
}
