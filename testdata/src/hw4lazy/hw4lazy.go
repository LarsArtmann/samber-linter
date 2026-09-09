// Package hw4lazy: a lazy registration whose type implements the context
// health check — until first resolution the sweep reports green without
// constructing the service. HW-4 fires (informational; a conscious choice).
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
