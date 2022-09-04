package usecase_test

import (
	"bytes"
	"testing"

	"github.com/m-mizutani/goast/pkg/usecase"
	"github.com/stretchr/testify/require"
)

func TestDumpLine(t *testing.T) {
	codePath := createTestCode(t, `package main

	func Add(a, b int) int {
		return a + b
	}

	func Sub(a, b int) int {
		return a - b
	}
	`)

	var w bytes.Buffer
	require.NoError(t, usecase.DumpWriter([]string{codePath}, &w, usecase.WithDumpLine(7)))
}

func TestStructAccess(t *testing.T) {
	codePath := createTestCode(t, `package main

	import "model"

	func main() {
		var u0 *model.User
		u1 := &model.User{}
		u2 := model.User{}
	}
	`)

	var w bytes.Buffer
	require.NoError(t, usecase.DumpWriter([]string{codePath}, &w, usecase.WithDumpLine(7)))
}
