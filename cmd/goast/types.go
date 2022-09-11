package main

import "github.com/m-mizutani/goerr"

type outputFormat int

const (
	outputText outputFormat = iota + 1
	outputJSON
)

var outputFormats = map[string]outputFormat{
	"text": outputText,
	"json": outputJSON,
}

func toOutputFormat(format string) (outputFormat, bool) {
	f, ok := outputFormats[format]
	return f, ok
}

var (
	errEvalFail = goerr.New("violation detected")
)
