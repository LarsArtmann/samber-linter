package main

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// TestBuildDriverOptions pins the flag-struct-to-driver.Options mapping: the
// two defaults the flag tags cannot carry (pattern fallback, baseline path
// fallback) and the pass-through of every explicit value. A drift here changes
// CLI behavior without touching the flag surface tests.
func TestBuildDriverOptions(t *testing.T) {
	t.Parallel()

	t.Run("empty baseline falls back to the driver default", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{}, nil, "test")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if opts.BaselinePath != driver.DefaultBaselinePath {
			t.Errorf("BaselinePath = %q, want driver.DefaultBaselinePath %q",
				opts.BaselinePath, driver.DefaultBaselinePath)
		}
	})

	t.Run("explicit baseline passes through", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{BaselinePath: "custom.json"}, nil, "test")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if opts.BaselinePath != "custom.json" {
			t.Errorf("BaselinePath = %q, want custom.json", opts.BaselinePath)
		}
	})

	t.Run("no patterns defaults to ./...", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{}, nil, "test")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if !reflect.DeepEqual(opts.Patterns, []string{"./..."}) {
			t.Errorf("Patterns = %v, want [./...]", opts.Patterns)
		}
	})

	t.Run("explicit patterns pass through verbatim", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{}, []string{"./pkg/...", "./cmd/..."}, "test")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if !reflect.DeepEqual(opts.Patterns, []string{"./pkg/...", "./cmd/..."}) {
			t.Errorf("Patterns = %v, want verbatim passthrough", opts.Patterns)
		}
	})

	t.Run("empty --output stays plain text", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{}, nil, "test")
		if err != nil {
			t.Fatalf("empty --output must parse (plain default), got %v", err)
		}

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

		opts, err := buildDriverOptions(&linterFlags{MinConfidence: 0.5}, nil, "test")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if opts.MinConfidence != finding.Confidence(0.5) {
			t.Errorf("MinConfidence = %v, want 0.5", opts.MinConfidence)
		}
	})

	t.Run("version reaches the driver", func(t *testing.T) {
		t.Parallel()

		opts, err := buildDriverOptions(&linterFlags{}, nil, "vtest.1")
		if err != nil {
			t.Fatalf("buildDriverOptions: %v", err)
		}

		if opts.Version != "vtest.1" {
			t.Errorf("Version = %q, want vtest.1", opts.Version)
		}
	})
}
