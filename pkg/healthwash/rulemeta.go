package healthwash

import "github.com/larsartmann/go-finding"

// RuleDescriptor is the single-source definition of one healthwash rule. The
// CLI driver's severity/confidence table, the golangci-lint plugin's docs, and
// the README rule-table drift test all derive from RuleTable, so a posture
// change (severity, confidence, default-enabled) is a one-line edit in one
// file instead of four hand-maintained lists.
type RuleDescriptor struct {
	// ID is the stable rule identifier ("HW-1"); it never changes once
	// shipped — suppressions and configs key on it.
	ID string

	// Slug is the rule name used in README §3 headings
	// ("unchecked-resource-holder"); empty for rules documented outside the
	// rule table (HW-0, HW-unresolved).
	Slug string

	// Severity is how bad the finding is if real; Confidence is how sure the
	// rule is that it is real. Only confidence maps to exit codes.
	Severity   finding.Severity
	Confidence finding.Confidence

	// Summary is the one-line description consumed by generated docs.
	Summary string

	// DefaultEnabled is the rule's default posture. Flipping it here is the
	// one-line posture change; the driver and plugin merge
	// DefaultDisabledRules into the analyzer's disable flag, and an explicit
	// --disable still wins on top.
	DefaultEnabled bool
}

// RuleTable lists every rule the analyzer can emit, in README §3 order, then
// the suppression meta-rule and the strict-mode placeholder. HW-6
// (health-coverage-ratchet) is absent by design: it is the driver-level CI
// gate, not an analyzer-emitted diagnostic.
//
//nolint:gochecknoglobals // read-only single-source rule table
var RuleTable = []RuleDescriptor{
	{
		ID:             RuleHW1,
		Slug:           "unchecked-resource-holder",
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "Shutdowner without Healthchecker renders an unconditional pass",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW2,
		Slug:           "contextless-check",
		Severity:       finding.SeverityInfo,
		Confidence:     finding.ConfidenceHigh,
		Summary:        "bare check cannot be cancelled by the sweep's per-service timeout",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW3,
		Slug:           "transient-health-washing",
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "the transient healthcheck is an upstream TODO that always returns nil",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW4,
		Slug:           "lazy-never-built-pass",
		Severity:       finding.SeverityInfo,
		Confidence:     finding.ConfidenceMedium,
		Summary:        "lazy service reports healthy until first resolution builds it",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW5,
		Slug:           "pointer-receiver-value-registration",
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "pointer-receiver check is unreachable from a value registration",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW7,
		Slug:           "unconditional-nil-check",
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "the check's body is exactly `return nil`; it can never fail",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW8,
		Slug:           "empty-check-body",
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "the check's body is a lone naked return on a named nil result",
		DefaultEnabled: true,
	},
	{
		ID:             RuleHW0,
		Severity:       finding.SeverityWarning,
		Confidence:     finding.ConfidenceFull,
		Summary:        "suppression directive without a reason",
		DefaultEnabled: true,
	},
	{
		ID:             RuleUnresolved,
		Severity:       finding.SeverityInfo,
		Confidence:     finding.ConfidenceMedium,
		Summary:        "service type could not be resolved statically (strict mode only)",
		DefaultEnabled: true,
	},
}

// LookupRule returns the descriptor for one rule ID ("HW-1").
func LookupRule(id string) (RuleDescriptor, bool) {
	for _, rule := range RuleTable {
		if rule.ID == id {
			return rule, true
		}
	}

	return RuleDescriptor{}, false
}

// RuleIDs returns every rule ID in RuleTable, in table order.
func RuleIDs() []string {
	ids := make([]string, 0, len(RuleTable))
	for _, rule := range RuleTable {
		ids = append(ids, rule.ID)
	}

	return ids
}

// DefaultDisabledRules returns the IDs of rules whose default posture is off.
// The driver and the plugin merge this into the analyzer's disable flag so
// the default posture travels with the table; it is empty while every rule
// ships enabled.
func DefaultDisabledRules() []string {
	var disabled []string
	for _, rule := range RuleTable {
		if !rule.DefaultEnabled {
			disabled = append(disabled, rule.ID)
		}
	}

	return disabled
}
