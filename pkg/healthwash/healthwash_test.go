package healthwash

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGoldenCorpus runs the analyzer over the frozen CV-incident fixtures.
// Every fixture must type-check (compile gate) and every `// want` comment
// must be matched exactly — unexpected diagnostics fail the run.
func TestGoldenCorpus(t *testing.T) {
	t.Parallel()

	analysistest.Run(t, testdataDir(), New(),
		"golden", "hw5value", "hw3transient", "hw4lazy", "hw2bare",
		"suppress", "suppressspan", "edges", "overr", "unresolvable",
	)
}

// TestStrictUnresolved verifies the --strict mode placeholder.
func TestStrictUnresolved(t *testing.T) {
	t.Parallel()

	a := New()
	if err := a.Flags.Set("strict", "true"); err != nil {
		t.Fatal(err)
	}

	analysistest.Run(t, testdataDir(), a, "unresolvedstrict")
}

// TestOrphanedDirectiveHW0 verifies that a malformed suppression directive
// with no violation to attach to still surfaces, at the comment itself. It
// lives outside TestGoldenCorpus because a `// want` comment on the
// directive line would parse as the missing reason and make it valid.
func TestOrphanedDirectiveHW0(t *testing.T) {
	t.Parallel()

	diags := runOnFixture(t, New(), "hw0orphan")

	var found bool

	for _, d := range diags {
		if d.Category == RuleHW0 {
			found = true
		}
	}

	if !found {
		t.Fatalf("orphaned malformed directive produced no HW-0: %v", diags)
	}
}

// newDisabled returns an analyzer with the given rule IDs muted — the
// "mutant analyzer" of the discrimination proofs. The fixtures themselves are
// never mutated (scratch-copy discipline).
func newDisabled(rules ...string) *analysis.Analyzer {
	a := New()
	if err := a.Flags.Set("disable", strings.Join(rules, ",")); err != nil {
		panic(err)
	}

	return a
}

// TestDiscriminationProofs demonstrates that each P0 rule's fixture actually
// fails on a mutant analyzer: with the rule disabled, the expected diagnostic
// disappears. A rule that cannot be broken by mutation is a rule whose tests
// test nothing.
func TestDiscriminationProofs(t *testing.T) {
	t.Parallel()

	// HW-0 is deliberately absent from the proof table: it is the meta-rule
	// that audits suppressions, so it is intentionally not disable-able —
	// muting the auditor would mute the audit. See the disable-flag handling
	// in healthwash.go.
	cases := []struct {
		rule   string
		pkg    string
		mutant string // the disabled rule of the mutant
	}{
		{RuleHW1, "golden", "HW-1"},
		{RuleHW5, "hw5value", "HW-5"},
		{RuleHW3, "hw3transient", "HW-3"},
		{RuleHW4, "hw4lazy", "HW-4"},
		{RuleHW2, "hw2bare", "HW-2"},
	}

	for _, testCase := range cases {
		t.Run(testCase.rule, func(t *testing.T) {
			t.Parallel()

			healthy := collectRules(t, New(), testCase.pkg)
			if !containsRule(healthy, testCase.rule) {
				t.Fatalf(
					"healthy analyzer produced no %s on %s; fixture or rule is broken",
					testCase.rule,
					testCase.pkg,
				)
			}

			mutant := collectRules(t, newDisabled(testCase.mutant), testCase.pkg)
			if containsRule(mutant, testCase.rule) {
				t.Fatalf(
					"mutant analyzer (disabled %s) still reports %s on %s; the corpus does not discriminate",
					testCase.mutant,
					testCase.rule,
					testCase.pkg,
				)
			}
		})
	}
}

// collectRules runs the analyzer over one fixture package and returns the
// set of rule codes reported. On analyzer errors it fails the test.
func collectRules(t *testing.T, a *analysis.Analyzer, pkg string) map[string]bool {
	t.Helper()
	diags := runOnFixture(t, a, pkg)

	rules := map[string]bool{}
	for _, d := range diags {
		rules[d.Category] = true
	}

	return rules
}

func containsRule(rules map[string]bool, rule string) bool { return rules[rule] }

// testdataDir returns the absolute path of the GOPATH-style fixture root.
func testdataDir() string {
	abs, err := filepath.Abs("../../testdata")
	if err != nil {
		panic(err)
	}

	return abs
}

func TestParseDirective(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		line        string
		wantOK      bool
		wantRule    string
		wantReason  string
		wantExpiry  string
		wantExpired bool
	}{
		{"not a directive", "// normal comment", false, "", "", "", false},
		{"valid hw-1", "//samber-linter:allow hw-1 inert config data", true, "hw-1", "inert config data", "", false},
		{"spaced prefix", "// samber-linter:allow hw-4 boot-critical", true, "hw-4", "boot-critical", "", false},
		{"all token", "//samber-linter:allow all accepted wholesale", true, "all", "accepted wholesale", "", false},
		{"no reason is still parsed (HW-0 material)", "//samber-linter:allow hw-1", true, "hw-1", "", "", false},
		{
			"until future",
			"//samber-linter:allow hw-2 legacy until 2027-01-01",
			true,
			"hw-2",
			"legacy",
			"2027-01-01",
			false,
		},
		{
			"until past",
			"//samber-linter:allow hw-2 legacy until 2020-01-01",
			true,
			"hw-2",
			"legacy",
			"2020-01-01",
			true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, ok := ParseDirective(testCase.line)
			if ok != testCase.wantOK {
				t.Fatalf("ok = %v, want %v", ok, testCase.wantOK)
			}

			if !ok {
				return
			}

			if parsed.Rule != testCase.wantRule {
				t.Errorf("rule = %q, want %q", parsed.Rule, testCase.wantRule)
			}

			if parsed.Reason != testCase.wantReason {
				t.Errorf("reason = %q, want %q", parsed.Reason, testCase.wantReason)
			}

			gotExpiry := ""
			if parsed.Expires != nil {
				gotExpiry = parsed.Expires.Format("2006-01-02")
			}

			if gotExpiry != testCase.wantExpiry {
				t.Errorf("expiry = %q, want %q", gotExpiry, testCase.wantExpiry)
			}

			if parsed.Expired(now) != testCase.wantExpired {
				t.Errorf("expired = %v, want %v", parsed.Expired(now), testCase.wantExpired)
			}
		})
	}
}

// TestFuzzParseDirectiveSmoke runs a deterministic corpus through the parser;
// run `go test -fuzz FuzzParseDirective` for the full fuzzing campaign. The
// parser must never panic — unexplained suppressions must not crash the tool.
func FuzzParseDirective(f *testing.F) {
	f.Add("//samber-linter:allow hw-1 reason")
	f.Add("//samber-linter:allow all")
	f.Add("//samber-linter:allow")
	f.Add("//")
	f.Add("///samber-linter:allow hw-2 until 2999-12-31")
	f.Add("//samber-linter:allow HW-9 nonexistent rule")
	f.Fuzz(func(t *testing.T, line string) {
		d, ok := ParseDirective(line)
		if ok && d.Rule == "" && d.Reason != "" {
			t.Errorf("malformed directive produced a reason without a rule: %q", line)
		}
	})
}
