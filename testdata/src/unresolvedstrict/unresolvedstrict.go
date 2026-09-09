// Package unresolvedstrict mirrors the unresolvable fixture for --strict
// runs: the interface-typed closure result emits HW-unresolved here.
package unresolvedstrict

import (
	do "github.com/samber/do/v2"
)

type Checker interface {
	HealthCheck() error
}

type real struct{}

func (real) HealthCheck() error { return nil }

func newReal(i do.Injector) (Checker, error) { return real{}, nil }

var _ = func() bool {
	do.Provide(nil, newReal) // want `HW-unresolved: .*`
	return true
}()
