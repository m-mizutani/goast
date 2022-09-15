package goast_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, "test/sync/dir", path)
			assert.Equal(t, fs.FileMode(0755), perm)
			return nil
		}),
		goast.WithCreateFunc(func(path string) (io.WriteCloser, error) {
			files := []string{
				"test/sync/dir/dest1.json",
				"test/sync/dir/dest2.json",
			}
			assert.Equal(t, files[createCount], path)
			createCount++
			return buf, nil
		}),
	)
	require.NoError(t, g.Sync("src_dir"))

	assert.Equal(t, 1, walkCount)
	assert.Equal(t, 2, mkdirCount)
	assert.Equal(t, 2, createCount)

	var out node
	r := bytes.NewReader(buf.Bytes())
	require.NoError(t, json.NewDecoder(r).Decode(&out))
	assert.Equal(t, "ExprStmt", out.Kind)
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
			assert.Equal(t, "test", path)
			assert.Equal(t, fs.FileMode(0755), perm)
			return nil
		}),
		goast.WithCreateFunc(func(path string) (io.WriteCloser, error) {
			createCount++
			assert.Equal(t, "test/file.json", path)
			return buf, nil
		}),
	)
	require.NoError(t, g.Sync("src_dir"))

	assert.Equal(t, 1, walkCount)
	assert.Equal(t, 1, mkdirCount)
	assert.Equal(t, 1, createCount)

	var out node
	r := bytes.NewReader(buf.Bytes())
	require.NoError(t, json.NewDecoder(r).Decode(&out))
	assert.Equal(t, "File", out.Kind)
}
