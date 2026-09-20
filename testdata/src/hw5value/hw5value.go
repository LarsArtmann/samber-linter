// want package:"healthwash: registration records"

// Package hw5value is the pointer-receiver trap: the check exists on *T only
// but the registration stores value T — the sweep's assertion never sees it.
package hw5value

import (
	"context"

	do "github.com/samber/do/v2"
)

type DeadCheck struct{}

func (d *DeadCheck) HealthCheck(context.Context) error { return nil } // want fact:`nil-body health check`

var _ = func() bool {
	do.ProvideValue(nil, DeadCheck{}) // want `HW-5: .*registered as value`
	return true
}()

type ClosureCreated struct{}

func (c *ClosureCreated) HealthCheck(context.Context) error { return nil } // want fact:`nil-body health check`

var _ = func() bool {
	do.Provide(nil, func(i do.Injector) (ClosureCreated, error) { // want `HW-5: .*`
		return ClosureCreated{}, nil
	})
	return true
}()

// The fix: register the pointer — HW-5 goes away, and HW-7 becomes visible:
// the check now RUNS and still cannot fail. Exactly the right complaint.
var _ = func() bool {
	do.ProvideValue(nil, &DeadCheck{}) // want `HW-7: .*`
	return true
}()
