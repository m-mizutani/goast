# goast [![test](https://github.com/m-mizutani/goast/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/test.yml)[![gosec](https://github.com/m-mizutani/goast/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/gosec.yml) [![trivy](https://github.com/m-mizutani/goast/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/trivy.yml)

Go [AST](https://pkg.go.dev/go/ast) (Abstract Syntax Tree) based static analysis tool with [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/).

![](https://user-images.githubusercontent.com/605953/187052104-03525b0d-cb7c-44b9-b395-b7b3692a0cc2.png)

## Motivation

There are a lot of static analysis tools for Go language. They inspect Go source code with best practices. However, we need to care not only common best practice but also internal rules of individual, team or organization when reviewing code. Additionally, some kind of function and resource has a rule to use them (e.g. required initialize at first). It may be difficult to check such original rules by a common static analysis tool.

`goast` is static analysis tool with [OPA]([Rego](https://www.openpolicyagent.org/docs/latest/policy-language/)) that is generic policy engine of [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/). It can separate static analysis tool to _implementation_ and _policy_ that a user can customize.

## Features

- Dump Go AST as JSON format (excluding ast.Object to avoid reference cycle)
- Evaluate Go source code with Rego policy
- Output the evaluation result as text or JSON format (compatible with [reviewdog](https://github.com/reviewdog/reviewdog))
- Test your policies against sample Go files declaratively with a `.goast.toml` config (`goast test`)

## Usage

### Install

```
$ go install github.com/m-mizutani/goast/cmd/goast@latest
```

Pre-built binaries for Linux, macOS and Windows (amd64/arm64) are also attached to each [GitHub Release](https://github.com/m-mizutani/goast/releases).

### Dump source code to confirm AST

`dump` subcommand output AST as JSON format. `--line` can specify line number and `--func` can specify function name for dump.

```bash
$ goast dump --line 6 examples/println/main.go | jq
{
  "Path": "examples/println/main.go",
  "Node": {
    "X": {
      "Fun": {
        "X": {
          "NamePos": 44,
          "Name": "fmt",
          "Obj": null
        },
        "Sel": {
          "NamePos": 48,
          "Name": "Println",
          "Obj": null
        }
      },
      "Lparen": 55,
      "Args": [
        {
          "ValuePos": 56,
          "Kind": 9,
          "Value": "\"hello\""
        }
      ],
      "Ellipsis": 0,
      "Rparen": 63
    }
  },
  "Kind": "ExprStmt"
}
```

### Write Rego policy

Here is example of a policy to prohibit `fmt.Println`. Policies are evaluated as [Rego v1](https://www.openpolicyagent.org/docs/latest/v0-upgrade/), so rules use the `contains` / `if` keywords (no `import rego.v1` is required).

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

`goast`'s policy rule is following.

- Package name: `goast`
- Input
  - `Path`: Source code file path
  - `Node`: Dumped AST (without *ast.Object)
  - `Kind`: Type of node
- Output
  - `fail`: A set of violation results
    - `pos`: Integer of *Pos (e.g. `NamePos`). It will be converted to line number and column of source code file
    - `msg`: Error message
    - `sev`: Severity. Choose one from `INFO`, `WARNING` or `ERROR`

### Evaluation

`eval` subcommand evaluates go source code (file or directly recursively) with Rego policy file(s).

```bash
$ goast eval -p ./policy/do_not_use_println.rego examples/println/main.go
[examples/println/main.go:6] - do not use fmt.Println

        Detected 1 violations

```

`--format, -f` option can specify output format `text` or `json`. JSON schema is according to [Reviewdog Diagnostic Format](https://github.com/reviewdog/reviewdog/tree/master/proto/rdf#rdjson).

Files are evaluated in parallel. `--workers, -w` (or `GOAST_WORKERS`) sets the number of files evaluated concurrently; `0` (the default) uses one worker per CPU, and `1` forces sequential evaluation. The order of reported violations is deterministic (source walk order) regardless of the worker count.

```bash
$ goast eval -p ./policy --workers 8 ./...
```

By default `goast eval` also reads an `[eval]` section from `.goast.toml` in the
current directory, so the policy, target paths, and output options can be
committed once and a bare `goast eval` run in CI:

```toml
[eval]
policy = ["policy"]                # policy file(s) or dir(s); always an array
paths  = ["./cmd", "./internal"]   # evaluation targets used when no arguments are given
fail   = true                      # non-zero exit on any violation (CI gate)
# format = "json"                  # "text" (default) or "json"
# output = "goast.rdjson"          # omit for stdout (the CLI "-" sentinel is not used in config)
# ignore_auto_generated = true
# workers = 8
# object_depth = 0
```

```bash
$ goast eval                       # policy and targets come from [eval]
$ goast eval -f json ./cmd         # CLI flags/args override the config
```

CLI flags always win over the config, which wins over the built-in defaults
(**CLI > `[eval]` > default**). Point at another file with `--config, -c` or the
`GOAST_CONFIG` environment variable. If the default `.goast.toml` is absent,
`eval` simply runs from CLI flags alone; an explicitly named `--config` that does
not exist is an error. Paths (`policy`, `paths`) are always arrays — a single
value is written as a one-element array.

> **Note on evaluation internals.** `eval` queries the `fail` rule of the `goast` package directly (`data.goast.fail`). Other rules in your policy package — helper rules, and in particular the test rules in same-package `*_test.rego` files — are therefore not evaluated on every AST node, which is what keeps `eval` fast on large trees. This does not change which violations are reported: `fail` was always the only rule `eval` consumed.
>
> To break the one reference cycle in `go/ast` (`*ast.Object`), `eval` parses with object resolution skipped, so `input.Node.Scope`, `input.Node.Unresolved` and `*ast.Ident.Obj` are `null`. These fields are deprecated in `go/ast`; a policy that genuinely needs the object graph can request it with `--object-depth, -d` (which restores the classic resolve path).

### Test policies

Writing a policy is only half the job — you also need to confirm it fires on the
code it should and stays quiet on the code it should not. The `test` subcommand
runs your policies against sample Go files and checks the result against
expectations you declare in a TOML config. It exercises the **real** evaluation
pipeline (parse Go → walk AST → run Rego), so a passing test means `goast eval`
will behave the same way.

By default `goast test` reads `.goast.toml` in the current directory (override
with `--config, -c`, or the `GOAST_CONFIG` environment variable). Each
`[[test.cases]]` entry names the policy, the Go source to run it over, and what
to expect:

```toml
[test]
# Default policy applied to every case (a case may override it). Always an array
# of file or directory paths (directories are walked recursively for *.rego).
policy = [".goast"]

# The bad sample must trigger the rule.
[[test.cases]]
name   = "forbids fmt.Println"
source = [".goast/testdata/no_println_bad.go"]
expect = "fail"            # "pass" (0 violations) or "fail" (>0 violations)
count  = 1                 # optional: assert the exact number of violations

# The good sample must stay clean.
[[test.cases]]
name   = "allows fmt.Fprintf"
source = [".goast/testdata/no_println_good.go"]
expect = "pass"

# Pin the exact finding a specific policy produces.
[[test.cases]]
name   = "reports the exact line and message"
policy = [".goast/no_println.rego"]   # override the default policy for this case
source = [".goast/testdata/no_println_bad.go"]

  [[test.cases.violations]]         # optional: each entry must match some finding
  line     = 6                      # every field here is optional...
  message  = "do not use fmt.Println"   # ...message is a substring match
  severity = "ERROR"                # ...severity is case-insensitive
```

```bash
$ goast test
ok    forbids fmt.Println  (1 violation)
ok    allows fmt.Fprintf  (0 violations)
ok    reports the exact line and message  (1 violation)

3 cases: 3 passed, 0 failed
```

`count` and each `violations` field are optional refinements on top of
`expect` — specify them when you want to pin the exact result, omit them to only
check pass/fail. Per-case `object_depth` and `ignore_auto_generated` mirror the
matching `eval` flags. `goast test` exits non-zero if any case fails (or if the
config or a policy cannot be loaded), so it works as a CI gate on its own.

`policy` and `source` paths are resolved relative to the **current working
directory** (not the config file's location), matching how `goast eval` takes
paths. Keep the config at the repository root and run `goast test` from there —
the intended CI usage.

### Debug

Also, you can use original `opa` command to debug policy. A schema of `dump` output is same with one to be evaluated. Then, `opa` command can use it with Rego file(s). An example is following.

```bash
goast . dump -l 6 examples/println/main.go | opa eval -b ./policy/ -I data
{
  "result": [
    {
      "expressions": [
        {
          "value": {
            "goast": {
              "fail": [
                {
                  "msg": "do not use fmt.Println",
                  "pos": 44,
                  "sev": "ERROR"
                }
              ]
            }
          },
          "text": "data",
          "location": {
            "row": 1,
            "col": 1
          }
        }
      ]
    }
  ]
}
```

## Claude Code Skill

This repository also ships a [Claude Code](https://claude.com/claude-code) skill, `goast-policy`, that guides Claude through writing and testing goast Rego policies (input format, AST JSON shapes, `go/token` integer tables, rule recipes). Install it as a plugin:

```
/plugin marketplace add m-mizutani/goast
/plugin install goast-policy@goast
```

Then ask Claude to write a goast policy (e.g. "write a goast policy that forbids fmt.Println"), or see [skills/goast-policy/SKILL.md](skills/goast-policy/SKILL.md) directly.

## License

Apache License v2
