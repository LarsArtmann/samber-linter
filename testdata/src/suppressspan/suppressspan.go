// want package:"healthwash: registration records"

// Package suppressspan exercises directive placement across multi-line
// registration calls: a directive inside the argument list spans the whole
// call, so the HW-1 at the call head is suppressed. The orphaned malformed
// directive lives in hw0orphan: its expected-diagnostic comment would
// otherwise parse as the reason and make it valid.
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
