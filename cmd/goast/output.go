package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/fatih/color"
	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr"
)

func outputFailAsText(w io.Writer, fail *goast.Fail) error {
	code, err := getLine(fail.Path, fail.Line)
	if err != nil {
		return err
	}

	type fprintf func(w io.Writer, format string, a ...any) (n int, err error)

	underLine := make([]rune, len(code))
	for i, c := range code {
		if i+1 < fail.Column {
			switch {
			case unicode.IsSpace(c):
				underLine[i] = c
			case i+1 < fail.Column:
				underLine[i] = ' '
			}
		} else {
			underLine[i] = '~'
		}
	}

	var cFprintf fprintf = fmt.Fprintf
	if w == os.Stdout {
		cFprintf = color.New(color.FgRed).Fprintf
	}

	fmt.Fprintf(w, "[%s:%d] - ", fail.Path, fail.Line)
	cFprintf(w, "%s\n", fail.Msg)
	fmt.Fprintf(w, "\n%s\n%s\n\n", code, string(underLine))

	return nil
}

func getLine(path string, line int) (string, error) {
	fd, err := os.Open(path)
	if err != nil {
		return "", goerr.Wrap(err)
	}
	defer func() {
		if err := fd.Close(); err != nil {
			logger.Warn(err.Error())
		}
	}()

	var idx int
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		idx++
		if idx == line {
			return scanner.Text(), nil
		}
	}

	return "", nil
}
