// want package:"healthwash: registration records"

// Package main registers types declared in hw7cross/lib: HW-7 fires at the
// registration site for ForeignNil (fact imported from lib) and stays silent
// for ForeignReal.
package main

import do "github.com/samber/do/v2"

import "hw7cross/lib"

var _ = func() bool {
	do.ProvideValue(nil, &lib.ForeignNil{}) // want `HW-7: .*`
	do.ProvideValue(nil, &lib.ForeignReal{})
	return true
}()
