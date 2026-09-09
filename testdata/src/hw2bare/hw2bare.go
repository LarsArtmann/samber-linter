// want package:"healthwash: registration records"

// Package hw2bare: bare HealthCheck() without the context variant. HW-2
// (info). Eager value registration with the check on the value: no HW-1/4/5.
package hw2bare

import do "github.com/samber/do/v2"

type NoCtx struct{}

func (n NoCtx) HealthCheck() error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, NoCtx{}) // want `HW-2: .*`
	return true
}()
