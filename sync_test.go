package goast_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

type buffer struct {
	bytes.Buffer
}

type node struct {
	Path string
	Node map[string]any
	Kind string
}

func (x *buffer) Close() error { return nil }

func TestSync(t *testing.T) {
	code := strings.NewReader(
		`package main

		import "fmt"

		func main() {
			fmt.Println("test1") // goast.sync:test/sync/dir/dest1.json
			fmt.Println("test2") // goast.sync: test/sync/dir/dest2.json
		}
		`,
	)

	buf := &buffer{}
	var walkCount, mkdirCount, createCount int
	g := goast.New(
		goast.WithWalkFunc(func(src string, cb func(fpath string, r io.Reader) error) error {
			walkCount++
			return cb("testcode.go", code)
		}),
		goast.WithMkDirFunc(func(path string, perm fs.FileMode) error {
			mkdirCount++
			gt.V(t, path).Equal("test/sync/dir")
			gt.V(t, perm).Equal(fs.FileMode(0755))
			return nil
		}),
		goast.WithCreateFunc(func(path string) (io.WriteCloser, error) {
			files := []string{
				"test/sync/dir/dest1.json",
				"test/sync/dir/dest2.json",
			}
			gt.V(t, path).Equal(files[createCount])
			createCount++
			return buf, nil
		}),
	)
	gt.NoError(t, g.Sync("src_dir")).Must()

	gt.V(t, walkCount).Equal(1)
	gt.V(t, mkdirCount).Equal(2)
	gt.V(t, createCount).Equal(2)

	var out node
	r := bytes.NewReader(buf.Bytes())
	gt.NoError(t, json.NewDecoder(r).Decode(&out)).Must()
	gt.V(t, out.Kind).Equal("ExprStmt")

}

func TestSyncFile(t *testing.T) {
	code := strings.NewReader(
		`// goast.sync: test/file.json
		package main

		import "fmt"

		func main() {
			fmt.Println("test")
		}
		`,
	)

	buf := &buffer{}
	var walkCount, mkdirCount, createCount int
	g := goast.New(
		goast.WithWalkFunc(func(src string, cb func(fpath string, r io.Reader) error) error {
			walkCount++
			return cb("testcode.go", code)
		}),
		goast.WithMkDirFunc(func(path string, perm fs.FileMode) error {
			mkdirCount++
			gt.V(t, path).Equal("test")
			gt.V(t, perm).Equal(fs.FileMode(0755))
			return nil
		}),
		goast.WithCreateFunc(func(path string) (io.WriteCloser, error) {
			createCount++
			gt.V(t, path).Equal("test/file.json")
			return buf, nil
		}),
	)
	gt.NoError(t, g.Sync("src_dir")).Must()

	gt.V(t, walkCount).Equal(1)
	gt.V(t, mkdirCount).Equal(1)
	gt.V(t, createCount).Equal(1)

	var out node
	r := bytes.NewReader(buf.Bytes())
	gt.NoError(t, json.NewDecoder(r).Decode(&out)).Must()
	gt.V(t, out.Kind).Equal("File")
}
