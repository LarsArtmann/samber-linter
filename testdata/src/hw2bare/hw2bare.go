// Package hw2bare: implements the bare HealthCheck() without the context
// variant — a hung bare check cannot be cancelled and degrades the sweep.
// HW-2 fires (info). Registered eagerly with the value implementing the
// check, so no HW-1/HW-4/HW-5 applies.
package hw2bare

import do "github.com/samber/do/v2"

type NoCtx struct{}

func (n NoCtx) HealthCheck() error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, NoCtx{}) // want `HW-2: .*`
	return true
}()
