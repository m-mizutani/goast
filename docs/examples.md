# Example policies: enforcing your own conventions

Generic linters already catch generic mistakes. goast earns its place on the
rules that are specific to *your* team — the conventions written in a style guide
or a `CLAUDE.md` that no off-the-shelf tool knows about. This page collects
complete, runnable policies for exactly those.

Each example follows the same shape: **the convention (and why)** → **a bad and a
good sample** → **the policy** → **how to pin it with `goast test`**. Every
policy here was verified to fire on its bad sample and stay silent on its good
one. The Rego techniques behind them (which AST node to anchor on, how the shapes
look) are documented in the [`goast-policy` skill](../skills/goast-policy/SKILL.md)
and its [recipes](../skills/goast-policy/references/patterns.md).

All rules live in `package goast` and contribute to the `fail` set. You can keep
them in one `.rego` file or split them across many — the sets merge.

- [1. Wrap errors with your error library, not `fmt.Errorf`](#1-wrap-errors)
- [2. Ban the global `slog` logger; use the context logger](#2-ban-global-slog)
- [3. Ban raw goroutines; use the dispatch helper](#3-ban-raw-goroutines)
- [4. Exported functions must take `context.Context` first](#4-context-first)
- [5. Forbid importing a package directly](#5-forbid-import)
- [6. Ban `time.Now()`; inject a clock](#6-ban-timenow)
- [Pinning examples with `goast test`](#pinning-with-goast-test)

## 1. Wrap errors

**Convention.** Errors must be wrapped with the project's error library
(`goerr.Wrap`) so they carry stack and context; the bare `fmt.Errorf` is
forbidden.

```go
// bad
return fmt.Errorf("failed to process %s", id)

// good
return goerr.Wrap(err, "failed to process", goerr.V("id", id))
```

`pkg.Func(...)` is a `CallExpr` whose `Fun` is a `SelectorExpr`:

```rego
package goast

fail contains res if {
	input.Kind == "CallExpr"
	input.Node.Fun.X.Name == "fmt"
	input.Node.Fun.Sel.Name == "Errorf"

	res := {
		"msg": "wrap errors with goerr.Wrap, do not use fmt.Errorf",
		"pos": input.Node.Fun.X.NamePos,
		"sev": "ERROR",
	}
}
```

Anchoring on `CallExpr` (not `ExprStmt`) also catches `fmt.Errorf` used inside a
larger expression, not just as a bare statement.

## 2. Ban global slog

**Convention.** Logs must go through a request/context-scoped logger, never the
package-level `slog.Info` / `slog.Error` family — those lose the fields the
context logger attaches.

```go
// bad
slog.Info("processing", "id", id)

// good
logger := loggerFrom(ctx)
logger.Info("processing", "id", id)
```

Collect the banned method names in a set so one rule covers all of them:

```rego
package goast

slog_global := {"Debug", "Info", "Warn", "Error"}

fail contains res if {
	input.Kind == "CallExpr"
	input.Node.Fun.X.Name == "slog"
	some m in slog_global
	input.Node.Fun.Sel.Name == m

	res := {
		"msg": sprintf("do not call slog.%s directly; use the context logger", [input.Node.Fun.Sel.Name]),
		"pos": input.Node.Fun.X.NamePos,
		"sev": "ERROR",
	}
}
```

The good sample's `logger.Info(...)` has `Fun.X.Name == "logger"`, so it does not
match. (Name matching does not resolve packages: an aliased `import s "log/slog"`
would slip through — add an import-alias rule if that matters to you.)

## 3. Ban raw goroutines

**Convention.** Background goroutines must launch through the project's
async-dispatch helper (panic recovery, logger propagation, error reporting), not
a bare `go func(){…}()`.

```go
// bad
go func() { process(id) }()

// good
dispatch(ctx, func(ctx context.Context) { process(ctx, id) })
```

This is the simplest kind of rule — "this statement kind must not appear". A
`go` statement is a `GoStmt`:

```rego
package goast

fail contains res if {
	input.Kind == "GoStmt"

	res := {
		"msg": "do not launch raw goroutines; use the async-dispatch helper",
		"pos": input.Node.Go,
		"sev": "ERROR",
	}
}
```

`input.Node.Go` is the position of the `go` keyword. The same one-line shape bans
`DeferStmt`, `SelectStmt`, and other statement kinds.

## 4. Context first

**Convention.** Every exported function takes `context.Context` as its first
argument, so cancellation and request scope thread through the whole call graph.

```go
// bad
func Handle(id string) error { ... }

// good
func Handle(ctx context.Context, id string) error { ... }
```

Anchor on `FuncDecl` and use a helper rule under `not`, so that both "no
parameters" and "wrong first parameter" fail:

```rego
package goast

exempt_funcs := {"main", "init"}

fail contains res if {
	input.Kind == "FuncDecl"
	regex.match(`^[A-Z]`, input.Node.Name.Name)   # exported names only
	not input.Node.Name.Name in exempt_funcs
	not endswith(input.FileName, "_test.go")
	input.Node.Body != null                        # skip forward declarations (null is truthy in Rego!)
	not first_param_is_context

	res := {
		"msg": sprintf("exported %s must take context.Context as its first argument", [input.Node.Name.Name]),
		"pos": input.Node.Name.NamePos,
		"sev": "ERROR",
	}
}

first_param_is_context if {
	p := input.Node.Type.Params.List[0]
	p.Type.X.Name == "context"
	p.Type.Sel.Name == "Context"
}
```

Check `input.Node.Recv` to target only methods (or only plain functions), or
`input.Node.Type.Results` for return-type rules.

## 5. Forbid import

**Convention.** A low-level package must not be imported directly — callers go
through an internal wrapper. Here, `database/sql` is hidden behind a repository
package.

```go
// bad
import "database/sql"

// good — use the repository package instead
import "github.com/my-org/app/repository"
```

Import rules anchor on `File` and scan the flattened `Imports` list. Note
`Path.Value` keeps its surrounding quotes, so `trim` them off:

```rego
package goast

fail contains res if {
	input.Kind == "File"
	some imp in input.Node.Imports
	trim(imp.Path.Value, "\"") == "database/sql"

	res := {
		"msg": "do not import database/sql directly; go through the repository package",
		"pos": imp.Path.ValuePos,
		"sev": "ERROR",
	}
}
```

To scope the ban to one layer, guard with the path — e.g.
`startswith(input.Path, "internal/domain/")` — so only that directory is
restricted.

## 6. Ban time.Now

**Convention.** Code must not read the wall clock directly; time comes from an
injected clock so behaviour is testable and deterministic.

```go
// bad
now := time.Now()

// good
now := clock.Now()   // clock is injected
```

Same `CallExpr` + `SelectorExpr` shape as example 1:

```rego
package goast

fail contains res if {
	input.Kind == "CallExpr"
	input.Node.Fun.X.Name == "time"
	input.Node.Fun.Sel.Name == "Now"

	res := {
		"msg": "do not call time.Now() directly; inject a clock for testability",
		"pos": input.Node.Fun.X.NamePos,
		"sev": "WARNING",
	}
}
```

## Pinning with goast test

A policy is only trustworthy if you know it fires on the bad case and stays quiet
on the good one. Keep a bad and a good sample next to each rule and pin both with
`goast test` (see [usage.md](usage.md#goast-test) and
[configuration.md](configuration.md#test-section)):

```toml
[test]
policy = ["policy"]

[[test.cases]]
name   = "bans fmt.Errorf"
source = ["policy/testdata/errorf_bad.go"]
expect = "fail"
count  = 1

[[test.cases]]
name   = "allows goerr.Wrap"
source = ["policy/testdata/errorf_good.go"]
expect = "pass"
```

```bash
goast test
```

```
ok    bans fmt.Errorf  (1 violation)
ok    allows goerr.Wrap  (0 violations)

2 cases: 2 passed, 0 failed
```

Wire `goast test` and `goast eval` into CI so neither the rules nor the code can
drift unnoticed — see [ci.md](ci.md).
