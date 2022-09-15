package goast

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr"
)

func (x *Goast) Sync(src string) error {
	dump := func(dst string, data *Node) error {
		dir := filepath.Dir(dst)

		if err := x.mkdir(dir, 0755); err != nil {
			return goerr.Wrap(err, "failed to create dump dir").With("dir", dir)
		}

		fd, err := x.create(dst)
		if err != nil {
			return goerr.Wrap(err, "failed to create dump file").With("path", dst)
		}
		defer func() {
			if err := fd.Close(); err != nil {
				logger.With("path", dst).Warn(err.Error())
			}
		}()

		encoder := json.NewEncoder(fd)
		if !x.dumpCompact {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(data); err != nil {
			return goerr.Wrap(err, "failed to encode goast.Node")
		}

		return nil
	}

	return x.walk(src, func(fpath string, r io.Reader) error {
		fSet := token.NewFileSet()
		f, err := parser.ParseFile(fSet, fpath, r, parser.ParseComments)
		if err != nil {
			return goerr.Wrap(err)
		}

		comments := map[int]string{}
		sync := func(data *Node) error {
			if file, ok := data.Node.(*ast.File); ok {
				comments = toCommentMap(file, fSet)

				// If goast.sync comment exists at head of file, dump *ast.File
				if dst, ok := comments[1]; ok {
					if err := dump(dst, data); err != nil {
						return err
					}
				}
				delete(comments, 1)

				return nil
			}

			pos := fSet.Position(data.Node.Pos())
			if dst, ok := comments[pos.Line]; ok {
				if err := dump(dst, data); err != nil {
					return err
				}

				delete(comments, pos.Line)
			}
			return nil
		}

		if err := Inspect(f, fSet, sync, x.inspectOpt...); err != nil {
			return err
		}

		return nil
	})
}

func toCommentMap(f *ast.File, fSet *token.FileSet) map[int]string {
	const commentPrefix = "goast.sync:"

	comments := map[int]string{}

	for _, c := range f.Comments {
		pos := fSet.Position(c.Pos())
		for _, l := range c.List {
			idx := strings.Index(l.Text, commentPrefix)
			if idx < 0 {
				continue
			}

			comments[pos.Line] = strings.TrimSpace(l.Text[idx+len(commentPrefix):])
		}
	}

	return comments
}

func walkGoCode(src string, cb func(fpath string, r io.Reader) error) error {
	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
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

		logger.With("file", fpath).Debug("loading file")

		fd, err := os.Open(fpath)
		if err != nil {
			return goerr.Wrap(err)
		}
		defer func() {
			if err := fd.Close(); err != nil {
				logger.Err(err).With("file", fpath).Warn("failed to close file")
			}
		}()

		if err := cb(fpath, fd); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return goerr.Wrap(err)
	}

	return nil
}
