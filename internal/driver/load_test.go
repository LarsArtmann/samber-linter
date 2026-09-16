package driver

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// hostileGoFlags sets a global GOFLAGS that breaks package loading in a repo
// without a vendor directory. The loader must sanitize the inherited value
// while still honoring explicit opts.Env overrides (last occurrence wins).
func hostileGoFlags(t *testing.T) {
	t.Helper()
	t.Setenv("GOFLAGS", "-mod=vendor")
}

// The Setenv-based load tests cannot run in parallel (t.Setenv panics under
// t.Parallel because the env mutation is process-global), so the paralleltest
// rule is intentionally suppressed here.
//
//nolint:paralleltest // t.Setenv is incompatible with t.Parallel
func TestLoadSanitizesInheritedGoFlags(t *testing.T) {
	hostileGoFlags(t)

	app := e2eModule(t)

	pkgs, err := load(context.Background(), Options{Dir: app, Patterns: []string{"./..."}})
	if err != nil {
		t.Fatalf("load under inherited GOFLAGS=-mod=vendor returned error: %v", err)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			t.Errorf("package %s loaded with errors under hostile GOFLAGS: %v", pkg.PkgPath, pkg.Errors)
		}
	}
}

//nolint:paralleltest // t.Setenv is incompatible with t.Parallel
func TestLoadExplicitEnvOverridesHostileInherited(t *testing.T) {
	hostileGoFlags(t)

	app := e2eModule(t)

	opts := Options{
		Dir:      app,
		Patterns: []string{"./..."},
		Env:      []string{"GOFLAGS=-mod=mod"},
	}

	pkgs, err := load(context.Background(), opts)
	if err != nil {
		t.Fatalf("load with explicit GOFLAGS override returned error: %v", err)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			t.Errorf("package %s loaded with errors despite explicit override: %v", pkg.PkgPath, pkg.Errors)
		}
	}
}

func TestLoadEnvSanitizedEntryAlwaysPresent(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=vendor -count=1")

	env := loadEnv()

	entry, found := "", false

	for _, pair := range env {
		if key, value, ok := strings.Cut(pair, "="); ok && key == "GOFLAGS" {
			entry, found = value, true
		}
	}

	if !found {
		t.Fatal("loadEnv produced no GOFLAGS entry; the sanitized shadow must always be present")
	}

	if entry != "-count=1" {
		t.Errorf("sanitized GOFLAGS = %q, want %q", entry, "-count=1")
	}

	if slices.Contains(env, "GOFLAGS=-mod=vendor -count=1") {
		t.Error("unsanitized inherited GOFLAGS survived in loadEnv output")
	}
}
