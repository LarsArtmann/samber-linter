// Package lib declares the services; the registrations live in hw7cross/main.
// HW-7's body verdict is exported here as a NilBodyFact and imported at the
// registration site — the standard declare-here/register-there architecture.
// (No package fact is exported here: this package does not import samber/do.)
package lib

import "context"

// ForeignNil lives in this package, is registered in main: HW-7 must reach
// across the package boundary via the exported fact.
type ForeignNil struct{}

func (f *ForeignNil) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

// ForeignReal delegates — the fact must NOT be exported (negative).
type ForeignReal struct{}

func (f *ForeignReal) HealthCheck(context.Context) error { return f.probe() }

func (f *ForeignReal) probe() error { return nil }
