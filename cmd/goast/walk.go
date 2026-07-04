package main

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr/v2"
	"golang.org/x/sync/errgroup"
)

type callback func(filePath string, r io.Reader) error

// listGoFiles walks the given paths and returns every .go file in deterministic
// walk order. Separating enumeration from evaluation lets the eval command fan
// the files out across workers while still emitting findings in a stable order.
func listGoFiles(codes []string) ([]string, error) {
	var files []string
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
			files = append(files, fpath)
			return nil
		}); err != nil {
			return nil, goerr.Wrap(err, "failed to walk code path", goerr.V("path", codePath))
		}
	}
	return files, nil
}

// evalFilesParallel evaluates files concurrently with a bounded worker pool and
// returns the findings in the input (walk) order, so output is deterministic
// regardless of worker scheduling. workers <= 0 means one worker per CPU.
//
// Fail-fast: the first file error is what Wait returns, and once it occurs the
// shared context is cancelled so files not yet started are skipped rather than
// evaluated for nothing. Files already in flight when the error happens run to
// completion — they are independent, so there is no partial work to unwind and
// their results are discarded by the caller on error anyway.
func evalFilesParallel(files []string, workers int, eval func(path string, r io.Reader) ([]*goast.Fail, error)) ([]*goast.Fail, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	results := make([][]*goast.Fail, len(files))

	grp, ctx := errgroup.WithContext(context.Background())
	grp.SetLimit(workers)

	for i, fpath := range files {
		// A prior file has failed and cancelled ctx: stop scheduling more work.
		if ctx.Err() != nil {
			break
		}
		grp.Go(func() error {
			// This file was scheduled just before the cancellation landed; skip
			// it so the run winds down instead of evaluating it for nothing.
			if ctx.Err() != nil {
				return nil
			}
			fd, err := os.Open(fpath) // #nosec G304 -- fpath comes from the operator-supplied code paths; reading them is the tool's purpose
			if err != nil {
				return goerr.Wrap(err, "failed to open go file", goerr.V("path", fpath))
			}
			defer func() {
				if err := fd.Close(); err != nil {
					goast.Logger().Warn("failed to close file", slog.String("file", fpath), slog.Any("error", err))
				}
			}()

			goast.Logger().Debug("loading file", slog.String("file", fpath))

			resp, err := eval(fpath, fd)
			if err != nil {
				return goerr.Wrap(err, "failed to evaluate file", goerr.V("path", fpath))
			}
			results[i] = resp
			return nil
		})
	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	var all []*goast.Fail
	for _, r := range results {
		all = append(all, r...)
	}
	return all, nil
}

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
