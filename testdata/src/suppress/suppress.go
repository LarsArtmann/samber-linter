// want package:"healthwash: registration records"

// Package suppress exercises the suppression model: reason-required
// directives, HW-0 for missing reasons, expiry resurfacing, and `all`.
package suppress

import (
	"context"

	do "github.com/samber/do/v2"
)

type Quiet struct {
	name string
}

func (q *Quiet) Shutdown(_ context.Context) error { return nil }

func NewQuiet(i do.Injector) (*Quiet, error) { return &Quiet{}, nil }

// Valid reason: suppressed, must NOT be reported.
var _ = func() bool {
	//samber-linter:allow hw-1 quiet is a batch worker; dashboard noise not worth a check
	do.Provide(nil, NewQuiet)
	return true
}()

// No reason: HW-0 (at the site) AND the unsuppressed HW-1 both fire.
var _ = func() bool {
	//samber-linter:allow hw-1
	do.Provide(nil, NewQuiet) // want `HW-0: .*` `HW-1: .*`
	return true
}()

// Expired re-review deadline: suppression no longer applies.
var _ = func() bool {
	//samber-linter:allow hw-1 was reviewed in 2024 until 2020-01-01
	do.Provide(nil, NewQuiet) // want `HW-1: .*`
	return true
}()

// `all` with a reason suppresses everything at the next line.
var _ = func() bool {
	//samber-linter:allow all third-party wrapper; explicitly accepted
	do.Provide(nil, NewQuiet)
	return true
}()
