package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/m-mizutani/goast"
	"github.com/m-mizutani/gt"
)

func ip(n int) *int { return &n }

func mkFail(line, col int, msg, sev string) *goast.Fail {
	return &goast.Fail{Path: "sample.go", Line: line, Column: col, Msg: msg, Sev: sev}
}

// TestCheckCase is the core of the command: it exhaustively pins how declared
// expectations (expect / count / violations, each optional) map to pass or a set
// of failure reasons over a given list of reported violations.
func TestCheckCase(t *testing.T) {
	type testCaseSpec struct {
		name         string
		c            *testCase
		fails        []*goast.Fail
		wantPass     bool
		wantSubstr   string // if set, the joined reasons must contain it
		wantNReasons int    // if > 0, the exact number of reasons expected
	}

	oneViolation := []*goast.Fail{mkFail(6, 2, "do not use fmt.Println", "ERROR")}
	twoViolations := []*goast.Fail{
		mkFail(6, 2, "do not use fmt.Println", "ERROR"),
		mkFail(8, 2, "do not use fmt.Println", "ERROR"),
	}

	specs := []testCaseSpec{
		// --- expect only ---
		{name: "expect pass, no violations", c: &testCase{Expect: "pass"}, fails: nil, wantPass: true},
		{name: "expect pass, one violation", c: &testCase{Expect: "pass"}, fails: oneViolation, wantPass: false, wantSubstr: "expected no violations"},
		{name: "expect fail, one violation", c: &testCase{Expect: "fail"}, fails: oneViolation, wantPass: true},
		{name: "expect fail, no violations", c: &testCase{Expect: "fail"}, fails: nil, wantPass: false, wantSubstr: "expected at least one violation"},
		{name: "expect is case-insensitive", c: &testCase{Expect: "FAIL"}, fails: oneViolation, wantPass: true},

		// --- expect derived from count / violations ---
		{name: "derived fail from count>0", c: &testCase{Count: ip(2)}, fails: twoViolations, wantPass: true},
		{name: "derived fail from violations", c: &testCase{Violations: []*expectedViolation{{Line: ip(6)}}}, fails: oneViolation, wantPass: true},
		{name: "derived pass from count==0", c: &testCase{Count: ip(0)}, fails: nil, wantPass: true},
		// count==0 must not duplicate the "no violations" reason with an "expected
		// exactly 0" one; the pass/fail branch already covers it.
		{name: "derived pass from count==0 but a violation exists", c: &testCase{Count: ip(0)}, fails: oneViolation, wantPass: false, wantNReasons: 1, wantSubstr: "expected no violations"},

		// --- count ---
		{name: "count exact match", c: &testCase{Expect: "fail", Count: ip(2)}, fails: twoViolations, wantPass: true},
		{name: "count too many", c: &testCase{Expect: "fail", Count: ip(1)}, fails: twoViolations, wantPass: false, wantSubstr: "expected exactly 1"},
		{name: "count too few", c: &testCase{Expect: "fail", Count: ip(3)}, fails: twoViolations, wantPass: false, wantSubstr: "expected exactly 3"},

		// --- violations, single field ---
		{name: "line only match", c: &testCase{Violations: []*expectedViolation{{Line: ip(6)}}}, fails: oneViolation, wantPass: true},
		{name: "line only mismatch", c: &testCase{Violations: []*expectedViolation{{Line: ip(7)}}}, fails: oneViolation, wantPass: false, wantSubstr: "line=7"},
		{name: "column only match", c: &testCase{Violations: []*expectedViolation{{Column: ip(2)}}}, fails: oneViolation, wantPass: true},
		{name: "column only mismatch", c: &testCase{Violations: []*expectedViolation{{Column: ip(9)}}}, fails: oneViolation, wantPass: false},
		{name: "message substring match", c: &testCase{Violations: []*expectedViolation{{Message: "Println"}}}, fails: oneViolation, wantPass: true},
		{name: "message not contained", c: &testCase{Violations: []*expectedViolation{{Message: "Printf"}}}, fails: oneViolation, wantPass: false},
		{name: "severity exact match", c: &testCase{Violations: []*expectedViolation{{Severity: "ERROR"}}}, fails: oneViolation, wantPass: true},
		{name: "severity case-insensitive match", c: &testCase{Violations: []*expectedViolation{{Severity: "error"}}}, fails: oneViolation, wantPass: true},
		{name: "severity mismatch", c: &testCase{Violations: []*expectedViolation{{Severity: "WARNING"}}}, fails: oneViolation, wantPass: false},

		// --- violations, combined fields (AND semantics) ---
		{name: "all fields match", c: &testCase{Violations: []*expectedViolation{{Line: ip(6), Message: "Println", Severity: "ERROR"}}}, fails: oneViolation, wantPass: true},
		{name: "line matches but message differs", c: &testCase{Violations: []*expectedViolation{{Line: ip(6), Message: "Printf"}}}, fails: oneViolation, wantPass: false},

		// --- order independence & subset ---
		{name: "subset: more actual than listed", c: &testCase{Violations: []*expectedViolation{{Line: ip(6)}}}, fails: twoViolations, wantPass: true},
		{name: "order independent", c: &testCase{Violations: []*expectedViolation{{Line: ip(8)}, {Line: ip(6)}}}, fails: twoViolations, wantPass: true},
		{name: "one listed violation missing", c: &testCase{Violations: []*expectedViolation{{Line: ip(6)}, {Line: ip(99)}}}, fails: twoViolations, wantPass: false, wantNReasons: 1, wantSubstr: "line=99"},

		// --- count + violations together ---
		{name: "count and violations both satisfied", c: &testCase{Count: ip(2), Violations: []*expectedViolation{{Line: ip(6)}, {Line: ip(8)}}}, fails: twoViolations, wantPass: true},
		{name: "count fails, violation ok", c: &testCase{Count: ip(3), Violations: []*expectedViolation{{Line: ip(6)}}}, fails: twoViolations, wantPass: false, wantNReasons: 1, wantSubstr: "expected exactly 3"},
		{name: "violation fails, count ok", c: &testCase{Count: ip(2), Violations: []*expectedViolation{{Line: ip(99)}}}, fails: twoViolations, wantPass: false, wantNReasons: 1, wantSubstr: "line=99"},

		// --- multiple reasons are aggregated, not short-circuited ---
		{name: "count and violation both fail", c: &testCase{Expect: "fail", Count: ip(1), Violations: []*expectedViolation{{Line: ip(99)}}}, fails: twoViolations, wantPass: false, wantNReasons: 2},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			reasons := checkCase(s.c, s.fails)
			joined := strings.Join(reasons, " | ")

			if s.wantPass {
				gt.A(t, reasons).Length(0)
				return
			}

			gt.A(t, reasons).Longer(0)
			if s.wantNReasons > 0 {
				gt.A(t, reasons).Length(s.wantNReasons)
			}
			if s.wantSubstr != "" {
				gt.V(t, strings.Contains(joined, s.wantSubstr)).Equal(true)
			}
		})
	}
}

