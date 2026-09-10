// want package:"healthwash: registration records"

// Package edges covers structural edge cases: the same type registered at
// multiple sites (both sites reported, coverage counts the type once), and
// registration calls nested inside closures (the AST walk must descend).
package edges

import (
	"context"

	do "github.com/samber/do/v2"
)

// Dup is registered twice below: two HW-1 sites, one coverage row.
type Dup struct{}

func (d *Dup) Shutdown(_ context.Context) {}

func NewDup(i do.Injector) (*Dup, error) { return &Dup{}, nil }

// Deep lives only inside nested closures: the registration is two closure
// levels down and must still be found.
type Deep struct{}

func (d *Deep) Shutdown(_ context.Context) {}

func NewDeep(i do.Injector) (*Deep, error) { return &Deep{}, nil }

var _ = func() bool {
	do.Provide(nil, NewDup) // want `HW-1: .*`
	do.Provide(nil, NewDup) // want `HW-1: .*`
	return true
}()

var _ = func() bool {
	do.Provide(nil, func(i do.Injector) (*Dup, error) { // want `HW-1: .*`
		do.Provide(nil, func(i2 do.Injector) (*Deep, error) { // want `HW-1: .*`
			do.Provide(nil, NewDeep) // want `HW-1: .*`
			return &Deep{}, nil
		})
		return &Dup{}, nil
	})
	return true
}()
