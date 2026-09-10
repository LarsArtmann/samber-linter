// want package:"healthwash: registration records"

// Package suppressspan exercises directive placement across multi-line
// registration calls: a directive inside the argument list spans the whole
// call, and an orphaned malformed directive is reported even with no
// violation to attach to.
package suppressspan

import (
	"context"

	do "github.com/samber/do/v2"
)

type Quiet struct {
	name string
}

func (q *Quiet) Shutdown(_ context.Context) error { return nil }

func NewQuiet(i do.Injector) (*Quiet, error) { return &Quiet{}, nil }

// Directive inside the multi-line provider closure suppresses the HW-1 at
// the call head.
var _ = func() bool {
	do.Provide(nil, func(i do.Injector) (*Quiet, error) {
		//samber-linter:allow hw-1 quiet is a batch worker; dashboard noise not worth a check
		return &Quiet{}, nil
	})
	return true
}()

// A malformed directive with nothing to attach to still surfaces as HW-0 at
// the comment: unexplained suppressions rot even when the site is clean.
var _ = func() bool {
	//samber-linter:allow hw-1 // want `HW-0: .*`
	return true
}()
