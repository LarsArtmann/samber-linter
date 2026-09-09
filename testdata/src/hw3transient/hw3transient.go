// Package hw3transient: a transient registration whose type implements a
// Healthchecker variant — the transient healthcheck is an upstream TODO that
// always returns nil, so the implementation is dead code advertising false
// confidence. HW-3 fires.
package hw3transient

import (
	"context"

	do "github.com/samber/do/v2"
)

type PerRequest struct{}

func (p *PerRequest) HealthCheck(context.Context) error { return nil }

func NewPerRequest(i do.Injector) (*PerRequest, error) { return &PerRequest{}, nil }

var _ = func() bool {
	do.ProvideTransient(nil, NewPerRequest) // want `HW-3: .*transiently \(ProvideTransient\)`
	return true
}()

var _ = func() bool {
	do.OverrideTransient(nil, NewPerRequest) // want `HW-3: .*transiently \(OverrideTransient\)`
	return true
}()
