package healthwash

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DirectivePrefix is the namespaced suppression directive. Chosen once,
// never changed — suppressions and configs key on it.
const DirectivePrefix = "samber-linter:allow"

// Directive is one parsed suppression comment.
type Directive struct {
	// Rule is the lowercased rule token ("hw-1" … "hw-6", "hw-0",
	// "hw-unresolved", or "all").
	Rule string
	// Reason is the mandatory free-text justification. Empty means the
	// directive itself is a finding (HW-0).
	Reason string
	// Expires is the optional "until YYYY-MM-DD" re-review deadline.
	Expires *time.Time
}

// directiveRe matches `//samber-linter:allow <token> <reason...>` with an
// optional space after the slashes for editor-wrapping friendliness.
var directiveRe = regexp.MustCompile(`^//\s*` + DirectivePrefix + `\s+(\S)(\S*)\s*(.*)$`)

// untilRe matches a trailing "until 2027-01-02" re-review clause.
var untilRe = regexp.MustCompile(`\s+until\s+(\d{4}-\d{2}-\d{2})\s*$`)

// ParseDirective parses one comment line. ok is false when the line is not a
// suppression directive at all. ParseDirective never fails on malformed
// directives: a missing rule token or empty reason still yields ok=true with
// the parts that parsed, so callers can raise HW-0 — unexplained suppressions
// must not silently vanish.
func ParseDirective(line string) (Directive, bool) {
	trimmed := strings.TrimRight(line, " \t")
	m := directiveRe.FindStringSubmatch(trimmed)
	if m == nil {
		// Accept a bare prefix with nothing after it as a malformed directive.
		if strings.HasPrefix(strings.TrimSpace(trimmed), "//") &&
			strings.Contains(strings.TrimSpace(trimmed), DirectivePrefix) {
			return Directive{}, true
		}
		return Directive{}, false
	}

	rule := strings.ToLower((m[1] + m[2]))
	rest := strings.TrimSpace(m[3])

	d := Directive{Rule: rule, Reason: rest}

	if mUntil := untilRe.FindStringSubmatch(rest); mUntil != nil {
		if t, err := time.Parse("2006-01-02", mUntil[1]); err == nil {
			d.Expires = &t
			d.Reason = strings.TrimSpace(untilRe.ReplaceAllString(rest, ""))
		}
	}
	return d, true
}

// Expired reports whether the directive's re-review deadline has passed.
func (d Directive) Expired(now time.Time) bool {
	return d.Expires != nil && now.After(*d.Expires)
}

// String renders the directive back to source form.
func (d Directive) String() string {
	s := "//" + DirectivePrefix + " " + d.Rule
	if d.Reason != "" {
		s += " " + d.Reason
	}
	if d.Expires != nil {
		s += fmt.Sprintf(" until %s", d.Expires.Format("2006-01-02"))
	}
	return s
}
