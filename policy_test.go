package goast_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

func writePolicyFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policy.rego")
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// PolicyEngine must evaluate every policy as Rego v1: v1 keyword syntax (contains
// / if) compiles without an `import rego.v1` line, and the legacy v0 partial-set
// syntax is rejected. This is the guarantee the opac-based path could not give.
func TestPolicyEngineForcesRegoV1(t *testing.T) {
	t.Run("v1 syntax compiles and evaluates without import", func(t *testing.T) {
		path := writePolicyFile(t, `package test

allow contains msg if {
	input.x == 1
	msg := "ok"
}
`)
		engine := gt.R1(goast.NewPolicyEngine(path)).NoError(t)

		var out struct {
			Allow []string `json:"allow"`
		}
		gt.NoError(t, engine.Query(context.Background(), "data.test", map[string]any{"x": 1}, &out))
		gt.A(t, out.Allow).Length(1)
		gt.V(t, out.Allow[0]).Equal("ok")
	})

	t.Run("legacy v0 partial-set syntax is rejected", func(t *testing.T) {
		path := writePolicyFile(t, `package test

allow[msg] {
	input.x == 1
	msg := "ok"
}
`)
		_, err := goast.NewPolicyEngine(path)
		gt.Error(t, err)
	})
}

func TestPolicyEngineQueryWithoutMatch(t *testing.T) {
	path := writePolicyFile(t, `package test

allow contains msg if {
	input.x == 1
	msg := "ok"
}
`)
	engine := gt.R1(goast.NewPolicyEngine(path)).NoError(t)

	var out struct {
		Allow []string `json:"allow"`
	}
	// input.x != 1, so the rule does not fire and output stays empty (no error).
	gt.NoError(t, engine.Query(context.Background(), "data.test", map[string]any{"x": 2}, &out))
	gt.A(t, out.Allow).Length(0)
}

// Query must honor a cancelled context. Since the engine passes an external
// Cancel to OPA (to avoid a per-eval goroutine), it checks the context itself
// at the top of Query; a context cancelled before the call must surface as an
// error rather than silently evaluating.
func TestPolicyEngineQueryRespectsCancelledContext(t *testing.T) {
	path := writePolicyFile(t, `package test

allow contains msg if {
	input.x == 1
	msg := "ok"
}
`)
	engine := gt.R1(goast.NewPolicyEngine(path)).NoError(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out struct {
		Allow []string `json:"allow"`
	}
	err := engine.Query(ctx, "data.test", map[string]any{"x": 1}, &out)
	gt.Error(t, err)
	gt.V(t, errors.Is(err, context.Canceled)).Equal(true)
}

func TestNewPolicyEngineNoPolicy(t *testing.T) {
	_, err := goast.NewPolicyEngine(t.TempDir())
	gt.Error(t, err)
}

func TestNewPolicyEngineDeduplicatesOverlappingPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.rego")
	gt.NoError(t, os.WriteFile(path, []byte("package test\n\nallow contains \"x\" if { input.x == 1 }\n"), 0600))

	abs, err := filepath.Abs(path)
	gt.NoError(t, err)

	// The directory, the relative file and its absolute form all resolve to the same
	// module. Without dedup the compiler would fail with a duplicate-rule error.
	gt.R1(goast.NewPolicyEngine(dir, path, abs)).NoError(t)
}

func TestNewPolicyEngineMissingPath(t *testing.T) {
	// A path that does not exist makes WalkDir fail at the root; the error must be
	// surfaced (wrapped), not silently treated as "no policy".
	_, err := goast.NewPolicyEngine(filepath.Join(t.TempDir(), "does-not-exist"))
	gt.Error(t, err)
}

// The bundled policies under policy/ must stay compilable as Rego v1 so the
// directory-loading path (a whole policy dir, as the CLI passes) keeps working.
func TestNewPolicyEngineLoadsBundledPolicies(t *testing.T) {
	gt.R1(goast.NewPolicyEngine("policy")).NoError(t)
}
