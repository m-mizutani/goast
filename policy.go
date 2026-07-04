package goast

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
)

// PolicyEngine compiles and evaluates Rego policies. Unlike a default OPA/opac
// setup, which parses modules as Rego v0 (ast.DefaultRegoVersion is a compile-time
// const = RegoV0 and opac exposes no knob to change it), this engine forces
// ast.RegoV1 so that every policy — including ones that omit `import rego.v1` — is
// always evaluated under Rego v1 syntax and semantics.
type PolicyEngine struct {
	compiler *ast.Compiler

	// prepared caches one PreparedEvalQuery per query string. Eval calls Query once
	// per AST node (potentially thousands of times over a single file), so preparing
	// the query just once and only swapping the input keeps the per-node cost down.
	mu       sync.Mutex
	prepared map[string]rego.PreparedEvalQuery

	// cancel is a single, never-fired Cancel shared across all Eval calls. Passing
	// it via rego.EvalExternalCancel stops OPA from spawning a per-eval goroutine
	// (plus channel) to watch ctx for cancellation — a cost that dominates once the
	// actual rule evaluation is cheap (a few microseconds per node). It is only an
	// atomic flag, so sharing one instance across concurrent Eval calls is safe; we
	// never call Cancel() on it. Context cancellation is instead honored per node by
	// checking ctx at the top of Query.
	cancel topdown.Cancel
}

// NewPolicyEngine reads every .rego file under the given file or directory paths
// and compiles them as Rego v1. A directory is walked recursively for *.rego files.
func NewPolicyEngine(paths ...string) (*PolicyEngine, error) {
	// Collect the .rego paths during the walk and read them afterwards, keeping the
	// file reads out of the WalkDir callback. Deduplicate by absolute path so that
	// overlapping arguments (e.g. a directory and a file inside it, or the same path
	// given in both relative and absolute form) do not load a module twice and make
	// the compiler fail with a duplicate-rule error. The readable, cleaned path is
	// kept as the module identifier so compile errors point at the given path.
	var regoFiles []string
	seen := map[string]bool{}
	for _, path := range paths {
		err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(p) != ".rego" {
				return nil
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				return goerr.Wrap(err, "failed to resolve policy path", goerr.V("path", p))
			}
			if seen[abs] {
				return nil
			}
			seen[abs] = true
			regoFiles = append(regoFiles, filepath.Clean(p))
			return nil
		})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to load policy path", goerr.V("path", path))
		}
	}

	policies := map[string]string{}
	for _, fpath := range regoFiles {
		raw, err := os.ReadFile(fpath) // #nosec G304 -- policy paths are operator-supplied, reading them is the tool's purpose
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read policy file", goerr.V("path", fpath))
		}
		policies[fpath] = string(raw)
	}

	if len(policies) == 0 {
		return nil, goerr.New("no policy file is found", goerr.V("paths", paths))
	}

	compiler, err := ast.CompileModulesWithOpt(policies, ast.CompileOpts{
		ParserOptions: ast.ParserOptions{
			ProcessAnnotation: true,
			RegoVersion:       ast.RegoV1,
		},
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to compile policy", goerr.V("paths", paths))
	}

	return &PolicyEngine{
		compiler: compiler,
		prepared: map[string]rego.PreparedEvalQuery{},
		cancel:   topdown.NewCancel(),
	}, nil
}

// Query evaluates the given Rego query against input and decodes the resulting
// value into output via JSON. It mirrors how opac decodes Rego output so that the
// only behavioral change from the previous opac-based path is the forced Rego v1.
func (e *PolicyEngine) Query(ctx context.Context, query string, input, output any) error {
	// With rego.EvalExternalCancel below, OPA no longer watches ctx for
	// cancellation mid-evaluation. Query runs once per AST node, so checking ctx
	// here gives cancellation at node granularity, which is the finest the caller
	// (Eval, looping over nodes) can act on anyway.
	if err := ctx.Err(); err != nil {
		return goerr.Wrap(err, "context done before policy evaluation", goerr.V("query", query))
	}

	pq, err := e.preparedQuery(ctx, query)
	if err != nil {
		return err
	}

	rs, err := pq.Eval(ctx, rego.EvalInput(input), rego.EvalExternalCancel(e.cancel))
	if err != nil {
		return goerr.Wrap(err, "failed to evaluate query", goerr.V("query", query))
	}

	// No matching rule (e.g. a file with no violations) yields no result. Treat it
	// as an empty output rather than an error so callers see zero findings.
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return nil
	}

	raw, err := json.Marshal(rs[0].Expressions[0].Value)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal query result")
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return goerr.Wrap(err, "failed to unmarshal query result")
	}

	return nil
}

// preparedQuery returns a PreparedEvalQuery for the query, building and caching it
// on first use. The returned query is safe for concurrent Eval calls.
func (e *PolicyEngine) preparedQuery(ctx context.Context, query string) (rego.PreparedEvalQuery, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if pq, ok := e.prepared[query]; ok {
		return pq, nil
	}

	pq, err := rego.New(
		rego.Query(query),
		rego.Compiler(e.compiler),
	).PrepareForEval(ctx)
	if err != nil {
		return rego.PreparedEvalQuery{}, goerr.Wrap(err, "failed to prepare query", goerr.V("query", query))
	}

	e.prepared[query] = pq
	return pq, nil
}
