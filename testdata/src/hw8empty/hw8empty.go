// want package:"healthwash: registration records"

// Package hw8empty is the naked-return check corpus (HW-8): a body whose only
// statement is a bare `return` on a named result — the implicit zero value
// makes the check a silent no-op. Panics, sole `return nil` (HW-7's shape),
// and naked returns among other statements are negative.
package hw8empty

import (
	"context"

	do "github.com/samber/do/v2"
)

// NakedEager: named result, sole naked return — the canonical HW-8 site.
type NakedEager struct{}

func (n *NakedEager) HealthCheck(context.Context) (err error) { return } // want HealthCheck:`naked-return health check`

var _ = func() bool {
	do.ProvideValue(nil, &NakedEager{}) // want `HW-8: .*`
	return true
}()

// NakedLazy: lazy registration, naked return — HW-4 and HW-8 compose.
type NakedLazy struct{}

func (n *NakedLazy) HealthCheck(context.Context) (err error) { return } // want HealthCheck:`naked-return health check`

func NewNakedLazy(i do.Injector) (*NakedLazy, error) { return &NakedLazy{}, nil }

var _ = func() bool {
	do.Provide(nil, NewNakedLazy) // want `HW-4: .*` `HW-8: .*`
	return true
}()

// SoleNil: exactly one explicit `return nil` — HW-7's shape, NOT HW-8 (the
// predicates are disjoint: one result vs none).
type SoleNil struct{}

func (s *SoleNil) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

var _ = func() bool {
	do.ProvideValue(nil, &SoleNil{}) // want `HW-7: .*`
	return true
}()

// NakedAmongStatements: a naked return after other statements — negative; err
// may have been set above it.
type NakedAmongStatements struct{}

func (n *NakedAmongStatements) HealthCheck(context.Context) (err error) {
	_ = n
	return
}

var _ = func() bool {
	do.ProvideValue(nil, &NakedAmongStatements{})
	return true
}()

// Delegates: the sole statement carries a call result — negative.
type Delegates struct{}

func (d *Delegates) HealthCheck(context.Context) (err error) { return d.probe() }

func (d *Delegates) probe() error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, &Delegates{})
	return true
}()
