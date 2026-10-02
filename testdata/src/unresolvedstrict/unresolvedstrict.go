// want package:"healthwash: registration records"

// Package unresolvedstrict mirrors unresolvable for --strict runs.
package unresolvedstrict

import do "github.com/samber/do/v2"

type Checker interface {
	HealthCheck() error
}

type real struct{}

func (real) HealthCheck() error { return nil } // want HealthCheck:`nil-body health check`

// newReal returns through an interface-typed variable (no concrete return
// evidence), so the registration stays unresolvable and --strict reports
// HW-unresolved at the call site.
func newReal(i do.Injector) (Checker, error) {
	var c Checker = real{}
	return c, nil
}

var _ = func() bool {
	do.Provide(nil, newReal) // want `HW-unresolved: .*`
	return true
}()
