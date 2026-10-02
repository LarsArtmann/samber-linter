// want package:"healthwash: registration records"

// Package unresolvable: registrations whose provider returns only
// interface-typed expressions carry no concrete type evidence and stay
// statically unknowable. Silent by default — zero diagnostics without
// --strict. (A direct `return real{}, nil` IS resolvable — provider-body
// inspection extracts the concrete instance; see the ifacebody fixture.)
package unresolvable

import (
	"context"

	do "github.com/samber/do/v2"
)

type Checker interface {
	HealthCheck(context.Context) error
}

type real struct{}

func (real) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

// newReal returns through an interface-typed variable: no return statement
// carries concrete type evidence, so the registration stays unresolvable
// even with provider-body inspection (isConcrete skips interfaces).
func newReal(i do.Injector) (Checker, error) {
	var c Checker = real{}
	return c, nil
}

var _ = func() bool {
	do.Provide(nil, newReal)
	return true
}()
