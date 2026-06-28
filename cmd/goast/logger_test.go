package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
)

func TestParseLogLevel(t *testing.T) {
	testCases := map[string]struct {
		input   string
		want    slog.Level
		wantErr bool
	}{
		"debug":   {input: "debug", want: slog.LevelDebug},
		"info":    {input: "info", want: slog.LevelInfo},
		"warn":    {input: "warn", want: slog.LevelWarn},
		"error":   {input: "error", want: slog.LevelError},
		"unknown": {input: "verbose", wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := parseLogLevel(tc.input)
			if tc.wantErr {
				gt.Error(t, err)
				return
			}
			gt.NoError(t, err)
			gt.Equal(t, got, tc.want)
		})
	}
}

func TestNewLoggerText(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, slog.LevelInfo, "text")
	gt.NoError(t, err)
	gt.NotNil(t, l)

	l.Info("hello world", slog.String("file", "main.go"))

	out := buf.String()
	gt.S(t, out).Contains("hello world").Contains("main.go")
}

func TestNewLoggerTextExpandsGoErr(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, slog.LevelInfo, "text")
	gt.NoError(t, err)

	gerr := goerr.New("boom", goerr.V("path", "/tmp/x.go"))
	l.Error("failed", slog.Any("error", gerr))

	// The GoErr attr hook expands goerr values, so the attached "path" value
	// must be visible in the console output.
	gt.S(t, buf.String()).Contains("boom").Contains("/tmp/x.go")
}

func TestNewLoggerJSON(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, slog.LevelInfo, "json")
	gt.NoError(t, err)
	gt.NotNil(t, l)

	l.Info("hello", slog.String("file", "main.go"))

	var record map[string]any
	gt.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	gt.Equal(t, record["msg"], "hello")
	gt.Equal(t, record["file"], "main.go")
	gt.Equal(t, record["level"], "INFO")
}

func TestNewLoggerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l, err := newLogger(&buf, slog.LevelWarn, "json")
	gt.NoError(t, err)

	l.Info("should be filtered out")
	gt.Equal(t, buf.Len(), 0)

	l.Warn("should appear")
	gt.S(t, buf.String()).Contains("should appear")
}

func TestNewLoggerUnsupportedFormat(t *testing.T) {
	_, err := newLogger(&bytes.Buffer{}, slog.LevelInfo, "xml")
	gt.Error(t, err)
}
