// want package:"healthwash: registration records"

// Package unresolvable: interface-typed closure results are statically
// unknowable. Silent by default — zero diagnostics without --strict.
package unresolvable

import (
	"context"

	do "github.com/samber/do/v2"
)

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
