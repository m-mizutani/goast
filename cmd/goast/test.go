package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

// errTestFail marks "the suite ran and one or more cases failed their
// assertions". It is deliberately distinct from load/parse/compile errors so a
// caller (and the tests) can tell an assertion failure apart from a broken test
// setup via errors.Is — the CLI turns both into a non-zero exit, but they mean
// different things.
var errTestFail = errors.New("policy test failures")

// checkCase compares the violations produced for a case against its declared
// expectations and returns a reason for every mismatch. An empty slice means the
// case passed. Every mismatch is collected (not short-circuited) so one run
// reports every way the case diverged.
func checkCase(c *testCase, fails []*goast.Fail) []string {
	var reasons []string

	switch expectFail := c.expectsFail(); {
	case expectFail && len(fails) == 0:
		reasons = append(reasons, "expected at least one violation, but got none")
	case !expectFail && len(fails) > 0:
		reasons = append(reasons, fmt.Sprintf("expected no violations, but got %d: %s", len(fails), formatFails(fails)))
	}

	// Only assert an explicit positive count here. count == 0 means "no
	// violations", which the pass/fail branch above already reports; checking it
	// again would just duplicate that reason for the same case.
	if c.Count != nil && *c.Count > 0 && len(fails) != *c.Count {
		reasons = append(reasons, fmt.Sprintf("expected exactly %d violation(s), but got %d: %s", *c.Count, len(fails), formatFails(fails)))
	}

	for _, want := range c.Violations {
		if !matchesAnyViolation(want, fails) {
			reasons = append(reasons, fmt.Sprintf("expected a violation matching %s, but got: %s", want.describe(), formatFails(fails)))
		}
	}

	return reasons
}

// matchesAnyViolation reports whether any reported violation satisfies every
// field set on want. Unset fields are wildcards, so an entry with only line set
// matches on line alone.
func matchesAnyViolation(want *expectedViolation, fails []*goast.Fail) bool {
	for _, f := range fails {
		if want.Line != nil && f.Line != *want.Line {
			continue
		}
		if want.Column != nil && f.Column != *want.Column {
			continue
		}
		if want.Message != "" && !strings.Contains(f.Msg, want.Message) {
			continue
		}
		if want.Severity != "" && !strings.EqualFold(f.Sev, want.Severity) {
			continue
		}
		return true
	}
	return false
}

