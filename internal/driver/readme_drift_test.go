package driver

import (
	"context"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/samber-linter/pkg/healthwash"
	"golang.org/x/tools/go/analysis"
)

// TestToFindingHonorsRuleTable: the driver's severity/confidence mapping is
// consumed from healthwash.RuleTable (the single source). This test pins the
// wiring: every table entry's severity and confidence must reach the finding
// unchanged — a regression to a hand-maintained map or a broken lookup would
// silently re-tier a rule's exit code.
func TestToFindingHonorsRuleTable(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "healthwash"}
	fset := token.NewFileSet()
	file := fset.AddFile("fixture.go", -1, 100)

	for _, rule := range healthwash.RuleTable {
		fnd := toFinding(analyzer, analysis.Diagnostic{
			Pos:      file.Pos(10),
			Category: rule.ID,
			Message:  rule.ID + ": fixture",
		}, fset)

		if fnd.Severity != rule.Severity {
			t.Errorf("%s: severity = %q, want %q", rule.ID, fnd.Severity, rule.Severity)
		}

		if fnd.Confidence != rule.Confidence {
			t.Errorf("%s: confidence = %v, want %v", rule.ID, fnd.Confidence, rule.Confidence)
		}
	}
}

// TestReadmeLatestReleaseMatchesGitTag: README §12 claims a latest tagged
// release; that claim must match the repository's highest version tag, or the
// README is lying about what consumers can pin. Skips where git is
// unavailable (hermetic nix sandboxes); runs for real in CI.
func TestReadmeLatestReleaseMatchesGitTag(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}

	lineRe := regexp.MustCompile(`Latest tagged release: \*\*(v[0-9]+\.[0-9]+\.[0-9]+)\*\*`)
	match := lineRe.FindSubmatch(readme)

	if match == nil {
		t.Fatalf("README §12 lost its `Latest tagged release: **vX.Y.Z**` line")
	}

	claimed := string(match[1])

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	tagOut, err := exec.CommandContext(context.Background(), gitPath, "tag", "--sort=-v:refname").Output()
	if err != nil {
		t.Skipf("git tag failed: %v", err)
	}

	var tags []string

	for line := range strings.SplitSeq(strings.TrimSpace(string(tagOut)), "\n") {
		if line != "" {
			tags = append(tags, line)
		}
	}

	if len(tags) == 0 {
		t.Skip("no tags in this checkout")
	}

	// git --sort=-v:refname already returns version order; index 0 is highest.
	highest := tags[0]
	if claimed != highest {
		t.Errorf("README claims latest release %s but the highest tag is %s; "+
			"bump the README before tagging (release flow)", claimed, highest)
	}
}
