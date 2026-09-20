package driver

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/samber-linter/pkg/healthwash"
)

// TestRuleMetaCoversEveryRule: the driver's severity/confidence table must
// have an explicit entry for every analyzer rule. A missing entry silently
// falls back to warning/Medium, which would quietly de-gate a Full-confidence
// rule.
func TestRuleMetaCoversEveryRule(t *testing.T) {
	t.Parallel()

	allRules := []string{
		healthwash.RuleHW0,
		healthwash.RuleHW1,
		healthwash.RuleHW2,
		healthwash.RuleHW3,
		healthwash.RuleHW4,
		healthwash.RuleHW5,
		healthwash.RuleHW7,
		healthwash.RuleUnresolved,
	}

	for _, ruleID := range allRules {
		if _, ok := ruleMetaByRule[ruleID]; !ok {
			t.Errorf("ruleMetaByRule has no entry for %s (falls back to warning/Medium)", ruleID)
		}
	}

	if len(ruleMetaByRule) != len(allRules) {
		t.Errorf("ruleMetaByRule has %d entries, want %d (stale entry for a retired rule?)",
			len(ruleMetaByRule), len(allRules))
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
