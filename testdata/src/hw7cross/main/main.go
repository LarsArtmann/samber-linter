// want package:"healthwash: registration records"

// Package main registers types declared in hw7cross/lib: HW-7 fires at the
// registration site for ForeignNil (fact imported from lib) and stays silent
// for ForeignReal. newForeignNil pins the composition of BOTH cross-package
// channels at one site: provider-body resolution finds the concrete instance
// in main's own newForeignNil body, and the nil-body verdict still crosses
// the package boundary via the exported fact.
package main

import (
	"context"

	"hw7cross/lib"

	do "github.com/samber/do/v2"
)

// Backend is main's registration surface; the concrete evidence behind it is
// declared in another package.
type Backend interface {
	HealthCheck(context.Context) error
}

func newForeignNil(i do.Injector) (Backend, error) { return &lib.ForeignNil{}, nil }

var _ = func() bool {
	do.ProvideValue(nil, &lib.ForeignNil{}) // want `HW-7: .*`
	do.ProvideValue(nil, &lib.ForeignReal{})
	do.Provide(nil, newForeignNil) // want `HW-4: .*` `HW-7: .*`
	return true
}()
