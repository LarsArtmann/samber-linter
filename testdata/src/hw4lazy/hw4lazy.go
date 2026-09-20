// want package:"healthwash: registration records"

// Package hw4lazy: lazy registration + Healthchecker variant — green until
// first resolution. HW-4 (informational; a conscious choice).
package hw4lazy

import (
	"context"

	do "github.com/samber/do/v2"
)

type BootCritical struct{}

// The body delegates (can fail): this fixture pins the lazy-timing rule;
// body shape is hw7nil's contract.
func (b *BootCritical) HealthCheck(context.Context) error { return b.probe() }

func (b *BootCritical) probe() error { return nil }

func NewBootCritical(i do.Injector) (*BootCritical, error) { return &BootCritical{}, nil }

var _ = func() bool {
	do.Provide(nil, NewBootCritical) // want `HW-4: .*registered lazily`
	return true
}()
