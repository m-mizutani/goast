# CLI usage

goast is a single binary with four subcommands:

| Subcommand | Purpose |
|---|---|
| [`eval`](#goast-eval) | Check Go source against a Rego policy — the main command |
| [`test`](#goast-test) | Run declarative policy tests from `.goast.toml` |
| [`dump`](#goast-dump) | Print a Go file's AST as JSON (for writing/​debugging policies) |
| [`sync`](#goast-sync) | Write AST dumps to files marked by comments |

For the `.goast.toml` schema shared by `eval` and `test`, see
[configuration.md](configuration.md). For writing the Rego policies themselves,
see the [`goast-policy` skill](../skills/goast-policy/SKILL.md).

## Evaluation model

`goast eval -p <policy> <target>` does the following:

1. Walks each `<target>` path recursively and selects every `*.go` file.
2. Parses each file (comments included) into a `go/ast` tree.
3. Visits **every node** of the tree (`ast.Inspect` order: `File` first, then
   declarations, statements, expressions, down to identifiers).
4. For each node, evaluates the Rego query `data.goast.fail` with that node as
   `input`, and collects the resulting `fail` set.

A policy is therefore a set of pattern matches over **single AST nodes** — a
rule cannot see sibling nodes, other files, or type information. See the
[skill](../skills/goast-policy/SKILL.md) for the consequences (and how to avoid
duplicate findings).

## Global flags

These apply to every subcommand and control logging (diagnostics go to stderr by
default, keeping stdout clean for results):

| Flag | Env | Default | Values |
|---|---|---|---|
| `--log-level, -l` | `GOAST_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `--log-format` | `GOAST_LOG_FORMAT` | `text` | `text` / `json` |
| `--log-output` | `GOAST_LOG_OUTPUT` | `stderr` | `stdout` / `stderr` / a file path |

## `goast eval`

Inspect and check Go code with a Rego policy.

```bash
goast eval -p ./policy/do_not_use_println.rego examples/println/main.go
```

```
[examples/println/main.go:6] - do not use fmt.Println

	fmt.Println("hello")
	~~~~~~~~~~~~~~~~~~~~


	Detected 1 violations
```

**Flags**

| Flag | Env | Default | Meaning |
|---|---|---|---|
| `--policy, -p` | | | Policy file or directory (repeatable). Directories are walked for `*.rego`. May instead come from `[eval].policy`. |
| `--format, -f` | | `text` | Output format: `text` or `json`. |
| `--output, -o` | | `-` (stdout) | Output file; `-` means stdout. |
| `--fail` | | off | Exit non-zero when any violation is reported. Off by default (goast only reports). |
| `--ignore-auto-generated` | | off | Skip auto-generated Go files. |
| `--workers, -w` | `GOAST_WORKERS` | `0` | Files evaluated in parallel; `0` = one per CPU, `1` = sequential. |
| `--config, -c` | `GOAST_CONFIG` | `.goast.toml` | Config file supplying `[eval]` defaults. |
| `--object-depth, -d` | | `0` | Levels of the `*ast.Object` graph to restore in `input` (see below). |

**Targets and config.** You can pass targets as positional arguments
(`./...`, a directory, or individual files) or set `[eval].paths` in
`.goast.toml`. With policy, paths, and options committed to `.goast.toml`, CI can
run a bare `goast eval`. CLI flags always override the config — see
[configuration.md](configuration.md).

```bash
goast eval -p ./policy -w 8 ./...     # explicit policy, 8 workers, recurse from cwd
goast eval                            # everything from [eval] in .goast.toml
```

**Output formats.** `text` (default) prints each violation with the offending
source line underlined, then a `Detected N violations` summary. `json` emits the
[reviewdog Diagnostic Format](https://github.com/reviewdog/reviewdog/tree/master/proto/rdf#rdjson)
(rdjson) on stdout, ready to pipe into reviewdog — see [ci.md](ci.md).

```bash
goast eval -p ./policy -f json ./cmd
```

**Parallelism and ordering.** Files are evaluated concurrently, but the order of
reported violations is deterministic (source walk order) regardless of the
worker count, so output is stable in CI.

**Exit code.** `0` when the run completes — *even if violations were printed* —
unless `--fail` is set, which turns any violation into a non-zero exit. A
run-time error (unreadable policy, bad path) always exits non-zero. See
[ci.md](ci.md#exit-codes) for how to gate a build.

**`--object-depth` and the object graph.** To break the one reference cycle in
`go/ast` (`*ast.Object`), goast parses with object resolution skipped, so
`input.Node.Scope`, `input.Node.Unresolved`, and `*ast.Ident.Obj` are `null`.
These fields are deprecated in `go/ast`; a policy that genuinely needs the object
graph can restore N levels with `--object-depth`. Only the `fail` rule of the
`goast` package is queried per node — helper rules and same-package `*_test.rego`
rules are not evaluated on every node, which keeps `eval` fast on large trees.

## `goast test`

Writing a policy is only half the job — you also need to confirm it fires on the
code it should and stays quiet on the code it should not. `goast test` runs your
policies against sample Go files and checks the result against expectations
declared in `.goast.toml`. It exercises the **real** evaluation pipeline (parse
Go → walk AST → run Rego), so a passing test means `goast eval` behaves the same.

```bash
goast test                    # reads [test] from ./.goast.toml
goast test -c config/goast.toml
```

A minimal `[test]` table:

```toml
[test]
policy = [".goast"]           # default policy for the cases below

[[test.cases]]
name   = "forbids fmt.Println"
source = [".goast/testdata/no_println_bad.go"]
expect = "fail"
count  = 1

[[test.cases]]
name   = "allows fmt.Fprintf"
source = [".goast/testdata/no_println_good.go"]
expect = "pass"
```

```
ok    forbids fmt.Println  (1 violation)
ok    allows fmt.Fprintf  (0 violations)

2 cases: 2 passed, 0 failed
```

`goast test` exits non-zero if any case fails, or if the config or a policy
cannot be loaded, so it works as a CI gate on its own. The full `[test]` schema —
`count`, `violations`, per-case overrides — is in
[configuration.md](configuration.md#test-section).

## `goast dump`

Print a Go file's AST as JSON. This is how you discover the exact node shape a
policy must match — the dump schema is identical to the `input.Node` a policy
receives.

```bash
goast dump --line 6 examples/println/main.go
```

```json
{
  "Path": "examples/println/main.go",
  "FileName": "main.go",
  "DirName": "examples/println",
  "Node": {
    "X": {
      "Fun": {
        "X": { "NamePos": 44, "Name": "fmt", "Obj": null },
        "Sel": { "NamePos": 48, "Name": "Println", "Obj": null }
      },
      "Lparen": 55,
      "Args": [ { "ValuePos": 56, "ValueEnd": 63, "Kind": 9, "Value": "\"hello\"" } ],
      "Ellipsis": 0,
      "Rparen": 63
    }
  },
  "Kind": "ExprStmt"
}
```

**Flags**

| Flag | Meaning |
|---|---|
| `--line, -l` | Only dump node(s) at the given line (repeatable). |
| `--func, -n` | Only dump the given function by name (repeatable). |
| `--object-depth, -d` | Levels of the `*ast.Object` graph to include (default `0`). |
| `--root-only, -r` | Dump only the top node, no recursion. |
| `--all-matched, -a` | Emit every node matching the condition, not just the first per line. |
| `--compact, -c` | Compact (single-line) JSON instead of pretty-printed. |

Pipe into `jq` to explore, or into `opa eval` to replay a node against a policy
(see [Debugging](#debugging-with-opa)).

## `goast sync`

Write AST dumps of specific nodes to files, driven by `// goast.sync: <path>`
comments in the source. When goast visits a node whose line carries such a
comment (or the file's first line, which targets the whole `*ast.File`), it
writes that node's JSON to `<path>`. This is a convenience for keeping a checked-in
AST snapshot next to the code it describes — for example, to hand a known-good
`input` shape to a policy test.

```go
package main

// goast.sync: testdata/main_call.json
func main() {
	fmt.Println("hello") // goast.sync: testdata/println.json
}
```

```bash
goast sync ./...
```

| Flag | Meaning |
|---|---|
| `--compact, -c` | Compact (single-line) JSON instead of pretty-printed. |

## Writing a first policy

A goast policy lives in `package goast` and contributes to one set, `fail`.
Every rule matches a single AST node and, on a match, adds a result object:

```rego
package goast

fail contains res if {
	input.Kind == "ExprStmt"
	input.Node.X.Fun.X.Name == "fmt"
	input.Node.X.Fun.Sel.Name == "Println"

	res := {
		"msg": "do not use fmt.Println",
		"pos": input.Node.X.Fun.X.NamePos,
		"sev": "ERROR",
	}
}
```

The contract:

- **Input** — `Path` (file path), `FileName`, `DirName`, `Kind` (the node type,
  e.g. `ExprStmt`), and `Node` (the AST node itself, JSON-marshalled).
- **Output** — a `fail` set of `{msg, pos, sev}` objects, where `pos` is an
  integer position field (e.g. `NamePos`) that goast converts to line/column,
  `msg` is the message, and `sev` is `INFO`, `WARNING`, or `ERROR`.

Policies are evaluated as [Rego v1](https://www.openpolicyagent.org/docs/latest/v0-upgrade/),
so rules use the `contains` / `if` keywords (no `import rego.v1` needed).

For the full input format, token tables, and a catalogue of recipes, see the
[`goast-policy` skill](../skills/goast-policy/SKILL.md), and
[examples.md](examples.md) for complete, runnable policies that enforce
organization-specific conventions.

## Debugging with opa

Because `goast dump` output *is* the policy input, you can replay any node
against your policy with the stock `opa` command:

```bash
goast dump -l 6 examples/println/main.go | opa eval -b ./policy/ -I data
```

```json
{
  "result": [
    { "expressions": [ { "value": { "goast": { "fail": [
      { "msg": "do not use fmt.Println", "pos": 44, "sev": "ERROR" }
    ] } }, "text": "data", "location": { "row": 1, "col": 1 } } ] }
  ]
}
```

An empty `fail` with no error almost always means one condition is *undefined*
(an absent field), not false — dump the node and check each path.