// TestRunTestsPassSuite exercises the whole pipeline (config -> policy compile ->
// eval real Go -> assertions) over a suite whose every case is correct.
func TestRunTestsPassSuite(t *testing.T) {
	cfg := gt.R1(loadConfig("testdata/test/pass_suite.toml")).NoError(t)

	var buf bytes.Buffer
	gt.NoError(t, runTests(cfg.Test, &buf))

	out := buf.String()
	gt.V(t, strings.Contains(out, "3 cases: 3 passed, 0 failed")).Equal(true)
}

// TestRunTestsFailSuite proves a suite with a wrong expectation reports the
// failure and surfaces errTestFail (distinct from a setup error) without
// aborting the remaining cases.
func TestRunTestsFailSuite(t *testing.T) {
	cfg := gt.R1(loadConfig("testdata/test/fail_suite.toml")).NoError(t)

	var buf bytes.Buffer
	err := runTests(cfg.Test, &buf)

	gt.NotNil(t, err)
	gt.V(t, errors.Is(err, errTestFail)).Equal(true)
	gt.V(t, strings.Contains(buf.String(), "2 cases: 1 passed, 1 failed")).Equal(true)
}

// TestEngineCacheReuse pins the non-obvious cache behavior: the same set of
// policy paths — even given in a different order — must share one compiled
// engine, while a different set must not.
func TestEngineCacheReuse(t *testing.T) {
	ec := newEngineCache()
	a := "testdata/test/no_println.rego"
	b := "testdata/test/no_print.rego"

	e1 := gt.R1(ec.get([]string{a, b})).NoError(t)
	// Same set, reversed order: the sorted key must resolve to the same engine.
	e2 := gt.R1(ec.get([]string{b, a})).NoError(t)
	gt.V(t, e1 == e2).Equal(true)

	// A different set must compile (and cache) a distinct engine.
	e3 := gt.R1(ec.get([]string{a})).NoError(t)
	gt.V(t, e1 == e3).Equal(false)
}

// TestRunTestsRepoRootConfig runs the repository's own .goast.toml end-to-end
// from the repo root, so the shipped dogfooding config (and the example policy it
// points at) stays green. It also covers the default config path the CLI uses.
func TestRunTestsRepoRootConfig(t *testing.T) {
	t.Chdir("../..") // move from cmd/goast to the repo root where .goast.toml lives

	cfg := gt.R1(loadConfig(".goast.toml")).NoError(t)
	gt.NoError(t, cfg.Test.validate())

	var buf bytes.Buffer
	gt.NoError(t, runTests(cfg.Test, &buf))
	gt.V(t, strings.Contains(buf.String(), "5 cases: 5 passed, 0 failed")).Equal(true)
}

// TestRunTestsPolicyError makes sure a broken policy is a hard error (aborting
// the run), not a silently-failed case.
func TestRunTestsPolicyError(t *testing.T) {
	cfg := gt.R1(loadConfig(writeConfig(t, `
[test]
policy = ["testdata/test/does_not_exist.rego"]
[[test.cases]]
source = ["testdata/test/no_println_bad.go"]
expect = "fail"
`))).NoError(t)

	var buf bytes.Buffer
	err := runTests(cfg.Test, &buf)

	gt.NotNil(t, err)
	// A missing/uncompilable policy is a setup error, not an assertion failure.
	gt.V(t, errors.Is(err, errTestFail)).Equal(false)
}
