package main

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// mustBuildOptions collapses the repeated build-and-fail dance so each
// subtest below carries only its assertion branches (gocognit nests closure
// complexity into the parent test function).
func mustBuildOptions(t *testing.T, flags *linterFlags, patterns []string, version string) driver.Options {
	t.Helper()

	opts, err := buildDriverOptions(flags, patterns, version)
	if err != nil {
		t.Fatalf("buildDriverOptions: %v", err)
	}

	return opts
}

// TestBuildDriverOptions pins the flag-struct-to-driver.Options mapping: the
// two defaults the flag tags cannot carry (pattern fallback, baseline path
// fallback) and the pass-through of every explicit value. A drift here changes
// CLI behavior without touching the flag surface tests.
func TestBuildDriverOptions(t *testing.T) {
	t.Parallel()

	t.Run("empty baseline falls back to the driver default", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{}, nil, "test")

		if opts.BaselinePath != driver.DefaultBaselinePath {
			t.Errorf("BaselinePath = %q, want driver.DefaultBaselinePath %q",
				opts.BaselinePath, driver.DefaultBaselinePath)
		}
	})

	t.Run("explicit baseline passes through", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{BaselinePath: "custom.json"}, nil, "test")

		if opts.BaselinePath != "custom.json" {
			t.Errorf("BaselinePath = %q, want custom.json", opts.BaselinePath)
		}
	})

	t.Run("no patterns defaults to ./...", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{}, nil, "test")

		if !reflect.DeepEqual(opts.Patterns, []string{"./..."}) {
			t.Errorf("Patterns = %v, want [./...]", opts.Patterns)
		}
	})

	t.Run("explicit patterns pass through verbatim", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{}, []string{"./pkg/...", "./cmd/..."}, "test")

		if !reflect.DeepEqual(opts.Patterns, []string{"./pkg/...", "./cmd/..."}) {
			t.Errorf("Patterns = %v, want verbatim passthrough", opts.Patterns)
		}
	})

	t.Run("empty --output stays plain text", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{}, nil, "test")

		if opts.OutputFormat != "" {
			t.Errorf("OutputFormat = %q, want empty (plain text default)", opts.OutputFormat)
		}
	})

	t.Run("invalid --output errors before any scan", func(t *testing.T) {
		t.Parallel()

		if _, err := buildDriverOptions(&linterFlags{OutputFormat: "bogus"}, nil, "test"); err == nil {
			t.Fatal("invalid --output must error")
		}
	})

	t.Run("min-confidence maps to finding.Confidence", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{MinConfidence: 0.5}, nil, "test")

		if opts.MinConfidence != finding.Confidence(0.5) {
			t.Errorf("MinConfidence = %v, want 0.5", opts.MinConfidence)
		}
	})

	t.Run("version reaches the driver", func(t *testing.T) {
		t.Parallel()

		opts := mustBuildOptions(t, &linterFlags{}, nil, "vtest.1")

		if opts.Version != "vtest.1" {
			t.Errorf("Version = %q, want vtest.1", opts.Version)
		}
	})
}
