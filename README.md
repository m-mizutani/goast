# goast [![test](https://github.com/m-mizutani/goast/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/test.yml)[![goast](https://github.com/m-mizutani/goast/actions/workflows/goast.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/goast.yml)[![gosec](https://github.com/m-mizutani/goast/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/gosec.yml) [![trivy](https://github.com/m-mizutani/goast/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/goast/actions/workflows/trivy.yml)

Go [AST](https://pkg.go.dev/go/ast) based static analysis for Go, with rules written as [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) policies. Encode your team's own conventions once and enforce them in CI.

![](https://user-images.githubusercontent.com/605953/187052104-03525b0d-cb7c-44b9-b395-b7b3692a0cc2.png)

## Motivation

Go has many excellent static analysis tools, but they check *common* best
practices. Every team also has its *own* rules — internal conventions, "this
resource must be initialized first", "use our error wrapper, not `fmt.Errorf`" —
that a generic linter cannot know about. goast splits a linter into
_implementation_ (goast itself) and _policy_ (Rego you write), so you can express
those rules yourself. It runs [OPA](https://www.openpolicyagent.org/)/Rego over
the Go AST, one node at a time.

## Features

- Evaluate Go source against Rego policies you control
- Report findings as text or JSON (compatible with [reviewdog](https://github.com/reviewdog/reviewdog))
- Test policies against sample files declaratively (`goast test`), so rules can't drift
- Dump the Go AST as JSON to discover exactly what your policy should match

## Getting started

Install with `go install` (pre-built binaries for Linux, macOS, and Windows are
also on each [GitHub Release](https://github.com/m-mizutani/goast/releases)):

```bash
go install github.com/m-mizutani/goast/cmd/goast@latest
```

Write a policy in `package goast`. Each rule matches one AST node and adds to the
`fail` set. This one forbids `fmt.Println`:

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

Evaluate a file (or a directory, or `./...`) against it:

```bash
$ goast eval -p ./policy/do_not_use_println.rego examples/println/main.go
[examples/println/main.go:6] - do not use fmt.Println

	fmt.Println("hello")
	~~~~~~~~~~~~~~~~~~~~


	Detected 1 violations
```

Pass `--fail` to make violations exit non-zero — that is your CI gate.

## How it works

goast parses each Go file into a `go/ast` tree and evaluates the query
`data.goast.fail` against **every node**, with the node as `input`. A policy is
therefore a set of pattern matches over single nodes: `input.Kind` is the node
type, `input.Node` is the node itself, and each match returns a `{msg, pos, sev}`
result. Use `goast dump` to see the exact shape of any node.

## Documentation

- **[docs/usage.md](docs/usage.md)** — full CLI reference: `eval`, `test`, `dump`, `sync`, and every flag.
- **[docs/configuration.md](docs/configuration.md)** — the `.goast.toml` schema (`[eval]` / `[test]`) and flag precedence.
- **[docs/ci.md](docs/ci.md)** — wiring goast into GitHub Actions, exit codes, and reviewdog PR comments.
- **[docs/examples.md](docs/examples.md)** — complete, runnable policies that enforce organization-specific conventions.
- **[goast-policy skill](skills/goast-policy/SKILL.md)** — deep guide to writing Rego policies (AST input format, node shapes, recipes).

## Claude Code Skill

This repository ships a [Claude Code](https://claude.com/claude-code) skill,
`goast-policy`, that guides Claude through writing and testing goast policies.
Install it as a plugin:

```
/plugin marketplace add m-mizutani/goast
/plugin install goast-policy@goast
```

Then ask Claude to write a policy (e.g. "write a goast policy that forbids
`fmt.Println`"), or read [skills/goast-policy/SKILL.md](skills/goast-policy/SKILL.md).

## License

Apache License v2
