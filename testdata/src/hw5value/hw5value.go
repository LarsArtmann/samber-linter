// Package hw5value is the pointer-receiver trap: the health check exists on
// receiver *T only, but the registration stores value T — the sweep's type
// assertion never sees the check. HW-5 fires on both registration styles.
package hw5value

import (
	"context"

	do "github.com/samber/do/v2"
)

// DeadCheck reads like a plausible implementation — that is exactly why HW-5
// is the nastiest variant (README §2.7).
type DeadCheck struct{}

func (d *DeadCheck) HealthCheck(context.Context) error { return nil }

var _ = func() bool {
	do.ProvideValue(nil, DeadCheck{}) // want `HW-5: .*registered as value`
	return true
}()

type ClosureCreated struct{}

func (c *ClosureCreated) HealthCheck(context.Context) error { return nil }

var _ = func() bool {
	do.Provide(nil, func(i do.Injector) (ClosureCreated, error) { // want `HW-5: .*`
		return ClosureCreated{}, nil
	})
	return true
}()

// The fix: register the pointer.
var _ = func() bool {
	do.ProvideValue(nil, &DeadCheck{})
	return true
}()
