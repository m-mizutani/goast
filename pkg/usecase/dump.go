package usecase

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goast/pkg/model"
	"github.com/m-mizutani/goast/pkg/utils"
	"github.com/m-mizutani/goerr"
)

func DumpDir(codes []string, outDir string) error {
	callback := func(codePath string, data *model.File) error {
		codeDir := filepath.Dir(codePath)
		dir := filepath.Join(outDir, codeDir)

		// #nosec
		if err := os.MkdirAll(dir, 0755); err != nil {
			return goerr.Wrap(err)
		}

		outPath := filepath.Join(dir, filepath.Base(codePath)+".json")
		fd, err := os.Create(filepath.Clean(outPath))
		if err != nil {
			return err
		}
		defer func() {
			if err := fd.Close(); err != nil {
				utils.Logger().Warn("failed to close dump file: %s", err)
			}
		}()

		enc := json.NewEncoder(fd)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			return goerr.Wrap(err)
		}

		return nil
	}

	return walkCode(codes, callback)
}

func DumpWriter(codes []string, w io.Writer) error {
	callback := func(path string, data *model.File) error {
		raw, err := json.Marshal(data)
		if err != nil {
			return goerr.Wrap(err)
		}

		fmt.Fprintln(w, string(raw))
		return nil
	}

	return walkCode(codes, callback)
}
