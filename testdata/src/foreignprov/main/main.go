// want package:"healthwash: registration records"

// Package main registers a provider declared in another package: the body is
// invisible here, the signature is interface-typed, and the registration
// stays unresolvable — silent without --strict. Pins the cross-package
// doctrine: facts about method bodies cross packages, provider bodies do not.
package main

import (
	do "github.com/samber/do/v2"
	"foreignprov/lib"
)

var _ = func() bool {
	do.Provide(nil, lib.NewConn)
	return true
}()
