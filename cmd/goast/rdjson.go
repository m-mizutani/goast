package main

import "github.com/m-mizutani/goast"

// rdjson mirrors the subset of Reviewdog Diagnostic Format (rdjson) that goast
// emits. It is intentionally hand-rolled rather than pulled from
// github.com/reviewdog/reviewdog/proto/rdf: importing that package dragged the
// entire reviewdog module graph (AWS/GCP/GitHub/GitLab SDKs, gRPC, protobuf,
// OpenTelemetry, ...) into go.mod just to borrow a handful of JSON struct tags.
// The field names and order below match the generated rdf structs so the
// encoding/json output stays byte-compatible with `reviewdog -f rdjson`.
type rdjsonResult struct {
	Diagnostics []*rdjsonDiagnostic `json:"diagnostics,omitempty"`
	Source      *rdjsonSource       `json:"source,omitempty"`
}

type rdjsonSource struct {
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

type rdjsonDiagnostic struct {
	Message  string          `json:"message,omitempty"`
	Location *rdjsonLocation `json:"location,omitempty"`
}

type rdjsonLocation struct {
	Path  string       `json:"path,omitempty"`
	Range *rdjsonRange `json:"range,omitempty"`
}

type rdjsonRange struct {
	Start *rdjsonPosition `json:"start,omitempty"`
}

type rdjsonPosition struct {
	Line   int32 `json:"line,omitempty"`
	Column int32 `json:"column,omitempty"`
}

// newRDJSONResult converts goast violations into an rdjson document tagged with
// goast as the diagnostic source.
func newRDJSONResult(failCases []*goast.Fail) *rdjsonResult {
	result := &rdjsonResult{
		Source: &rdjsonSource{
			Name: "goast",
			Url:  "https://github.com/m-mizutani/goast",
		},
	}

	for _, fail := range failCases {
		result.Diagnostics = append(result.Diagnostics, &rdjsonDiagnostic{
			Message: fail.Msg,
			Location: &rdjsonLocation{
				Path: fail.Path,
				Range: &rdjsonRange{
					Start: &rdjsonPosition{
						Line:   int32(fail.Line),
						Column: int32(fail.Column),
					},
				},
			},
		})
	}

	return result
}
