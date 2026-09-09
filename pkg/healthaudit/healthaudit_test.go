package healthaudit

import (
	"context"
	"errors"
	"testing"

	do "github.com/samber/do/v2"
)

type healthy struct{}

func (healthy) HealthCheck(context.Context) error { return nil }

type flaky struct{}

var errBoom = errors.New("boom")

func (flaky) HealthCheck(context.Context) error { return errBoom }

func TestAuditCounts(t *testing.T) {
	audit := New()

	injector := do.NewWithOpts(&do.InjectorOpts{
		HookAfterRegistration: []func(*do.Scope, string){audit.AfterRegistration()},
		HookAfterInvocation:   []func(*do.Scope, string, error){audit.AfterInvocation()},
	}, func(i do.Injector) {
		do.Provide(i, func(i do.Injector) (healthy, error) { return healthy{}, nil })
		do.Provide(i, func(i do.Injector) (*flaky, error) { return &flaky{}, nil })
	})
	if injector == nil {
		t.Fatal("injector nil")
	}

	if got := audit.Registered(); got != 2 {
		t.Fatalf("registered = %d, want 2", got)
	}

	// HW-4 semantics in action: the sweep does NOT build lazy services —
	// service_lazy.go returns nil for unbuilt services, so both would be
	// skipped. Resolve them first; production code that cares about sweep
	// honesty must do the same (or register boot-critical services eagerly).
	if _, err := do.Invoke[healthy](injector); err != nil {
		t.Fatalf("invoke healthy: %v", err)
	}
	if _, err := do.Invoke[*flaky](injector); err != nil {
		t.Fatalf("invoke flaky: %v", err)
	}

	results := audit.Sweep(context.Background(), injector)

	if len(results) == 0 {
		t.Fatal("sweep returned no results")
	}
	// Only the failing service is PROVEN fail-capable.
	if got := audit.Errored(); got != 1 {
		t.Fatalf("errored = %d, want 1 (only a non-nil result proves a check can fail)", got)
	}
	skipped := audit.Skipped()
	if len(skipped) != 1 || !strings_Contains(skipped, "example.com/healthaudit/healthy") &&
		!strings_Contains(skipped, "healthy") {
		t.Fatalf("skipped = %v, want the healthy service only", skipped)
	}
}

func strings_Contains(list []string, needle string) bool {
	for _, s := range list {
		if s == needle || (len(s) > 0 && len(needle) > 0 && contains(s, needle)) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
