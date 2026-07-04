# Running goast in CI

goast is a single Go binary that reads its settings from `.goast.toml`, so a CI
job is usually just "install goast, run `goast test`, run `goast eval`". This
page covers the exit-code rules, a full GitHub Actions example, and optional
inline PR comments via reviewdog.

- [Exit codes](#exit-codes)
- [Commit the config, run bare commands](#commit-the-config)
- [GitHub Actions](#github-actions)
- [Inline PR comments (reviewdog)](#inline-pr-comments-reviewdog)
- [Unit-testing policies with opa](#unit-testing-policies-with-opa)

## Exit codes

Two facts decide how you gate a build:

- **Without `--fail`, `goast eval` exits `0` even when it prints violations** —
  by default it only *reports* them. Pass `--fail` (or set `fail = true` under
  `[eval]`) to turn any violation into a non-zero exit. That is your gate.
- A run-time error (unreadable policy, bad path, malformed config) exits
  non-zero **regardless of `--fail`**, so a broken setup never passes silently.

Every non-zero exit is code `1`; there are no per-cause codes. `goast test`
exits non-zero whenever a case fails or the config/policy cannot be loaded, so it
gates a build on its own without any extra flag.

## Commit the config

Put the policy, targets, and the `fail` gate in `.goast.toml` once so the CI step
is a bare command (see [configuration.md](configuration.md)):

```toml
[eval]
policy = [".goast"]
paths  = ["./cmd", "./internal"]
fail   = true
```

```bash
goast eval        # policy, targets, and the fail gate all come from [eval]
```

> **goast has no exclude flag** and skips neither `testdata` nor dot-directories.
> Lint your source directories explicitly (e.g. `./cmd ./internal`) rather than
> the repo root, or goast will also walk fixtures such as intentionally-violating
> test data. List the exact directories in `[eval].paths`.

## GitHub Actions

A job that verifies the policies against their samples and then gates your
source. Pin a released tag so CI is reproducible:

```yaml
name: goast
on: [pull_request]

jobs:
  goast:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - run: go install github.com/m-mizutani/goast/cmd/goast@latest   # or a pinned tag, e.g. @v0.2.0

      # 1. Verify the policies themselves against good/bad samples.
      - run: goast test

      # 2. Lint your source; fail the job on any violation.
      #    With policy/paths/fail set in [eval], this is just `goast eval`.
      - run: goast eval
```

`goast test` catches a policy that has quietly stopped firing (a refactor of the
rule, an upgrade of Go's AST shape); `goast eval --fail` catches a new violation
in the source. Running both means neither the rules nor the code can drift
unnoticed. This is exactly how goast lints itself — see
[`.github/workflows/goast.yml`](../.github/workflows/goast.yml).

## Inline PR comments (reviewdog)

`goast eval -f json` writes the [reviewdog Diagnostic Format](https://github.com/reviewdog/reviewdog/tree/master/proto/rdf#rdjson)
(rdjson) to stdout. goast has **no runtime dependency** on reviewdog — it just
emits the byte-compatible shape — so pipe it straight in and let reviewdog own
the reporting and the pass/fail decision. Pass `--fail=false` explicitly: if
`fail = true` is committed to `[eval]`, it would otherwise make goast exit
non-zero, and GitHub Actions runs steps with `pipefail`, so the pipeline would
fail before reviewdog could report:

```yaml
      - uses: reviewdog/action-setup@v1
      - run: goast eval -f json --fail=false | reviewdog -f=rdjson -reporter=github-pr-review
        env:
          REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Each finding then appears as an inline comment on the pull request diff.

## Unit-testing policies with opa

`goast test` exercises the end-to-end pipeline. If you also write pure Rego unit
tests (functions and `test_*` rules in `*_test.rego` files), run them with the
stock `opa` command as an extra layer — goast does not run them (`eval` only
queries the `fail` rule):

```yaml
      - uses: open-policy-agent/setup-opa@v2
        with:
          version: latest
      - run: opa test -v .goast
```
