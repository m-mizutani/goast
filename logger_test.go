package goast_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

func TestLogger(t *testing.T) {
	// Preserve and restore the package-level logger so this test does not leak
	// state into other tests that rely on goast.Logger().
	original := goast.Logger()
	t.Cleanup(func() { goast.SetLogger(original) })

	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	goast.SetLogger(l)
	gt.Equal(t, goast.Logger(), l)

	goast.Logger().Info("hello", slog.String("key", "value"))
	gt.S(t, buf.String()).Contains("hello").Contains("key=value")
}
