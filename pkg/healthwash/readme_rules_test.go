package healthwash

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

// TestReadmeRuleTableMatchesRegistry pins README §3 to healthwash.RuleTable:
// every rule section in the README must correspond to a real rule in the
// single-source table, and every table entry with a slug must carry that
// exact slug in its README heading, so documentation can no longer silently
// drift from detection. HW-6 is whitelisted: it is the documented CI gate
// mode, not an analyzer-emitted diagnostic.
func TestReadmeRuleTableMatchesRegistry(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}

	registry := RuleIDs()

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

	// Every §3-heading rule must carry the table's slug verbatim — the slug
	// is part of the rule's public identity, not prose.
	for _, rule := range RuleTable {
		if rule.Slug == "" {
			continue
		}

		heading := regexp.MustCompile(
			`(?m)^### ` + regexp.QuoteMeta(rule.ID) + " `" + regexp.QuoteMeta(rule.Slug) + "`")
		if !heading.Match(readme) {
			t.Errorf("README §3 heading for %s does not carry the table slug %q", rule.ID, rule.Slug)
		}
	}
}
