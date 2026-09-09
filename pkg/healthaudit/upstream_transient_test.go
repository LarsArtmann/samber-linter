//go:build upstream_issue

// This file is the executable core of the upstream issue draft
// (docs/upstream/ISSUE_DRAFT.md). It asserts the behavior samber/do SHOULD
// have: a transient service implementing HealthcheckerWithContext must have
// its check dispatched by the sweep.
//
// It FAILS on samber/do v2.1.0 (service_transient.go:62-66 is an upstream
// TODO returning nil unconditionally). Run it with:
//
//	go test -tags upstream_issue ./pkg/healthaudit/ -run TestTransientCheckShouldDispatch -v
package healthaudit

import (
	"context"
	"errors"
	"testing"

	do "github.com/samber/do/v2"
)

type perRequestCheck struct{}

var errTransientDown = errors.New("transient service down")

func (perRequestCheck) HealthCheck(context.Context) error { return errTransientDown }

func TestTransientCheckShouldDispatch(t *testing.T) {
	injector := do.New(func(i do.Injector) {
		do.ProvideTransient(i, func(i do.Injector) (perRequestCheck, error) {
			return perRequestCheck{}, nil
		})
	})

	results := injector.HealthCheckWithContext(context.Background())

	svcName := "example.com/healthaudit.perRequestCheck"
	err, ok := results[svcName]
	if !ok {
		t.Fatalf("transient service %s missing from sweep results: %v", svcName, results)
	}
	if !errors.Is(err, errTransientDown) {
		t.Fatalf("transient service implements a check but the sweep never dispatched it: got %v, want %v", err, errTransientDown)
	}
}
