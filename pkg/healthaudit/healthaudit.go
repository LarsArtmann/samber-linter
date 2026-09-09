// Package healthaudit is the optional runtime companion to the samber-linter
// static analyzer (README §7). Static analysis cannot see third-party
// registrations, and cannot catch the "implements but always returns nil"
// class of fake checks. healthaudit counts, per sweep, services that were
// actually dispatched to a Healthchecker implementation versus skipped as
// non-implementers, and compares the ratio against the committed static
// baseline.
//
// Semantics (verified against samber/do v2.1.0):
//   - A service that was never built (lazy, never invoked) is SKIPPED —
//     service_lazy.go:128-134 returns nil without construction.
//   - A transient service is ALWAYS skipped — service_transient.go:62-66 is
//     an upstream TODO, and isHealthchecker() returns false unconditionally
//     (:58-60), even when the concrete type implements a check.
//   - A service whose sweep result errored is CHECKED — only a dispatched
//     check can produce an error ("healthy or doesn't implement",
//     scope.go:733-735).
//
// The dispatch-honest metrics this package exposes:
//
//	healthwash_registered  — services known to the scope (registration hook)
//	healthwash_invoked     — services actually constructed (invocation hook)
//	healthwash_errored     — services whose sweep result was non-nil (the only
//	                         rows PROVEN fail-capable at runtime)
//
// If errored/registered dips below the static analyzer's baseline, the
// runtime caught what the compiler could not.
package healthaudit

import (
	"context"
	"sort"
	"sync"

	do "github.com/samber/do/v2"
)

// Audit records per-scope service activity via samber/do's injector hooks.
type Audit struct {
	mu        sync.Mutex
	order     []string
	registered map[string]bool
	invoked   map[string]bool
	errored   map[string]bool
}

// New creates an empty Audit.
func New() *Audit {
	return &Audit{
		registered: map[string]bool{},
		invoked:    map[string]bool{},
		errored:    map[string]bool{},
	}
}

// AfterRegistration returns a do registration hook recording every service
// name entering the audited scopes. Wire it into InjectorOpts.
//
//	audit := healthaudit.New()
//	injector := do.NewWithOpts(&do.InjectorOpts{
//	    HookAfterRegistration: audit.AfterRegistration(),
//	    HookAfterInvocation:   audit.AfterInvocation(),
//	}, packages...)
func (a *Audit) AfterRegistration() func(scope *do.Scope, serviceName string) {
	return func(_ *do.Scope, serviceName string) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if !a.registered[serviceName] {
			a.registered[serviceName] = true
			a.order = append(a.order, serviceName)
		}
	}
}

// AfterInvocation records that a service was actually constructed. Unbuilt
// lazy services are the largest skipped class.
func (a *Audit) AfterInvocation() func(scope *do.Scope, serviceName string, err error) {
	return func(_ *do.Scope, serviceName string, err error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.invoked[serviceName] = true
	}
}

// Sweep runs one health sweep through the injector and records which
// services errored — the only runtime proof that a check can actually fail.
func (a *Audit) Sweep(ctx context.Context, injector do.Injector) map[string]error {
	results := injector.HealthCheckWithContext(ctx)
	a.mu.Lock()
	defer a.mu.Unlock()
	for name, err := range results {
		if err != nil {
			a.errored[name] = true
		}
	}
	return results
}

// Registered returns the number of services known to the audited scopes.
func (a *Audit) Registered() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.registered)
}

// Invoked returns the number of services that were actually constructed.
func (a *Audit) Invoked() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.invoked)
}

// Errored returns the number of services whose check demonstrably failed at
// least once — the only rows PROVEN fail-capable.
func (a *Audit) Errored() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.errored)
}

// Skipped lists services that never errored: they were either never built or
// their (unknown-quality) check passed. Against the static baseline this is
// the "green but why?" watchlist.
func (a *Audit) Skipped() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for _, name := range a.order {
		if !a.errored[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
