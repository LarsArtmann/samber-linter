// want package:"healthwash: registration records"

// Package hw4lazy: lazy registration + Healthchecker variant — green until
// first resolution. HW-4 (informational; a conscious choice).
package hw4lazy

import (
	"context"

	do "github.com/samber/do/v2"
)

type BootCritical struct{}

func (b *BootCritical) HealthCheck(context.Context) error { return nil }

func NewBootCritical(i do.Injector) (*BootCritical, error) { return &BootCritical{}, nil }

var _ = func() bool {
	do.Provide(nil, NewBootCritical) // want `HW-4: .*registered lazily`
	return true
}()
