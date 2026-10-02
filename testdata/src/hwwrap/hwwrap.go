// want package:"healthwash: registration records"

// Package hwwrap: registrations hidden behind repo-local wrapper functions
// and generics — the 2026-09-20 consumer false negative. Direct-call matching
// saw none of these sites; one-level param→arg resolution attributes every
// rule at the wrapper CALL SITE (where the fix lands) and never double-counts
// the parameterized body call.
package hwwrap

import (
	"context"

	do "github.com/samber/do/v2"
)

type CheckedStore struct{}

func (s *CheckedStore) HealthCheck(context.Context) error { return s.probe() }
func (s *CheckedStore) probe() error                      { return nil }
func NewCheckedStore(do.Injector) (*CheckedStore, error)  { return &CheckedStore{}, nil }

type ShutStore struct{ conn int }

func (s *ShutStore) Shutdown(context.Context)      {}
func NewShutStore(do.Injector) (*ShutStore, error) { return &ShutStore{}, nil }

type PtrStore struct{}

func (p *PtrStore) HealthCheck(context.Context) error { return p.ping() }
func (p *PtrStore) ping() error                       { return nil }

// Backend + pg pin the wrapper × provider-body composition: the wrapper
// substitutes the call-site provider argument, and provider-body resolution
// must then see newBackend's body in THIS package — one concrete return
// through an interface signature, exactly like the ifacebody fixture, but
// reached through the wrapper channel.
type Backend interface {
	HealthCheck(context.Context) error
}

type pg struct{}

func (pg) HealthCheck(context.Context) error { return nil } // want HealthCheck:`nil-body health check`

func newBackend(i do.Injector) (Backend, error) { return pg{}, nil }

// Handler implements no lifecycle interface: the wrapper channel must not
// make rule precision worse.
type Handler struct{ dep *CheckedStore }

func NewHandler(do.Injector) (*Handler, error) { return &Handler{}, nil }

// provideNamed is the verified consumer shape: a generic wrapper around
// do.ProvideNamed. Its body call is parameterized — never concrete — and must
// not be reported or counted by itself.
func provideNamed[T any](i do.Injector, name string, provider do.Provider[T]) {
	do.ProvideNamed(i, name, provider)
}

func provide[T any](i do.Injector, provider do.Provider[T]) {
	do.Provide(i, provider)
}

func provideValue[T any](i do.Injector, value T) {
	do.ProvideValue(i, value)
}

var _ = func() bool {
	provideNamed[*CheckedStore](nil, "checked", NewCheckedStore) // want `HW-4: .*registered lazily`
	provide(nil, NewShutStore)                                   // want `HW-1: .*implements do.Shutdowner`
	provideValue(nil, PtrStore{})                                // want `HW-5: .*declares its health check on receiver`
	provide(nil, newBackend)                                     // want `HW-4: .*` `HW-7: .*`

	//samber-linter:allow hw-4 checked store is resolved during boot
	provideNamed(nil, "checked-allowed", NewCheckedStore)

	provide(nil, NewHandler)
	return true
}()
