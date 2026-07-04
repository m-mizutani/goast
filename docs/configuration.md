# Configuration (`.goast.toml`)

`goast eval` and `goast test` both read their settings from a `.goast.toml`
file so the policy, the target paths, and the output options can be committed
once and re-run with a bare command in CI.

A single file may declare `[eval]`, `[test]`, or both. Each subcommand
validates and consumes only its own section, so an `eval` run does not require a
`[test]` table and vice versa.

- [File lookup](#file-lookup)
- [`[eval]` section](#eval-section)
- [Precedence: CLI > config > default](#precedence)
- [`[test]` section](#test-section)
- [Path resolution](#path-resolution)

## File lookup

Both subcommands default to `./.goast.toml` (relative to the current working
directory). Point at another file with `--config, -c` or the `GOAST_CONFIG`
environment variable:

```bash
goast eval --config config/goast.toml
GOAST_CONFIG=config/goast.toml goast eval
```

The default file is optional, but an explicitly named one is not:

| Situation | Behaviour |
|---|---|
| Default `.goast.toml` is absent | `eval` runs from CLI flags alone; `test` errors (it has nothing to run) |
| `--config X` given and `X` is absent | Error — your explicit intent is never silently ignored |
| File present but malformed | Error (parse failure) |

## `[eval]` section

Every key is optional. An omitted key falls back to the matching CLI flag's
built-in default. Path-valued keys (`policy`, `paths`) are always TOML arrays —
write a single value as a one-element array.

```toml
[eval]
policy = ["policy"]                # policy file(s) or dir(s), always an array
paths  = ["./cmd", "./internal"]   # evaluation targets used when no arguments are given
fail   = true                      # non-zero exit on any violation (the CI gate)
# format = "json"                  # "text" (default) or "json"
# output = "goast.rdjson"          # omit for stdout (there is no "-" sentinel in config)
# ignore_auto_generated = true     # skip files with a generated-code header
# workers = 8                      # files evaluated in parallel (0 = one per CPU)
# object_depth = 0                 # levels of *ast.Object to restore (0 = none)
```

| Key | Type | Default | Meaning |
|---|---|---|---|
| `policy` | `[]string` | — (required for a run) | Policy files or directories; directories are walked for `*.rego`. May also be supplied by `--policy`. |
| `paths` | `[]string` | — | Evaluation targets used **only when no positional arguments** are given. |
| `format` | `string` | `"text"` | Output format: `"text"` or `"json"` (rdjson). |
| `output` | `string` | stdout | Output file path. Omit (or empty) for stdout — unlike the CLI, there is no `"-"` sentinel here. |
| `fail` | `bool` | `false` | Exit non-zero when any violation is reported. |
| `ignore_auto_generated` | `bool` | `false` | Skip auto-generated Go files. |
| `workers` | `int` | `0` | Number of files evaluated concurrently; `0` uses one worker per CPU, `1` forces sequential. Must be `>= 0`. |
| `object_depth` | `int` | `0` | Levels of the `*ast.Object` graph to restore in the input. `0` keeps it stripped. Must be `>= 0`. |

A run needs **both** a policy and at least one target. If neither `--policy`
nor `[eval].policy` is set, or neither positional arguments nor `[eval].paths`
resolve to a target, `eval` reports the missing piece and exits non-zero.

## Precedence

Effective settings are resolved in this order, highest first:

```
CLI flag (explicitly set)  >  [eval] value  >  built-in default
```

- An explicitly-set CLI flag always wins. A flag left at its default does **not**
  override a config value — the config fills in.
- **Targets are special**: positional arguments are the primary way to name
  targets. `[eval].paths` is used only when no positional arguments are given.
- The `output` `"-"` sentinel (stdout) exists on the CLI only. In config, use an
  empty/omitted `output` for stdout and a plain path otherwise.

```bash
goast eval                     # policy + targets both come from [eval]
goast eval -f json ./cmd       # -f json and ./cmd override the config
```

## `[test]` section

`[test]` drives `goast test` (see [usage.md](usage.md#goast-test)). It carries
defaults shared by every case plus the list of cases.

```toml
[test]
policy = [".goast"]            # default policy for every case; an array of files/dirs
# object_depth = 0            # default, overridable per case
# ignore_auto_generated = false

[[test.cases]]
name   = "forbids fmt.Println"
source = [".goast/testdata/no_println_bad.go"]   # array of Go files/dirs
expect = "fail"               # "pass" (0 violations) or "fail" (>0 violations)
count  = 1                    # optional: assert the exact number of violations

  [[test.cases.violations]]   # optional: each entry must match some finding
  line     = 6                # every field is independently optional...
  message  = "do not use fmt.Println"   # ...message is a substring match
  severity = "ERROR"          # ...severity is case-insensitive
```

**`[test]` defaults**

| Key | Type | Default | Meaning |
|---|---|---|---|
| `policy` | `[]string` | — | Default policy applied to every case that does not set its own. |
| `object_depth` | `int` | `0` | Default `object_depth` for every case. |
| `ignore_auto_generated` | `bool` | `false` | Default for every case. |

**`[[test.cases]]` — one case per entry**

| Key | Type | Required | Meaning |
|---|---|---|---|
| `name` | `string` | no | Label shown in the report; defaults to the first source path. |
| `policy` | `[]string` | no | Overrides `[test].policy` for this case. |
| `source` | `[]string` | **yes** | Go files/directories to evaluate. |
| `expect` | `string` | see note | `"pass"` (0 violations) or `"fail"` (>0). |
| `count` | `int` | no | Assert the exact number of violations (`>= 0`). |
| `object_depth` | `int` | no | Overrides the section default. |
| `ignore_auto_generated` | `bool` | no | Overrides the section default. |
| `violations` | `[[...]]` | no | Fine-grained assertions (below). |

**`[[test.cases.violations]]` — assert a specific finding**

Each entry must match **at least one** reported violation. Every field is
independently optional (an unset field is a wildcard), but an entry with no
fields at all is rejected.

| Field | Type | Match |
|---|---|---|
| `line` | `int` | exact line |
| `column` | `int` | exact column |
| `message` | `string` | **substring** of the finding message |
| `severity` | `string` | **case-insensitive** equality (`INFO`/`WARNING`/`ERROR`) |

**Assertions and their consistency.** A case needs at least one of `expect`,
`count`, or `violations` — otherwise it asserts nothing and is rejected at load
time. Contradictory combinations are also rejected up front rather than failing
confusingly at run time:

- `expect = "pass"` with `count > 0` or any declared `violations`
- `expect = "fail"` with `count = 0`
- `count = 0` with declared `violations`

When `expect` is omitted, the expectation is derived from `count` / `violations`
(any positive count or any declared violation implies "fail").

## Path resolution

`policy`, `paths` (eval), and `source` (test) are resolved relative to the
**current working directory** — not the config file's location — matching how
`goast eval` takes paths on the command line. Keep `.goast.toml` at the
repository root and run goast from there, which is the intended CI usage.
