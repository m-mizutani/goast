---
name: goast-policy
description: >-
  Write, review, and debug Rego policies for goast — the Go AST static analysis
  tool that lints Go source code with OPA. Use this skill whenever the user
  wants to create a custom Go lint rule with goast, write or fix a .rego file
  for `package goast`, understand goast's AST JSON input format, translate a Go
  coding convention ("forbid fmt.Println", "every function must take
  context.Context first") into a policy, or debug why a goast policy does not
  match. Trigger even if the user only says "add a goast rule" or shows a
  failing `goast eval` run.
---

# Writing goast Policies

goast (github.com/m-mizutani/goast) parses Go source files into `go/ast` trees
and evaluates a Rego policy **once per AST node**. A policy is therefore a set
of pattern matches over single nodes, not over the whole program at once.
This skill explains the input format, the rule contract, and a workflow that
avoids the common pitfalls (duplicate findings, integer token codes, untyped
interface fields).

## Evaluation model — read this first

`goast eval -p <policy-file-or-dir> <go-file-or-dir>`:

1. Walks the given paths recursively and picks every `*.go` file.
2. Parses each file (comments included) into an AST.
3. Visits **every node recursively** (`ast.Inspect` order: the `File` node
   first, then declarations, statements, expressions, down to identifiers).
4. For **each node**, evaluates the query `data.goast` with the node as
   `input`, and collects the `fail` set.

Consequences that shape how you write rules:

- A rule anchored on a deep pattern like `input.Node.X.Fun.Sel.Name` can match
  the same source code **multiple times** — once when the input is the
  `ExprStmt`, again when it is the inner `CallExpr` — producing duplicate
  findings. **Always constrain `input.Kind` first** to pin the rule to exactly
  one node type.
- Prefer the **smallest node kind that still carries all the information you
  need** (e.g. `CallExpr` for call bans, `FuncDecl` for signature rules,
  `File` for import/package-level rules).
- Each node is evaluated independently: a rule cannot see sibling nodes,
  other files, or type information. There is no cross-node state.

## Input format

Every evaluation receives this input:

```json
{
  "Path": "examples/println/main.go",
  "FileName": "main.go",
  "DirName": "examples/println",
  "Kind": "ExprStmt",
  "Node": { ... }
}
```

- `Kind` — the `go/ast` struct type name without the `*ast.` prefix:
  `"File"`, `"FuncDecl"`, `"CallExpr"`, `"AssignStmt"`, `"Ident"`, …
- `Node` — the AST node marshalled as JSON with **Go field names**
  (`Name`, `Fun`, `Args`, `NamePos`, …). `*ast.Object` back-references are
  nulled out to break cycles, so `"Obj": null` everywhere.
- Position fields (`NamePos`, `ValuePos`, `Lparen`, …) are **integer byte
  offsets** (`token.Pos`), not line numbers. goast converts the `pos` you
  return into line/column for you.
- Interface-typed fields (`ast.Expr`, `ast.Stmt`, `ast.Decl`) carry **no type
  tag** in JSON — you distinguish node types by their field shape (an `Ident`
  has `Name`; a `SelectorExpr` has `X` and `Sel`). Referencing a field that
  does not exist simply makes the rule body fail to match, which is safe.
- Operator and token fields (`Op`, `Tok`, `Kind` of a `BasicLit`) are
  **integers** from `go/token`. See [references/tokens.md](references/tokens.md)
  for the full value tables (e.g. `9` = STRING literal, `17` = `&`, `47` = `:=`).

For JSON shapes of the frequently used node kinds (File, FuncDecl, CallExpr,
AssignStmt, ImportSpec, …), see
[references/input-format.md](references/input-format.md).

## Rule contract

```rego
package goast

fail contains res if {
    input.Kind == "CallExpr"
    input.Node.Fun.X.Name == "fmt"
    input.Node.Fun.Sel.Name == "Println"

    res := {
        "msg": "do not use fmt.Println",
        "pos": input.Node.Fun.X.NamePos,
        "sev": "ERROR",
    }
}
```

- Package name must be `goast`. goast queries `data.goast` and reads only the
  `fail` set; helper rules and functions in the same package are fine.
- Each violation is an object with exactly these keys:
  - `msg` (string) — the message shown to the user.
  - `pos` (integer) — copy a `*Pos` field from the matched node, pointing at
    the offending token. goast converts it to file/line/column.
  - `sev` (string) — one of `"INFO"`, `"WARNING"`, `"ERROR"`. It is carried in
    results but not currently rendered in text/rdjson output; set it anyway
    for forward compatibility.
- Multiple `.rego` files given via `-p` (files or directories, walked
  recursively) are compiled together; `fail contains` rules accumulate across
  files because they share the package.

## Rego v1 — syntax rules

goast compiles every policy as **Rego v1** (no `import rego.v1` needed;
writing it is harmless). This means:

- Partial set rules must use `contains` + `if`:
  `fail contains res if { ... }`. The v0 form `fail[res] { ... }` **fails to
  compile**.
- `if` is required before every rule body, including helpers:
  `is_test_file if { endswith(input.FileName, "_test.go") }`.
- `every`, `in`, and `some ... in` are available without imports:

```rego
    every stmt in input.Node.Body.List {
        stmt.X.Fun.Name != "logging"
    }
```

- Iterate collections with `some arg in input.Node.Args` (or the classic
  `input.Node.Args[_]`).
- Build messages with `sprintf`:
  `sprintf("%s must take context.Context", [input.Node.Name.Name])`.
- `print()` produces no output inside goast (print statements are not
  enabled in its engine). Debug with the `opa` CLI instead — see below.

## Workflow: from convention to policy

**1. Write a minimal Go sample** that contains the violation (and ideally a
compliant counterpart).

**2. Dump the AST of the offending line** to learn the exact JSON shape you
must match — never guess the structure:

```bash
goast dump --line 6 sample.go | jq        # nodes touching line 6
goast dump --func MyFunc sample.go | jq   # a whole function declaration
goast dump --root-only sample.go | jq     # the File node only
```

Without a filter, `dump` prints **every** node recursively — huge even for
small files. Always narrow with `--line` / `--func` / `--root-only`
(`--line` prints only the first node starting on that line; add
`--all-matched` to see every node on the line).

**3. Pick the anchor kind.** Find the smallest `Kind` in the dump that
contains everything the rule needs, and start the rule with
`input.Kind == "..."`.

**4. Write the rule** against the dumped shape. Copy field names verbatim
from the dump output. Pick `pos` from a `*Pos` field near the offending
token.

**5. Test it:**

```bash
goast eval -p policy.rego sample.go
```

Expected output on a match:

```
[sample.go:6] - do not use fmt.Println
...
        Detected 1 violations
```

**6. Debug non-matches with the opa CLI.** `dump` output is byte-for-byte the
evaluation input, so you can replay it:

```bash
goast dump --line 6 sample.go | opa eval -b ./policy/ -I data.goast
```

If `fail` is empty, bisect the rule body: comment out conditions one by one,
or `opa eval ... 'input.Node.Fun'` to inspect what the input actually holds
at each path.

**7. Run against the real tree, then wire into CI:**

```bash
goast eval -p ./policy ./...        # directories are walked recursively
goast eval -p ./policy --fail .     # non-zero exit on violations (CI gate)
goast eval -p ./policy -f json .    # Reviewdog Diagnostic Format (rdjson)
goast eval -p ./policy --ignore-auto-generated .   # skip "Code generated" files
```

## Verifying and testing policies

Three layers, cheapest first. Use all three for policies that will gate CI.

**1. End-to-end check with sample files.** Keep a known-violating and a
known-compliant Go sample next to the policy and run both — a policy that
never fires and a policy that fires on everything look identical if you only
test one side:

```bash
goast eval -p ./policy testdata/bad.go     # expect: Detected N violations
goast eval -p ./policy testdata/good.go    # expect: Detected 0 violations
goast eval -p ./policy --fail testdata/bad.go || echo "exit code is non-zero as expected"
```

**2. Replay the exact input with the opa CLI.** `goast dump` output is
byte-for-byte the evaluation input, so any node can become a fixture:

```bash
goast dump --line 6 testdata/bad.go > testdata/println_node.json
opa eval -b ./policy -I 'data.goast.fail' < testdata/println_node.json
```

**3. Unit tests with `opa test`.** Write `test_*` rules in `package goast`
and feed hand-crafted (or dump-captured) inputs via `with input as`:

```rego
package goast

println_input := {
    "Path": "main.go", "FileName": "main.go", "DirName": ".",
    "Kind": "CallExpr",
    "Node": {"Fun": {
        "X": {"Name": "fmt", "NamePos": 44},
        "Sel": {"Name": "Println", "NamePos": 48},
    }},
}

test_println_detected if {
    some res in fail with input as println_input
    res.sev == "ERROR"
    res.pos == 44
}

test_sprintf_allowed if {
    count(fail) == 0 with input as json.patch(println_input,
        [{"op": "replace", "path": "/Node/Fun/Sel/Name", "value": "Sprintf"}])
}
```

```bash
opa test -v ./policy ./policy-test    # opa needs both dirs to compile the package
opa check ./policy                    # quick syntax/compile lint (OPA v1)
```

Keep test files **outside** the directory passed to `goast eval -p`: goast
walks it recursively and compiles every `.rego` it finds, so test rules would
be re-evaluated on every AST node (only `fail` is read, but you pay the
evaluation cost thousands of times). A layout that works:

```
policy/               # goast eval -p ./policy
    no_println.rego
policy-test/          # opa test ./policy ./policy-test
    no_println_test.rego
    testdata/*.go     # bad/good samples for layer 1
```

Assert on the full result — `msg`, `pos`, and `sev` — at least once per rule;
`pos` mistakes (wrong field, line number instead of offset) survive
count-only tests and surface as findings pointing at the wrong line.

## Common recipes

Ready-made, tested rule patterns (ban a qualified call, restrict imports,
enforce signatures, forbid direct struct assignment, require a call inside
function bodies, file-level rules, scoping by file name) live in
[references/patterns.md](references/patterns.md). Start from the closest
recipe rather than from scratch.

Two shapes worth memorizing:

```rego
# Qualified call fmt.Println — Fun is a SelectorExpr {X: Ident, Sel: Ident}
fail contains res if {
    input.Kind == "CallExpr"
    input.Node.Fun.X.Name == "fmt"
    input.Node.Fun.Sel.Name == "Println"
    res := {"msg": "do not use fmt.Println", "pos": input.Node.Fun.X.NamePos, "sev": "ERROR"}
}

# Import ban — anchor on File so each import is reported once
fail contains res if {
    input.Kind == "File"
    some imp in input.Node.Imports
    imp.Path.Value == "\"unsafe\""    # BasicLit Value keeps the quotes
    res := {"msg": "do not import unsafe", "pos": imp.Path.ValuePos, "sev": "ERROR"}
}
```

## Pitfall checklist

Before shipping a policy, verify:

- [ ] Every rule starts with an `input.Kind == "..."` guard (no duplicate
      findings from parent/child node visits).
- [ ] Rego v1 syntax: `fail contains res if { ... }`, `if` on all helpers.
- [ ] String literals compared against `BasicLit.Value` include the quotes
      (`"\"unsafe\""`), or use `trim(v, "\"")` first.
- [ ] Operator/token comparisons use the integer codes from
      [references/tokens.md](references/tokens.md), with a comment naming the
      token (`rhs.Op == 17  # token.AND (&)`).
- [ ] `pos` comes from a `*Pos` integer field of the matched node — never a
      line number.
- [ ] Negative conditions tolerate absent fields: `input.Node.X.Name != "foo"`
      is **undefined** (rule silently skips) when `X` is not an `Ident`; use
      `not` with a helper rule when you need "does not contain" semantics.
- [ ] Null checks are explicit. Nil AST pointers marshal as JSON `null`, and
      `null` is **truthy** in Rego — write `input.Node.Doc == null` /
      `input.Node.Body != null`, never bare `input.Node.Body` or
      `not input.Node.Doc`.
- [ ] Tested with both a violating and a compliant sample file.
