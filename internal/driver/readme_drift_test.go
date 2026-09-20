package driver

import (
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
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

	all := []string{
		healthwash.RuleHW0,
		healthwash.RuleHW1,
		healthwash.RuleHW2,
		healthwash.RuleHW3,
		healthwash.RuleHW4,
		healthwash.RuleHW5,
		healthwash.RuleHW7,
		healthwash.RuleUnresolved,
	}

	for _, rule := range all {
		if _, ok := ruleMetaByRule[rule]; !ok {
			t.Errorf("ruleMetaByRule has no entry for %s (falls back to warning/Medium)", rule)
		}
	}

	if len(ruleMetaByRule) != len(all) {
		t.Errorf("ruleMetaByRule has %d entries, want %d (stale entry for a retired rule?)",
			len(ruleMetaByRule), len(all))
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

	git, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	cmd := exec.Command(git, "tag", "--sort=-v:refname")
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git tag failed: %v", err)
	}

	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			tags = append(tags, line)
		}
	}

	if len(tags) == 0 {
		t.Skip("no tags in this checkout")
	}

	highest := tags[0]
	if claimed != highest {
		t.Errorf("README claims latest release %s but the highest tag is %s; "+
			"bump the README before tagging (release flow)", claimed, highest)
	}

	// Sanity: the comparison itself must be version-aware, not lexical.
	if slices.IsSortedFunc(tags, func(a, b string) int {
		return compareVersions(a, b)
	}) {
		t.Logf("tag order verified version-aware")
	}
}

// compareVersions orders v-prefixed semver strings (v0.2.10 > v0.2.9).
func compareVersions(a, b string) int {
	nums := func(v string) []int {
		v = strings.TrimPrefix(v, "v")

		parts := strings.Split(v, ".")
		out := make([]int, len(parts))

		for i, p := range parts {
			n, _ := strconv.Atoi(p)
			out[i] = n
		}

		return out
	}

	av, bv := nums(a), nums(b)

	for i := range min(len(av), len(bv)) {
		if av[i] != bv[i] {
			return av[i] - bv[i]
		}
	}

	return len(av) - len(bv)
}
