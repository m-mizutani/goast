# goast policy recipes

Copy the closest recipe and adapt it. Every recipe is anchored on a single
`input.Kind` (see SKILL.md for why), written in Rego v1, and returns
`{msg, pos, sev}` objects into the `fail` set. All rules live in
`package goast` — split them across as many `.rego` files as you like; the
`fail` sets merge.

## Table of contents

1. [Ban a qualified call (`fmt.Println`)](#1-ban-a-qualified-call)
2. [Ban a builtin or local call (`panic`)](#2-ban-a-builtin-or-local-call)
3. [Import rules (ban, dot-import, allowlist)](#3-import-rules)
4. [Enforce a function signature (`context.Context` first)](#4-enforce-a-function-signature)
5. [Forbid direct composite-literal assignment](#5-forbid-direct-composite-literal-assignment)
6. [Require a call somewhere in a function body (walk)](#6-require-a-call-somewhere-in-a-function-body)
7. [Ban a statement kind (goroutines)](#7-ban-a-statement-kind)
8. [Scope rules by file path](#8-scope-rules-by-file-path)
9. [Comment / doc rules](#9-comment--doc-rules)

## 1. Ban a qualified call

`pkg.Func(...)` calls have `Fun` as a `SelectorExpr`. Anchoring on `CallExpr`
(not `ExprStmt`) also catches calls used as expressions (`x := fmt.Sprintf(...)`
style usage of a banned function, calls inside arguments, etc.).

```rego
package goast

banned_calls := {
    ["fmt", "Println"],
    ["fmt", "Printf"],
    ["log", "Fatal"],
}

fail contains res if {
    input.Kind == "CallExpr"
    some entry in banned_calls
    pkg := entry[0]
    name := entry[1]
    input.Node.Fun.X.Name == pkg
    input.Node.Fun.Sel.Name == name

    res := {
        "msg": sprintf("do not use %s.%s", [pkg, name]),
        "pos": input.Node.Fun.X.NamePos,
        "sev": "ERROR",
    }
}
```

Caveat: this matches on names, not resolved packages. An aliased import
(`import p "fmt"; p.Println(...)`) slips through, and a local variable named
`fmt` false-positives. When that matters, add an import-alias rule (recipe 3)
alongside.

## 2. Ban a builtin or local call

Unqualified calls have `Fun` as a plain `Ident`.

```rego
fail contains res if {
    input.Kind == "CallExpr"
    input.Node.Fun.Name == "panic"

    res := {
        "msg": "do not call panic; return an error instead",
        "pos": input.Node.Fun.NamePos,
        "sev": "ERROR",
    }
}
```

## 3. Import rules

Anchor on `File` and use the flattened `Imports` list. Remember
`Path.Value` keeps its quotes.

```rego
# Ban an import
fail contains res if {
    input.Kind == "File"
    some imp in input.Node.Imports
    trim(imp.Path.Value, "\"") == "unsafe"

    res := {
        "msg": "do not import unsafe",
        "pos": imp.Path.ValuePos,
        "sev": "ERROR",
    }
}

# Ban dot imports (Name is null when unaliased, {"Name": "."} for dot imports)
fail contains res if {
    input.Kind == "File"
    some imp in input.Node.Imports
    imp.Name.Name == "."

    res := {
        "msg": sprintf("do not dot-import %s", [imp.Path.Value]),
        "pos": imp.Name.NamePos,
        "sev": "WARNING",
    }
}

# Allowlist: only permitted prefixes may be imported in this package
allowed_prefixes := ["github.com/my-org/", "context", "fmt", "log/slog"]

fail contains res if {
    input.Kind == "File"
    startswith(input.Path, "pkg/domain/")
    some imp in input.Node.Imports
    path := trim(imp.Path.Value, "\"")
    not import_allowed(path)

    res := {
        "msg": sprintf("%s must not be imported from pkg/domain", [path]),
        "pos": imp.Path.ValuePos,
        "sev": "ERROR",
    }
}

import_allowed(path) if {
    some prefix in allowed_prefixes
    startswith(path, prefix)
}
```

## 4. Enforce a function signature

"Every function's first parameter must be `context.Context`." Use a helper
rule plus `not`, so that both "no parameters" and "wrong first parameter"
fail. A parameter type `context.Context` is a `SelectorExpr`.

```rego
exempt_funcs := {"main", "init"}

fail contains res if {
    input.Kind == "FuncDecl"
    not input.Node.Name.Name in exempt_funcs
    not endswith(input.FileName, "_test.go")
    input.Node.Body != null               # skip forward declarations (null is truthy in Rego!)
    not first_param_is_context

    res := {
        "msg": sprintf("%s must take context.Context as its first argument", [input.Node.Name.Name]),
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

Variants: check `input.Node.Recv` to target only methods (or only plain
functions); check `input.Node.Type.Results` for return-type rules (e.g.
"exported functions must return error last").

## 5. Forbid direct composite-literal assignment

"Construct `User` only via `NewUser()`." A value literal `User{...}` is a
`CompositeLit` in `Rhs`; `&User{...}` wraps it in a `UnaryExpr` with
`Op: 17` (`token.AND`).

```rego
fail contains res if {
    input.Kind == "AssignStmt"
    some rhs in input.Node.Rhs
    rhs.Type.Name == "User"               # CompositeLit: User{...}

    res := {
        "msg": "do not construct User directly; use NewUser()",
        "pos": rhs.Type.NamePos,
        "sev": "ERROR",
    }
}

fail contains res if {
    input.Kind == "AssignStmt"
    some rhs in input.Node.Rhs
    rhs.Op == 17                          # token.AND: &User{...}
    rhs.X.Type.Name == "User"

    res := {
        "msg": "do not construct User directly; use NewUser()",
        "pos": rhs.X.Type.NamePos,
        "sev": "ERROR",
    }
}
```

For qualified types (`model.User{}`), the literal's `Type` is a
`SelectorExpr`: match `rhs.Type.X.Name == "model"` and
`rhs.Type.Sel.Name == "User"`. Note this anchor misses `var u = User{}`
(that is a `ValueSpec`) — add a sibling rule anchored on `ValueSpec` scanning
`Values` if you need it.

## 6. Require a call somewhere in a function body

Checking only `Body.List` sees just top-level statements. To search the whole
subtree — any nesting depth — use Rego's built-in `walk`, which visits every
`[path, value]` pair of a JSON document. This is the JSON-side counterpart of
`ast.Inspect`.

```rego
fail contains res if {
    input.Kind == "FuncDecl"
    input.Node.Name.Name != "main"
    input.Node.Body != null
    not calls_anywhere(input.Node.Body, "logging")

    res := {
        "msg": sprintf("%s() must call logging()", [input.Node.Name.Name]),
        "pos": input.Node.Name.NamePos,
        "sev": "ERROR",
    }
}

calls_anywhere(subtree, name) if {
    walk(subtree, [_, value])
    value.Fun.Name == name
}
```

`walk` is equally useful for "X must NOT appear anywhere inside Y"
(e.g. no `time.Sleep` inside test helpers): call `calls_anywhere`-style
helpers positively instead of under `not`.

## 7. Ban a statement kind

Some rules are just "this node kind must not appear". `GoStmt`, `DeferStmt`,
`SelectStmt`, `TypeAssertExpr`, … all work the same way.

```rego
fail contains res if {
    input.Kind == "GoStmt"

    res := {
        "msg": "do not launch raw goroutines; use the async-dispatch helper",
        "pos": input.Node.Go,
        "sev": "ERROR",
    }
}
```

(`input.Node.Go` is the position of the `go` keyword — statement nodes carry
their keyword position under the keyword's name.)

## 8. Scope rules by file path

`Path`, `FileName`, and `DirName` are part of every input. Package the
predicates as helpers and reuse them across rules.

```rego
is_test_file if endswith(input.FileName, "_test.go")

in_domain_layer if startswith(input.Path, "pkg/domain/")

fail contains res if {
    input.Kind == "CallExpr"
    not is_test_file
    input.Node.Fun.X.Name == "os"
    input.Node.Fun.Sel.Name == "Getenv"

    res := {
        "msg": "read configuration at the CLI layer, not via os.Getenv",
        "pos": input.Node.Fun.X.NamePos,
        "sev": "WARNING",
    }
}
```

Generated files are better excluded with `goast eval --ignore-auto-generated`
(or `ignore_auto_generated = true` under `[eval]` in `.goast.toml`) than in the
policy, but a `Path`-based guard works when only some rules should skip them.

## 9. Comment / doc rules

Doc comments hang off their nodes (`FuncDecl.Doc`, `GenDecl.Doc`); a missing
doc is `null`. Free-floating comments are only reachable through the `File`
node's `Comments` list.

```rego
# Exported functions must have a doc comment.
# A missing Doc is JSON null, and null is truthy in Rego — `not input.Node.Doc`
# would never match. Compare against null explicitly.
fail contains res if {
    input.Kind == "FuncDecl"
    regex.match(`^[A-Z]`, input.Node.Name.Name)
    input.Node.Doc == null

    res := {
        "msg": sprintf("exported function %s requires a doc comment", [input.Node.Name.Name]),
        "pos": input.Node.Name.NamePos,
        "sev": "WARNING",
    }
}

# Forbid nolint-style escape hatches
fail contains res if {
    input.Kind == "File"
    some group in input.Node.Comments
    some c in group.List
    contains(c.Text, "//nolint")

    res := {
        "msg": "do not suppress linters with //nolint",
        "pos": c.Slash,
        "sev": "WARNING",
    }
}
```

## Debugging any recipe

```bash
# 1. Confirm the node shape you think you are matching
goast dump --line <N> sample.go | jq

# 2. Replay exactly that input against the policy
goast dump --line <N> sample.go | opa eval -b ./policy/ -I 'data.goast.fail'

# 3. Inspect a sub-path of the input when a condition mysteriously fails
goast dump --line <N> sample.go | opa eval -I 'input.Node.Fun'
```

An empty `fail` with no error almost always means one condition is
*undefined* (absent field), not false — check each path against the dump.
