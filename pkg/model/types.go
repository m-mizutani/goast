package model

type OutputFormat int

const (
	OutputText OutputFormat = iota + 1
	OutputJSON
)

var outputFormats = map[string]OutputFormat{
	"text": OutputText,
	"json": OutputJSON,
}

func ToOutputFormat(format string) (OutputFormat, bool) {
	f, ok := outputFormats[format]
	return f, ok
}
