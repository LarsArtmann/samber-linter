package healthwash

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

// TestReadmeRuleTableMatchesRegistry pins README §3 to the analyzer: every
// rule section in the README must correspond to a real rule ID in this
// package, so documentation can no longer silently drift from detection.
// HW-6 is whitelisted: it is the documented CI gate mode, not an
// analyzer-emitted diagnostic.
func TestReadmeRuleTableMatchesRegistry(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}

	registry := []string{
		RuleHW0, RuleHW1, RuleHW2, RuleHW3, RuleHW4, RuleHW5, RuleHW7,
		RuleUnresolved,
	}

	sectionRe := regexp.MustCompile(`(?m)^### (HW-[0-9A-Za-z-]+) `)
	sections := sectionRe.FindAllSubmatch(readme, -1)
	if len(sections) == 0 {
		t.Fatalf("no `### HW-N` sections found in README §3; the rule table moved?")
	}

	for _, section := range sections {
		documentedID := string(section[1])
		if documentedID == "HW-6" {
			continue
		}

		if !slices.Contains(registry, documentedID) {
			t.Errorf("README documents %s but the analyzer has no such rule; registry: %v",
				documentedID, registry)
		}
	}

	// And the README rules line must not have forgotten a shipped rule.
	for _, ruleID := range registry {
		if ruleID == RuleUnresolved {
			continue // documented separately as the --strict placeholder
		}

		ruleMention := regexp.MustCompile(`\*\*` + regexp.QuoteMeta(ruleID) + `\*\*`)
		if !ruleMention.Match(readme) {
			t.Errorf("analyzer rule %s missing from the README rules line", ruleID)
		}
	}
}
