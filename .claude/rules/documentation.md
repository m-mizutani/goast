# Documentation

When a change alters observable behavior — a new/changed CLI flag or subcommand,
a new/changed `.goast.toml` config key, a change to the Rego input shape or the
`goast` package API, or any breaking change to a config schema — update **every**
documentation surface in the repo in the same change, not just one.

## Documentation surfaces in this repo

This project has no `docs/` directory. Its documentation lives in several places,
and it is easy to update one and forget the rest. Before declaring a change done,
check all of them:

- **`README.md`** — the CLI reference (`goast eval` / `test` / `dump` / `sync`),
  config examples, and output-format notes.
- **The bundled skill under `skills/goast-policy/`** — this ships as a Claude Code
  skill and is first-class documentation, NOT an afterthought:
  - `skills/goast-policy/SKILL.md` — CLI usage, the `.goast.toml` schema, CI wiring.
  - `skills/goast-policy/references/*.md` — `input-format.md`, `patterns.md`,
    `tokens.md`.
  - A change to `eval`/`test` behavior or the config schema almost always needs a
    matching edit here. Do not stop at README.
- **Inline CLI help** — the `Usage` strings and flag descriptions in
  `cmd/goast/*.go` are user-facing text; keep them in sync with the flags.
- **Example config** — the repo-root `.goast.toml` is a live example; keep it
  valid against the current schema.

## How to catch every surface

`grep -rn` the changed name (flag, config key, subcommand, function) across
`README.md`, `skills/`, and `cmd/` to find every place that references it. A
breaking change to a config schema (e.g. a field's accepted TOML shape) means
**every example of the old form must be migrated in the same change** —
`README.md`, `SKILL.md`, `references/`, `testdata/**/*.toml`, and the repo-root
`.goast.toml` included.

Treat documentation updates as part of the implementation and list them in the
spec / implementation plan from the start.
