package goast

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
)

func (x *Goast) Sync(src string) error {
	dump := func(dst string, data *Node) error {
		dir := filepath.Dir(dst)

		if err := x.mkdir(dir, 0755); err != nil {
			return goerr.Wrap(err, "failed to create dump dir", goerr.V("dir", dir))
		}

		fd, err := x.create(dst)
		if err != nil {
			return goerr.Wrap(err, "failed to create dump file", goerr.V("path", dst))
		}
		defer func() {
			if err := fd.Close(); err != nil {
				Logger().Warn("failed to close dump file", slog.String("path", dst), slog.Any("error", err))
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
			return goerr.Wrap(err, "failed to parse go file", goerr.V("path", fpath))
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
			return goerr.Wrap(err, "failed to walk source directory", goerr.V("path", path))
		}
		if d.IsDir() {
			return nil
		}

		fpath := filepath.Clean(path)
		if filepath.Ext(fpath) != ".go" {
			return nil
		}

		Logger().Debug("loading file", slog.String("file", fpath))

		fd, err := os.Open(fpath)
		if err != nil {
			return goerr.Wrap(err, "failed to open go file", goerr.V("path", fpath))
		}
		defer func() {
			if err := fd.Close(); err != nil {
				Logger().Warn("failed to close file", slog.String("file", fpath), slog.Any("error", err))
			}
		}()

		if err := cb(fpath, fd); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return goerr.Wrap(err, "failed to walk go source")
	}

	return nil
}
