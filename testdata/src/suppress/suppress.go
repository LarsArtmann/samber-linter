// Package suppress exercises the suppression model: a valid directive with a
// reason suppresses the finding; a directive without a reason is itself a
// finding (HW-0); an expired `until` deadline lets the finding resurface.
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

// Valid reason: the HW-1 below is suppressed and must NOT be reported.
var _ = func() bool {
	//samber-linter:allow hw-1 quiet is a batch worker; dashboard noise not worth a check
	do.Provide(nil, NewQuiet)
	return true
}()

// No reason: the directive itself is a finding (HW-0) and the HW-1 fires.
var _ = func() bool {
	//samber-linter:allow hw-1
	do.Provide(nil, NewQuiet) // want `HW-1: .*`
	return true
}()

// Expired re-review deadline: suppression no longer applies, HW-1 fires.
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