func (v *expectedViolation) describe() string {
	var parts []string
	if v.Line != nil {
		parts = append(parts, fmt.Sprintf("line=%d", *v.Line))
	}
	if v.Column != nil {
		parts = append(parts, fmt.Sprintf("column=%d", *v.Column))
	}
	if v.Message != "" {
		parts = append(parts, fmt.Sprintf("message~=%q", v.Message))
	}
	if v.Severity != "" {
		parts = append(parts, fmt.Sprintf("severity=%q", v.Severity))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatFails(fails []*goast.Fail) string {
	if len(fails) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(fails))
	for _, f := range fails {
		parts = append(parts, fmt.Sprintf("[%s:%d:%d] %q", f.Path, f.Line, f.Column, f.Msg))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// engineCache compiles each distinct policy set once and reuses it across cases.
// A goast policy suite typically points many cases at the same .goast directory,
// so compiling per case would repeat the (relatively expensive) Rego compile for
// no reason.
type engineCache struct {
	engines map[string]*goast.PolicyEngine
}

func newEngineCache() *engineCache {
	return &engineCache{engines: map[string]*goast.PolicyEngine{}}
}

func (ec *engineCache) get(paths []string) (*goast.PolicyEngine, error) {
	// Key on the sorted paths: NewPolicyEngine dedups by absolute path and is
	// order-independent, so two cases naming the same policies in a different
	// order should share one compiled engine.
	key := make([]string, len(paths))
	copy(key, paths)
	sort.Strings(key)
	cacheKey := strings.Join(key, "\x00")

	if e, ok := ec.engines[cacheKey]; ok {
		return e, nil
	}
	e, err := goast.NewPolicyEngine(paths...)
	if err != nil {
		return nil, err
	}
	ec.engines[cacheKey] = e
	return e, nil
}

// runCase compiles the case's policy, collects its source .go files, and
// evaluates them, returning the findings in deterministic walk order.
func runCase(ec *engineCache, section testSection, c *testCase) ([]*goast.Fail, error) {
	paths := c.policyPaths(section)
	engine, err := ec.get(paths)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load policy", goerr.V("policy", paths))
	}

	opts := []goast.Option{goast.WithPolicyEngine(engine)}
	if depth := c.objectDepth(section); depth > 0 {
		opts = append(opts, goast.WithInspectOptions(goast.WithObjectDepth(depth)))
	}
	if c.ignoreAutoGen(section) {
		opts = append(opts, goast.WithIgnoreAutoGen())
	}
	g := goast.New(opts...)

	files, err := listGoFiles(c.Source)
	if err != nil {
		return nil, err
	}

	fails, err := evalFilesParallel(files, 0, g.Eval)
	if err != nil {
		return nil, err
	}
	return fails, nil
}

type caseOutcome struct {
	name       string
	violations int
	reasons    []string // empty means the case passed
}

// runTests runs every case in order and writes a per-case report plus a summary
// to w. A broken setup (unreadable/uncompilable policy, unparseable fixture) is
// returned as an error and aborts the run; assertion failures do not abort but
// make the final return errTestFail.
func runTests(section testSection, w io.Writer) error {
	ec := newEngineCache()
	outcomes := make([]caseOutcome, 0, len(section.Cases))

	for i, c := range section.Cases {
		name := c.displayName(i)
		fails, err := runCase(ec, section, c)
		if err != nil {
			return goerr.Wrap(err, "failed to run test case", goerr.V("case", name))
		}
		outcomes = append(outcomes, caseOutcome{
			name:       name,
			violations: len(fails),
			reasons:    checkCase(c, fails),
		})
	}

	return reportOutcomes(w, outcomes)
}

func reportOutcomes(w io.Writer, outcomes []caseOutcome) error {
	passed := 0
	for _, o := range outcomes {
		if len(o.reasons) == 0 {
			passed++
			printStatus(w, true, o.name)
			fmt.Fprintf(w, "  (%s)\n", pluralViolations(o.violations))
			continue
		}
		printStatus(w, false, o.name)
		fmt.Fprintln(w)
		for _, r := range o.reasons {
			fmt.Fprintf(w, "        %s\n", r)
		}
	}

	failed := len(outcomes) - passed
	fmt.Fprintf(w, "\n%d cases: %d passed, %d failed\n", len(outcomes), passed, failed)

	if failed > 0 {
		return goerr.Wrap(errTestFail, "policy tests failed", goerr.V("failed", failed), goerr.V("total", len(outcomes)))
	}
	return nil
}

func pluralViolations(n int) string {
	if n == 1 {
		return "1 violation"
	}
	return fmt.Sprintf("%d violations", n)
}

// printStatus writes the "ok"/"FAIL" label, colored only when writing to a real
// terminal (stdout) so piped/captured output stays plain — the same rule the
// eval text output follows.
func printStatus(w io.Writer, ok bool, name string) {
	label := "ok  "
	col := color.New(color.FgGreen)
	if !ok {
		label = "FAIL"
		col = color.New(color.FgRed)
	}

	// Colorize only for a real terminal (stdout); Sprint returns a plain string
	// so there is no write error to drop, unlike the color Fprint helpers.
	if w == os.Stdout {
		label = col.Sprint(label)
	}
	fmt.Fprintf(w, "%s  %s", label, name)
}

func cmdTest() *cli.Command {
	var config string

	return &cli.Command{
		Name:    "test",
		Usage:   "run declarative policy tests defined in a TOML config",
		Aliases: []string{"t"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				// Generic name (not test-specific) so other subcommands can adopt
				// the same env var later without a breaking rename.
				Sources:     cli.EnvVars("GOAST_CONFIG"),
				Destination: &config,
				Value:       ".goast.toml",
				Usage:       "Path to the config TOML file",
			},
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg, err := loadConfig(config)
			if err != nil {
				return err
			}
			if err := cfg.Test.validate(); err != nil {
				return goerr.Wrap(err, "invalid test config", goerr.V("path", config))
			}
			return runTests(cfg.Test, os.Stdout)
		},
	}
}
