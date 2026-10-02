// want package:"healthwash: registration records"

// Package ifacebody pins provider-body resolution: an interface-typed
// provider signature whose body returns one concrete instance type is
// analyzed as that concrete type (README §4 step 2 — the sweep asserts the
// stored instance, not the registration interface).
package ifacebody

import (
	"context"

	do "github.com/samber/do/v2"
)

type Backend interface {
	HealthCheck(context.Context) error
}

type pg struct{}

func (pg) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

func newBackend(i do.Injector) (Backend, error) { return pg{}, nil }

var _ = func() bool {
	do.Provide(nil, newBackend) // want `HW-4: .*` `HW-7: .*`
	return true
}()
