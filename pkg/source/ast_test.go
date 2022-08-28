package source_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/m-mizutani/goast/pkg/source"
)

func TestImport(t *testing.T) {
	f, err := source.Import("./../../examples/main.go")
	require.NoError(t, err)
	require.NotNil(t, f)
	p := f.Pos(15)
	assert.Equal(t, 3, p.Line)

	var out bytes.Buffer
	require.NoError(t, json.NewEncoder(&out).Encode(p))
	assert.Less(t, len(out.Bytes()), 100)
}
