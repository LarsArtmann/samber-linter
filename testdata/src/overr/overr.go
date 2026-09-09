// want package:"healthwash: registration records"

// Package overr exercises the Override* family (di.go:187-251): identical
// wrappers, identical semantics. Pointer-registration overrides fire HW-1;
// the value override with pointer-receiver Shutdown is a different trap
// (the sweep sees neither interface) and stays clean under HW-5's scope.
package overr

import (
	"context"

	do "github.com/samber/do/v2"
)

type Resource struct {
	conns []context.CancelFunc
}

func (r *Resource) Shutdown(context.Context) {}

func NewResource(i do.Injector) (*Resource, error) { return &Resource{}, nil }

var _ = func() bool {
	do.Override(nil, NewResource) // want `HW-1: \*Resource`
	return true
}()

var _ = func() bool {
	do.OverrideNamed(nil, "resource-2", NewResource) // want `HW-1: \*Resource`
	return true
}()

var _ = func() bool {
	do.OverrideValue(nil, Resource{})
	return true
}()

// Honest eager override with both interfaces: clean.
type HonestEager struct{}

func (h *HonestEager) Shutdown(context.Context)          {}
func (h *HonestEager) HealthCheck(context.Context) error { return nil }

var _ = func() bool {
	do.OverrideValue(nil, &HonestEager{})
	return true
}()

// Aliases: tracked, never attributed.
type IResource interface {
	HealthCheck(context.Context) error
}

var _ = func() bool {
	do.As[*HonestEager, IResource](nil)
	return true
}()
