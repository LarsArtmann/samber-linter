// want package:"healthwash: registration records"

// Package hw0orphan holds a malformed suppression directive with no
// registration violation to attach to. It must still surface as HW-0 at the
// comment itself: unexplained suppressions rot even when the site is clean.
// Asserted manually in TestOrphanedDirectiveHW0 — a `// want` comment on the
// directive line would parse as the missing reason and make it valid.
package hw0orphan

import (
	"context"

	do "github.com/samber/do/v2"
)

type Clean struct{}

func (c *Clean) HealthCheck(_ context.Context) error { return nil }

func NewClean(i do.Injector) (*Clean, error) { return &Clean{}, nil }

// Registered properly (ctx check, eager): nothing to report here. The bare
// malformed directive below is the only diagnostic source in this package.
var _ = func() bool {
	do.ProvideValue(nil, &Clean{})
	return true
}()

var _ = func() bool {
	//samber-linter:allow hw-1
	return true
}()
