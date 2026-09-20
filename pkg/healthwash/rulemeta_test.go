package healthwash

import (
	"slices"
	"testing"
)

// TestRuleTableCoversEveryRuleConstant is the single-source guard: every rule
// ID constant the analyzer emits must carry a RuleTable entry, and the table
// must not carry anything else. Adding a Rule constant without a descriptor
// (or retiring one without removing its entry) fails here — that is exactly
// the silent-fallback drift the table exists to kill.
func TestRuleTableCoversEveryRuleConstant(t *testing.T) {
	t.Parallel()

	constants := []string{
		RuleHW0, RuleHW1, RuleHW2, RuleHW3, RuleHW4, RuleHW5,
		RuleHW7, RuleHW8, RuleUnresolved,
	}

	for _, id := range constants {
		if !slices.Contains(RuleIDs(), id) {
			t.Errorf("rule constant %s has no RuleTable entry; add the descriptor", id)
		}
	}

	for _, id := range RuleIDs() {
		if !slices.Contains(constants, id) {
			t.Errorf("RuleTable entry %s matches no rule constant; stale entry?", id)
		}
	}
}

// TestRuleTableWellFormed pins the invariants every descriptor must hold:
// unique IDs, unique slugs, explicit severity and confidence (a zero value
// would silently de-gate a rule at the driver's exit-code mapping).
func TestRuleTableWellFormed(t *testing.T) {
	t.Parallel()

	seenIDs := map[string]bool{}
	seenSlugs := map[string]bool{}

	for _, rule := range RuleTable {
		if rule.ID == "" {
			t.Error("RuleTable entry with empty ID")
		}

		if seenIDs[rule.ID] {
			t.Errorf("duplicate RuleTable ID %s", rule.ID)
		}
		seenIDs[rule.ID] = true

		if rule.Slug == "" && rule.ID != RuleHW0 && rule.ID != RuleUnresolved {
			t.Errorf("%s: empty slug (only HW-0 and HW-unresolved live outside README §3)", rule.ID)
		}

		if rule.Slug != "" {
			if seenSlugs[rule.Slug] {
				t.Errorf("duplicate RuleTable slug %q", rule.Slug)
			}
			seenSlugs[rule.Slug] = true
		}

		if rule.Severity == "" {
			t.Errorf("%s: zero severity", rule.ID)
		}

		if rule.Confidence == 0 {
			t.Errorf("%s: zero confidence", rule.ID)
		}

		if rule.Summary == "" {
			t.Errorf("%s: empty summary (generated docs would render a hole)", rule.ID)
		}
	}
}

// TestDefaultDisabledRulesEmpty documents today's posture: every shipped rule
// is enabled by default. When a posture flip is deliberate, update this test
// in the same change — the flip must never happen by accident.
func TestDefaultDisabledRulesEmpty(t *testing.T) {
	t.Parallel()

	if got := DefaultDisabledRules(); len(got) != 0 {
		t.Errorf("DefaultDisabledRules = %v, want empty (all rules default-on)", got)
	}
}

// TestLookupRuleFoundAndMiss exercises the accessor both ways.
func TestLookupRuleFoundAndMiss(t *testing.T) {
	t.Parallel()

	desc, ok := LookupRule(RuleHW1)
	if !ok || desc.Slug != "unchecked-resource-holder" {
		t.Errorf("LookupRule(%s) = %+v, %v; want the headline descriptor", RuleHW1, desc, ok)
	}

	if _, ok := LookupRule("HW-999"); ok {
		t.Error("LookupRule(HW-999) reported a hit; unknown IDs must miss")
	}
}
