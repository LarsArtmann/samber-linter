// want package:"healthwash: registration records"

// Package hw7nil is the unconditional-nil corpus: health checks whose body
// is exactly `return nil` (HW-7), next to the shapes v1 deliberately leaves
// negative — delegation, multi-statement bodies, unreachable bodies, and
// types never registered in a container.
package hw7nil

import (
	"context"

	do "github.com/samber/do/v2"
)

// NilEager: pointer registration, ctx variant, body is `return nil` — the
// canonical HW-7 site.
type NilEager struct{}

func (n *NilEager) HealthCheck(context.Context) error { return nil }

func NewNilEager() *NilEager { return &NilEager{} }

var _ = func() bool {
	do.ProvideValue(nil, NewNilEager()) // want `HW-7: .*`
	return true
}()

// NilLazy: lazy registration with a nil body reports HW-4 (green until first
// resolution) AND HW-7 (green forever after) — two diseases, two fixes.
type NilLazy struct{}

func (n *NilLazy) HealthCheck(context.Context) error { return nil }

func NewNilLazy(i do.Injector) (*NilLazy, error) { return &NilLazy{}, nil }

var _ = func() bool {
	do.Provide(nil, NewNilLazy) // want `HW-4: .*` `HW-7: .*`
	return true
}()

// NilBare: bare variant on a value registration — HW-2 and HW-7 compose.
type NilBare struct{}

func (n NilBare) HealthCheck() error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, NilBare{}) // want `HW-2: .*` `HW-7: .*`
	return true
}()

// Delegates: `return s.probe()` can fail — negative in v1.
type Delegates struct{}

func (d *Delegates) HealthCheck(context.Context) error { return d.probe() }

func (d *Delegates) probe() error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, &Delegates{})
	return true
}()

// MultiStatement: log-then-nil needs flow data to judge — negative in v1.
type MultiStatement struct{}

func (m *MultiStatement) HealthCheck(_ context.Context) error {
	_ = m
	return nil
}

var _ = func() bool {
	do.ProvideValue(nil, &MultiStatement{})
	return true
}()

// PointerOnlyNil: check on *T registered as value T — HW-5 owns the site;
// the sweep never reaches that body, so HW-7 must stay silent.
type PointerOnlyNil struct{}

func (p *PointerOnlyNil) HealthCheck(context.Context) error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, PointerOnlyNil{}) // want `HW-5: .*`
	return true
}()

// Unregistered: a nil body on a type no container stores is out of scope —
// the linter detects container health-washing, not style.
type Unregistered struct{}

func (u *Unregistered) HealthCheck(context.Context) error { return nil }
