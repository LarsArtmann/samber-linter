// Package unresolvable: interface-typed closure results are statically
// unknowable (the sweep asserts the stored concrete instance). Silent by
// default — this fixture must produce ZERO diagnostics without --strict.
package unresolvable

import (
	"context"

	do "github.com/samber/do/v2"
)

// Checker is the registration interface; which concrete type lands in the
// container is a runtime decision.
type Checker interface {
	HealthCheck(context.Context) error
}

type real struct{}

func (real) HealthCheck(context.Context) error { return nil }

func newReal(i do.Injector) (Checker, error) { return real{}, nil }

var _ = func() bool {
	do.Provide(nil, newReal)
	return true
}()
