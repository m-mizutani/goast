package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

func TestNewRDJSONResult(t *testing.T) {
	failCases := []*goast.Fail{
		{Path: "foo.go", Line: 10, Column: 5, Msg: "first violation"},
		{Path: "bar/baz.go", Line: 42, Column: 1, Msg: "second violation"},
	}

	result := newRDJSONResult(failCases)

	gt.NotNil(t, result.Source)
	gt.Equal(t, result.Source.Name, "goast")
	gt.Equal(t, result.Source.Url, "https://github.com/m-mizutani/goast")

	gt.A(t, result.Diagnostics).Length(2).
		At(0, func(t testing.TB, d *rdjsonDiagnostic) {
			gt.Equal(t, d.Message, "first violation")
			gt.Equal(t, d.Location.Path, "foo.go")
			gt.Equal(t, d.Location.Range.Start.Line, 10)
			gt.Equal(t, d.Location.Range.Start.Column, 5)
		}).
		At(1, func(t testing.TB, d *rdjsonDiagnostic) {
			gt.Equal(t, d.Message, "second violation")
			gt.Equal(t, d.Location.Path, "bar/baz.go")
			gt.Equal(t, d.Location.Range.Start.Line, 42)
			gt.Equal(t, d.Location.Range.Start.Column, 1)
		})
}

func TestNewRDJSONResultEmpty(t *testing.T) {
	result := newRDJSONResult(nil)

	// Even with no violations the source must be present so downstream tools can
	// attribute the (empty) report to goast.
	gt.NotNil(t, result.Source)
	gt.Nil(t, result.Diagnostics)
}

// TestRDJSONResultEncoding pins the on-the-wire JSON shape. The output is a
// public contract consumed by `reviewdog -f rdjson`, so the field names and
// nesting must not drift.
func TestRDJSONResultEncoding(t *testing.T) {
	result := newRDJSONResult([]*goast.Fail{
		{Path: "foo.go", Line: 10, Column: 5, Msg: "boom"},
	})

	var buf bytes.Buffer
	gt.NoError(t, json.NewEncoder(&buf).Encode(result))

	// Decode into a generic map to assert exact key layout independent of the
	// Go struct definitions.
	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	want := map[string]any{
		"source": map[string]any{
			"name": "goast",
			"url":  "https://github.com/m-mizutani/goast",
		},
		"diagnostics": []any{
			map[string]any{
				"message": "boom",
				"location": map[string]any{
					"path": "foo.go",
					"range": map[string]any{
						"start": map[string]any{
							"line":   float64(10),
							"column": float64(5),
						},
					},
				},
			},
		},
	}

	gt.Equal(t, got, want)
}

// TestRDJSONResultEmptyEncoding ensures omitempty drops the diagnostics array
// (and zero-valued fields) when there is nothing to report.
func TestRDJSONResultEmptyEncoding(t *testing.T) {
	var buf bytes.Buffer
	gt.NoError(t, json.NewEncoder(&buf).Encode(newRDJSONResult(nil)))

	var got map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	gt.Equal(t, got, map[string]any{
		"source": map[string]any{
			"name": "goast",
			"url":  "https://github.com/m-mizutani/goast",
		},
	})
}
