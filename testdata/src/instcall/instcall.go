// want package:"healthwash: registration records"

// Package instcall pins explicitly instantiated registration calls:
// do.Provide[Backend](...) must produce the same records and rules as the
// inferred form (pkg/healthwash/wrapper.go doRegistrationCall unwraps
// *ast.IndexExpr/*ast.IndexListExpr before the selector match). Without the
// unwrap these registrations were invisible — no record, no rule findings,
// not even counted in the HW-6 denominator — while README §4 step 2 names
// Provide[SomeInterface] as a supported shape.
package instcall

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
	do.Provide[Backend](nil, newBackend)                            // want `HW-4: .*` `HW-7: .*`
	do.Provide[Backend](nil, func(_ do.Injector) (Backend, error) { // want `HW-4: .*` `HW-7: .*`
		return pg{}, nil
	})
	return true
}()
