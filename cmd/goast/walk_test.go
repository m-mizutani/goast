package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
)

// listGoFiles must return only .go files, in a stable walk order, so that
// downstream parallel evaluation can still emit findings deterministically.
func TestListGoFiles(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0750))
	write := func(rel string) {
		gt.NoError(t, os.WriteFile(filepath.Join(dir, rel), []byte("package x\n"), 0600))
	}
	write("a.go")
	write("b.go")
	write(filepath.Join("sub", "c.go"))
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0600))

	files := gt.R1(listGoFiles([]string{dir})).NoError(t)

	want := []string{
		filepath.Join(dir, "a.go"),
		filepath.Join(dir, "b.go"),
		filepath.Join(dir, "sub", "c.go"),
	}
	gt.V(t, files).Equal(want)

	// A second call yields the same order (WalkDir is lexical, hence stable).
	again := gt.R1(listGoFiles([]string{dir})).NoError(t)
	gt.V(t, again).Equal(want)
}

// evalFilesParallel must emit findings in input (walk) order no matter how many
// workers run, and no matter the order in which they happen to finish. The eval
// closure sleeps for a jittered, index-dependent duration so that completion
// order is deliberately scrambled relative to input order.
func TestEvalFilesParallelDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for i := range 50 {
		p := filepath.Join(dir, "f"+strconv.Itoa(i)+".go")
		gt.NoError(t, os.WriteFile(p, []byte("package x\n"), 0600))
		files = append(files, p)
	}

	// Each file yields one Fail tagged with its own path, so the concatenated
	// result directly reflects ordering. Later-indexed files finish sooner,
	// scrambling completion order away from input order.
	eval := func(path string, _ io.Reader) ([]*goast.Fail, error) {
		idx, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "f"), ".go"))
		time.Sleep(time.Duration(50-idx) * 100 * time.Microsecond)
		return []*goast.Fail{{Path: path, Msg: "x"}}, nil
	}

	for _, workers := range []int{1, 4, 16, 64} {
		got := gt.R1(evalFilesParallel(files, workers, eval)).NoError(t)
		paths := make([]string, len(got))
		for i, f := range got {
			paths[i] = f.Path
		}
		gt.V(t, paths).Equal(files)
	}
}

// A real evaluation error must be surfaced (fail-fast), wrapped with the file
// path for diagnosis.
func TestEvalFilesParallelFailFast(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		p := filepath.Join(dir, name)
		gt.NoError(t, os.WriteFile(p, []byte("package x\n"), 0600))
		files = append(files, p)
	}

	sentinel := goerr.New("boom")
	eval := func(path string, _ io.Reader) ([]*goast.Fail, error) {
		if filepath.Base(path) == "b.go" {
			return nil, sentinel
		}
		return nil, nil
	}

	_, err := evalFilesParallel(files, 4, eval)
	gt.Error(t, err)
	// The original error must survive wrapping so callers can inspect it, and
	// the offending path must be attached for diagnosis.
	gt.V(t, errors.Is(err, sentinel)).Equal(true)
	gt.V(t, goerr.Values(err)["path"]).Equal(filepath.Join(dir, "b.go"))
}

// The parallel CLI path must detect exactly the same violations regardless of
// worker count, over the shared corpus that the package-level golden test pins.
func TestEvalFilesParallelMatchesAcrossWorkerCounts(t *testing.T) {
	engine := gt.R1(goast.NewPolicyEngine("../../testdata/eval/policy")).NoError(t)
	g := goast.New(goast.WithPolicyEngine(engine))

	files := gt.R1(listGoFiles([]string{"../../testdata/eval/src"})).NoError(t)
	gt.A(t, files).Longer(0)

	serial := gt.R1(evalFilesParallel(files, 1, g.Eval)).NoError(t)
	gt.A(t, serial).Longer(0)

	for _, workers := range []int{2, 8, 32} {
		got := gt.R1(evalFilesParallel(files, workers, g.Eval)).NoError(t)
		gt.V(t, got).Equal(serial)
	}
}
