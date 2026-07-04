# go/token integer values in goast input

goast marshals `go/ast` nodes with plain `encoding/json`, so every
`go/token`-typed field arrives as an integer. This table maps the integers
back to tokens (values verified against `go/token/token.go`; they are stable
across Go releases because new tokens are only appended).

Where these integers appear:

| Input field | Type | Typical values |
|---|---|---|
| `BasicLit.Kind` | literal kind | 5 INT, 6 FLOAT, 7 IMAG, 8 CHAR, 9 STRING |
| `UnaryExpr.Op` / `BinaryExpr.Op` | operator | 12–48 (see below) |
| `AssignStmt.Tok` | assignment | 42 `=`, 47 `:=`, 23–33 op-assign |
| `IncDecStmt.Tok` | inc/dec | 37 `++`, 38 `--` |
| `GenDecl.Tok` | decl keyword | 75 `import`, 64 `const`, 84 `type`, 85 `var` |
| `BranchStmt.Tok` | branch keyword | 61 `break`, 65 `continue`, 73 `goto`, 69 `fallthrough` |
| `RangeStmt.Tok` | range assign | 42 `=`, 47 `:=`, 0 (bare `for range x`) |
| `ChanType.Dir` | channel direction (`go/ast`, not token) | 1 SEND (`chan<-`), 2 RECV (`<-chan`), 3 both |

Always leave a comment naming the token next to the magic number:

```rego
rhs.Op == 17    # token.AND: &X{} (address-of composite literal)
```

## Literals

| Value | Token | Source text |
|---|---|---|
| 4 | IDENT | `main` |
| 5 | INT | `12345` |
| 6 | FLOAT | `123.45` |
| 7 | IMAG | `123.45i` |
| 8 | CHAR | `'a'` |
| 9 | STRING | `"abc"` |

## Operators and delimiters

| Value | Token | Source | Value | Token | Source |
|---|---|---|---|---|---|
| 12 | ADD | `+` | 31 | SHL_ASSIGN | `<<=` |
| 13 | SUB | `-` | 32 | SHR_ASSIGN | `>>=` |
| 14 | MUL | `*` | 33 | AND_NOT_ASSIGN | `&^=` |
| 15 | QUO | `/` | 34 | LAND | `&&` |
| 16 | REM | `%` | 35 | LOR | `\|\|` |
| 17 | AND | `&` | 36 | ARROW | `<-` |
| 18 | OR | `\|` | 37 | INC | `++` |
| 19 | XOR | `^` | 38 | DEC | `--` |
| 20 | SHL | `<<` | 39 | EQL | `==` |
| 21 | SHR | `>>` | 40 | LSS | `<` |
| 22 | AND_NOT | `&^` | 41 | GTR | `>` |
| 23 | ADD_ASSIGN | `+=` | 42 | ASSIGN | `=` |
| 24 | SUB_ASSIGN | `-=` | 43 | NOT | `!` |
| 25 | MUL_ASSIGN | `*=` | 44 | NEQ | `!=` |
| 26 | QUO_ASSIGN | `/=` | 45 | LEQ | `<=` |
| 27 | REM_ASSIGN | `%=` | 46 | GEQ | `>=` |
| 28 | AND_ASSIGN | `&=` | 47 | DEFINE | `:=` |
| 29 | OR_ASSIGN | `\|=` | 48 | ELLIPSIS | `...` |
| 30 | XOR_ASSIGN | `^=` | | | |

Note: unary `&X{}` and `*p` reuse the binary operator codes — `UnaryExpr.Op`
is `17` (AND) for address-of and `StarExpr` (no `Op` at all) for
dereference/pointer types. Unary minus is `13` (SUB), logical not `43` (NOT),
channel receive `36` (ARROW).

Delimiters (49–58: `(`, `[`, `{`, `,`, `.`, `)`, `]`, `}`, `;`, `:`) never
appear as field values — their positions do (`Lparen`, `Rbrace`, …).

## Keywords

| Value | Token | Value | Token | Value | Token |
|---|---|---|---|---|---|
| 61 | BREAK | 70 | FOR | 79 | RANGE |
| 62 | CASE | 71 | FUNC | 80 | RETURN |
| 63 | CHAN | 72 | GO | 81 | SELECT |
| 64 | CONST | 73 | GOTO | 82 | STRUCT |
| 65 | CONTINUE | 74 | IF | 83 | SWITCH |
| 66 | DEFAULT | 75 | IMPORT | 84 | TYPE |
| 67 | DEFER | 76 | INTERFACE | 85 | VAR |
| 68 | ELSE | 77 | MAP | 88 | TILDE (`~`) |
| 69 | FALLTHROUGH | 78 | PACKAGE | | |

(Values 3, 10, 11, 59, 60, 86, 87 are internal boundary markers and never
appear in input.)
