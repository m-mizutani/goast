# goast policy input format

Each policy evaluation receives exactly one AST node as `input`. This document
describes the envelope, the node kinds you can anchor on, and the JSON shapes
of the most frequently matched nodes.

The single source of truth for any shape is always a real dump:

```bash
goast dump --line <N> file.go | jq
```

## Table of contents

- [Input envelope](#input-envelope)
- [Serialization rules](#serialization-rules)
- [Node kinds (`input.Kind`)](#node-kinds-inputkind)
- [JSON shapes of common nodes](#json-shapes-of-common-nodes)
- [Comments](#comments)

## Input envelope

```json
{
  "Path": "examples/println/main.go",
  "FileName": "main.go",
  "DirName": "examples/println",
  "Kind": "ExprStmt",
  "Node": { "X": { "Fun": { ... } } }
}
```

| Field | Meaning |
|---|---|
| `Path` | File path as given to `goast eval` (useful with `glob.match` / `startswith`) |
| `FileName` | Base name, e.g. `main_test.go` — scope rules to test files with `endswith` |
| `DirName` | Directory part of `Path` |
| `Kind` | `go/ast` struct type name without `*ast.` prefix |
| `Node` | The node itself, JSON-marshalled (see below) |

## Serialization rules

- Field names are the **exported Go field names** of the `go/ast` structs
  (`Fun`, `Args`, `NamePos`, …) — no lowercasing, no json tags.
- `*ast.Object` fields (e.g. `Ident.Obj`) are always `null` — goast blanks
  them to break reference cycles (`--object-depth` can restore N levels, but
  policies should not depend on it).
- Position fields (`NamePos`, `ValuePos`, `TokPos`, `Lparen`, …) are integer
  byte offsets into the file set. Return one of them as `pos`; goast converts
  it to line/column. `0` means "no position".
- Nil pointers marshal as `null`; empty slices as `null` or `[]`.
- Interface-typed fields (`ast.Expr`, `ast.Stmt`, `ast.Decl`) marshal as the
  concrete struct **without any type discriminator**. Identify the concrete
  type by its fields: `{"Name": ...}` is an `Ident`, `{"X": ..., "Sel": ...}`
  is a `SelectorExpr`, `{"Value": ..., "Kind": ...}` is a `BasicLit`.
  Accessing a field that is absent makes the expression undefined — the rule
  body simply does not match, which is usually what you want.
- Enum-ish fields (`Op`, `Tok`, `BasicLit.Kind`, `ChanType.Dir`) are integers
  from `go/token` — see [tokens.md](tokens.md).

## Node kinds (`input.Kind`)

Every node of the file is visited, from `File` down to each `Ident`. Anchor
each rule on exactly one kind with `input.Kind == "..."`.

| Category | Kinds |
|---|---|
| Root | `File` |
| Declarations | `FuncDecl`, `GenDecl` (import/const/type/var blocks), `BadDecl` |
| Specs (inside GenDecl) | `ImportSpec`, `ValueSpec`, `TypeSpec` |
| Statements | `ExprStmt`, `AssignStmt`, `ReturnStmt`, `IfStmt`, `ForStmt`, `RangeStmt`, `SwitchStmt`, `TypeSwitchStmt`, `SelectStmt`, `GoStmt`, `DeferStmt`, `SendStmt`, `IncDecStmt`, `BranchStmt`, `LabeledStmt`, `BlockStmt`, `DeclStmt`, `CaseClause`, `CommClause`, `EmptyStmt` |
| Expressions | `Ident`, `BasicLit`, `FuncLit`, `CompositeLit`, `ParenExpr`, `SelectorExpr`, `IndexExpr`, `IndexListExpr`, `SliceExpr`, `TypeAssertExpr`, `CallExpr`, `StarExpr`, `UnaryExpr`, `BinaryExpr`, `KeyValueExpr`, `Ellipsis` |
| Type expressions | `ArrayType`, `StructType`, `FuncType`, `InterfaceType`, `MapType`, `ChanType` |
| Structure helpers | `Field`, `FieldList`, `CommentGroup`, `Comment` |

Rules of thumb for choosing the anchor:

- Call bans → `CallExpr`
- Signature / naming / body-content rules → `FuncDecl`
- Assignment rules → `AssignStmt`
- Import / package / whole-file rules → `File`
- Struct definition rules → `TypeSpec` (has the name) or `StructType`
- goroutine / defer bans → `GoStmt` / `DeferStmt`

## JSON shapes of common nodes

Shapes below list the fields that matter for policies (position fields
abbreviated). All follow `go/ast` — dump to confirm against your Go version.

### File

```json
{
  "Doc": {"List": [{"Slash": 1, "Text": "// Package doc"}]},
  "Package": 1,
  "Name": {"Name": "main", "NamePos": 9},
  "Decls": [ ...GenDecl and FuncDecl... ],
  "Imports": [ ...ImportSpec, flattened from all import decls... ],
  "Comments": [ ...every CommentGroup in the file... ]
}
```

`Imports` is the convenient flat list for import rules. `Comments` holds all
comment groups (including free-floating ones), which is the way to write
comment-content rules — free-floating comments are not visited as standalone
nodes.

### FuncDecl

```json
{
  "Doc": null,
  "Recv": {"List": [{"Names": [{"Name": "x"}], "Type": {"Star": 30, "X": {"Name": "Server"}}}]},
  "Name": {"Name": "Handle", "NamePos": 40},
  "Type": {
    "Params":  {"List": [{"Names": [{"Name": "ctx"}], "Type": {"X": {"Name": "context"}, "Sel": {"Name": "Context"}}}]},
    "Results": {"List": [{"Names": null, "Type": {"Name": "error"}}]}
  },
  "Body": {"List": [ ...statements... ]}
}
```

- `Recv` is `null` for plain functions; its presence distinguishes methods.
- A parameter's `Type` is whatever expression the source uses: `Ident`
  (`{"Name": "string"}`), `SelectorExpr` (`context.Context`), `StarExpr`
  (`{"Star": ..., "X": ...}`), etc.
- `Body` is `null` for forward declarations (e.g. assembly stubs).

### CallExpr

```json
{"Fun": ..., "Lparen": 55, "Args": [...], "Ellipsis": 0, "Rparen": 63}
```

`Fun` for `fmt.Println(...)` is a `SelectorExpr`:

```json
{"X": {"Name": "fmt", "NamePos": 44, "Obj": null}, "Sel": {"Name": "Println", "NamePos": 48}}
```

`Fun` for a local call `setup(...)` is a plain `Ident`: `{"Name": "setup"}`.
Method chains nest: `a.b().c()` has `Fun.X` being another `CallExpr`.

### ExprStmt

`{"X": <expression>}` — a bare expression used as a statement. A top-level
function call appears both as this `ExprStmt` and as the inner `CallExpr`
(separate evaluations) — anchor on one, not both.

### AssignStmt

```json
{"Lhs": [{"Name": "u"}], "TokPos": 70, "Tok": 47, "Rhs": [{"Type": {"Name": "User"}, "Elts": [...]}]}
```

`Tok` is `47` (`:=`), `42` (`=`), or an op-assign code (see tokens.md). The
`Rhs` example above is a `CompositeLit` for `User{...}`; `&User{...}` wraps it
in a `UnaryExpr` with `Op: 17`.

### GenDecl / ImportSpec / ValueSpec / TypeSpec

```json
{"Tok": 75, "Lparen": 20, "Specs": [ ... ], "Rparen": 60}
```

`Tok`: `75` import, `64` const, `84` type, `85` var.

```json
{"Name": null, "Path": {"Kind": 9, "Value": "\"unsafe\"", "ValuePos": 30}}
```

`ImportSpec.Name` is the alias `Ident` (`{"Name": "_"}` for blank imports,
`{"Name": "."}` for dot imports) or `null` when unaliased. **`Path.Value`
keeps the surrounding quotes.**

### BasicLit / Ident / SelectorExpr

```json
{"ValuePos": 56, "Kind": 9, "Value": "\"hello\""}
{"NamePos": 44, "Name": "fmt", "Obj": null}
{"X": {"Name": "fmt"}, "Sel": {"Name": "Println"}}
```

`BasicLit.Kind`: `5` INT, `6` FLOAT, `7` IMAG, `8` CHAR, `9` STRING. `Value`
is the source text: strings keep quotes, numbers keep their literal form
(`"0x1F"`). Use `trim(v, "\"")` or `json.unmarshal` sparingly when you need
the unquoted value.

### UnaryExpr / BinaryExpr / StarExpr

```json
{"OpPos": 75, "Op": 17, "X": { ... }}
{"X": {...}, "OpPos": 80, "Op": 39, "Y": {...}}
{"Star": 12, "X": {"Name": "User"}}
```

### GoStmt / DeferStmt / ReturnStmt

```json
{"Go": 100, "Call": { ...CallExpr... }}
{"Defer": 120, "Call": { ...CallExpr... }}
{"Return": 140, "Results": [ ...expressions..., null when bare ]}
```

### StructType / Field / FieldList

```json
{"Struct": 20, "Fields": {"Opening": 32, "List": [
  {"Names": [{"Name": "ID"}], "Type": {"Name": "string"}, "Tag": {"Kind": 9, "Value": "`json:\"id\"`"}}
], "Closing": 90}}
```

`Field.Tag` is a `BasicLit` (backquoted, quotes included) or `null`.
Embedded fields have `Names: null`.

## Comments

`Comment.Text` contains the raw text including `//` or `/* */`. Doc comments
hang off their node (`FuncDecl.Doc`, `GenDecl.Doc`, `Field.Doc`) as a
`CommentGroup` (`{"List": [{"Slash": ..., "Text": "// ..."}]}`). For rules
about arbitrary comments, anchor on `File` and scan `input.Node.Comments`.
