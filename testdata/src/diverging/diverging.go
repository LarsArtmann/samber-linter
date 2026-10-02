// want package:"healthwash: registration records"

// Package diverging: a provider body whose returns name TWO different
// concrete types is a run-time choice the type checker cannot make — the
// registration stays unresolvable and silent (README §4: divergence between
// multiple implementations). Pins the resolver's divergence guard: neither
// aStore's nil-body check (HW-7/HW-4) nor bStore's shutdown-without-check
// (HW-1) may fire on statically unknowable evidence.
package diverging

import (
	"context"

	do "github.com/samber/do/v2"
)

type Repo interface {
	Get() string
}

type aStore struct{}

func (aStore) Get() string                       { return "" }
func (aStore) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

type bStore struct{}

func (bStore) Get() string              { return "" }
func (bStore) Shutdown(context.Context) {}

var useA = true

func newRepo(i do.Injector) (Repo, error) {
	if useA {
		return aStore{}, nil
	}

	return bStore{}, nil
}

var _ = func() bool {
	do.Provide(nil, newRepo)
	return true
}()
