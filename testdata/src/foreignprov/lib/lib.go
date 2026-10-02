// Package lib declares the provider AND its return interface; main only
// registers. The provider body lives here, outside the registering package's
// file set: provider-body inspection must NOT see it (the same doctrine as
// HW-7's cross-package method bodies — facts about bodies cross packages,
// provider bodies do not), so the interface-typed signature leaves the
// registration unresolvable and silent.
package lib

import (
	"context"

	do "github.com/samber/do/v2"
)

// Storage is the registration surface main sees.
type Storage interface {
	HealthCheck(context.Context) error
}

// conn implements Storage with a nil-body check: if the body were visible at
// the registration site, HW-7 (and HW-4, lazy) would fire there. It must not.
type conn struct{}

func (conn) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

// NewConn's body returns exactly one concrete type — resolvable when declared
// in the registering package (see ifacebody), invisible from another package.
func NewConn(i do.Injector) (Storage, error) { return conn{}, nil }
