package driver

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// writeConsumerModule scaffolds a self-contained consumer module under dir:
// an "app" directory with the given main.go and a local samber/do v2 stub
// (via replace) so loading works offline. Returns the app path.
func writeConsumerModule(t *testing.T, mainGo string) string {
	t.Helper()
	dir := t.TempDir()

	stub := filepath.Join(dir, "dostub")
	must(t, os.MkdirAll(stub, 0o755))
	must(t, os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644))
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644))

	app := filepath.Join(dir, "app")
	must(t, os.MkdirAll(app, 0o755))

	goMod := "module example.com/app\n\ngo 1.26\n\n" +
		"require github.com/samber/do/v2 v2.1.0\n\n" +
		"replace github.com/samber/do/v2 => ../dostub\n"
	must(t, os.WriteFile(filepath.Join(app, "go.mod"), []byte(goMod), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(mainGo), 0o644))

	return app
}

// e2eModule writes a self-contained module: an app with one HW-1 site,
// one honest checker, one clean handler, and a local samber/do v2 stub (via
// replace) so loading works offline. Coverage: 1/3 checked.
func e2eModule(t *testing.T) string {
	t.Helper()

	mainGo := `package main

import (
	"context"

	do "github.com/samber/do/v2"
)

// Store: Shutdowner without Healthchecker -> HW-1.
type Store struct{}

func (s *Store) Shutdown(context.Context) {}

func NewStore(i do.Injector) (*Store, error) { return &Store{}, nil }

// Honest: both interfaces, a body that can fail -> clean.
type Honest struct{}

func (h *Honest) Shutdown(context.Context) {}

func (h *Honest) HealthCheck(context.Context) error { return h.probe() }

func (h *Honest) probe() error { return nil }

// Handler: nothing -> clean.
type Handler struct{}

func NewHandler(i do.Injector) (*Handler, error) { return &Handler{}, nil }

func main() {
	do.Provide(nil, NewStore) // HW-1
	do.ProvideValue(nil, &Honest{})
	do.Provide(nil, NewHandler)
}
`

	return writeConsumerModule(t, mainGo)
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func TestRunEndToEnd(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns:      []string{"./..."},
		Dir:           app,
		Env:           []string{"GOFLAGS=-mod=mod"},
		Version:       "test",
		Stdout:        &out,
		Stderr:        &errOut,
		CoverageMin:   -1,
		MinConfidence: finding.ConfidenceHigh,
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (high-confidence HW-1); stderr: %s", code, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "HW-1") {
		t.Errorf("output missing HW-1 finding:\n%s", output)
	}

	if !strings.Contains(output, "health-coverage: 1/3 = 33%") {
		t.Errorf("output missing coverage line 1/3:\n%s", output)
	}
}

// TestSetBaselineAndRatchet: --set-baseline writes the floor atomically; a
// committed higher floor fails on regression.
func TestSetBaselineAndRatchet(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)
	baselinePath := filepath.Join(app, DefaultBaselinePath)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns:     []string{"./..."},
		Dir:          app,
		Env:          []string{"GOFLAGS=-mod=mod"},
		Version:      "test",
		Stdout:       &out,
		Stderr:       &errOut,
		SetBaseline:  true,
		BaselinePath: baselinePath,
	})
	if code != 1 {
		t.Fatalf("set-baseline run exit = %d, want 1 (HW-1 finding still gates); stderr: %s", code, errOut.String())
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("baseline not written: %v", err)
	}

	var b baseline
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("baseline parse: %v", err)
	}

	if b.Coverage <= 0 || b.Coverage >= 1 {
		t.Errorf("baseline coverage = %v, want in (0,1)", b.Coverage)
	}

	// Committed higher floor -> regression gate fails (exit 1). The floor must
	// stay internally consistent (coverage == checked/registered) or
	// validateBaseline rejects it as corrupt before the ratchet is applied.
	b.Coverage = 0.9
	b.Checked, b.Registered = 9, 10
	higher, _ := json.Marshal(b, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	must(t, os.WriteFile(baselinePath, append(higher, '\n'), 0o644))

	out.Reset()
	errOut.Reset()

	code = Run(Options{
		Patterns:     []string{"./..."},
		Dir:          app,
		Env:          []string{"GOFLAGS=-mod=mod"},
		Version:      "test",
		Stdout:       &out,
		Stderr:       &errOut,
		BaselinePath: baselinePath,
	})
	if code != 1 {
		t.Fatalf("regressed coverage exit = %d, want 1; out: %s err: %s", code, out.String(), errOut.String())
	}

	if !strings.Contains(errOut.String(), "below the committed baseline") {
		t.Errorf("stderr missing regression message: %s", errOut.String())
	}
}

// hw4Module writes a module whose only findings are two HW-4 sites: Medium
// confidence, below the default --min-confidence, so findings alone exit 0
// and the baseline ratchet is the only gate that can fail the run.
func hw4Module(t *testing.T) string {
	t.Helper()

	mainGo := `package main

import (
	"context"

	do "github.com/samber/do/v2"
)

// Lazy and Other are healthcheckers registered without an eager option, so
// each site is HW-4 (lazy + check) — advisory, never a confidence-gate exit.
// Their bodies delegate (can fail): body shape is HW-7's contract, and a
// Full-confidence finding here would break the triage-only exit this module
// exists to pin.
type Lazy struct{}

func (l *Lazy) HealthCheck(context.Context) error { return l.probe() }

func (l *Lazy) probe() error { return nil }

func NewLazy(i do.Injector) (*Lazy, error) { return &Lazy{}, nil }

type Other struct{}

func (o *Other) HealthCheck(context.Context) error { return o.probe() }

func (o *Other) probe() error { return nil }

func NewOther(i do.Injector) (*Other, error) { return &Other{}, nil }

func main() {
	do.Provide(nil, NewLazy)
	do.Provide(nil, NewOther)
}
`

	return writeConsumerModule(t, mainGo)
}

// TestBaselineV2PerRuleRatchet: --set-baseline records per-rule finding
// counts; any rule exceeding its committed count fails the gate even while
// aggregate coverage stays flat.
func TestBaselineV2PerRuleRatchet(t *testing.T) {
	t.Parallel()

	app := hw4Module(t)
	baselinePath := filepath.Join(app, DefaultBaselinePath)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		SetBaseline:  true,
		BaselinePath: baselinePath,
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 2 {
		t.Fatalf("set-baseline exit = %d, want 2 (HW-4 findings are triage-only); stderr: %s", code, errOut.String())
	}

	if strings.Contains(errOut.String(), "baseline") && !strings.Contains(errOut.String(), "no baseline") {
		t.Errorf("set-baseline run reported a baseline error: %s", errOut.String())
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("baseline not written: %v", err)
	}

	var b baseline
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("baseline parse: %v", err)
	}

	if b.Version != baselineSchemaVersion {
		t.Errorf("baseline version = %d, want %d", b.Version, baselineSchemaVersion)
	}

	if b.Findings["HW-4"] != 2 {
		t.Errorf("baseline findings[HW-4] = %d, want 2", b.Findings["HW-4"])
	}

	run := func() (int, string) {
		var out, errOut bytes.Buffer

		code := Run(Options{
			Patterns: []string{"./..."}, Dir: app, Version: "test",
			Env:          []string{"GOFLAGS=-mod=mod"},
			BaselinePath: baselinePath,
			Stdout:       &out, Stderr: &errOut,
		})

		return code, errOut.String()
	}

	// A baseline counting one site while the scan sees two regresses the rule
	// floor while coverage stays flat — exactly what v1 could not catch.
	regressed, _ := json.Marshal(baseline{
		Version: 2, Checked: 2, Registered: 2, Coverage: 1,
		Findings: map[string]int{"HW-4": 1},
	}, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	must(t, os.WriteFile(baselinePath, append(regressed, '\n'), 0o644))

	code, errMsg := run()
	if code != 1 {
		t.Fatalf("rule regression exit = %d, want 1; stderr: %s", code, errMsg)
	}

	if !strings.Contains(errMsg, "HW-4 findings 2 exceed the committed baseline 1") {
		t.Errorf("stderr missing per-rule regression message: %s", errMsg)
	}

	// The floor at the current count passes clean.
	locked, _ := json.Marshal(baseline{
		Version: 2, Checked: 2, Registered: 2, Coverage: 1,
		Findings: map[string]int{"HW-4": 2},
	}, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	must(t, os.WriteFile(baselinePath, append(locked, '\n'), 0o644))

	code, errMsg = run()
	if code != 2 {
		t.Fatalf("locked floor exit = %d, want 2 (triage-only, no gate failure); stderr: %s", code, errMsg)
	}

	if strings.Contains(errMsg, "exceed the committed baseline") {
		t.Errorf("locked floor must not report regressions: %s", errMsg)
	}
}

// TestBaselineV2Validation: baseline files this run cannot enforce honestly
// fail the gate loudly instead of silently degrading to no-baseline.
func TestBaselineV2Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name:    "v1 baseline names the migration",
			file:    `{"version":1,"checked":2,"registered":2,"coverage":1}`,
			wantErr: "predates v2",
		},
		{
			name:    "future schema fails loudly",
			file:    `{"version":3,"checked":2,"registered":2,"coverage":1}`,
			wantErr: "newer than this tool supports",
		},
		{
			name:    "coverage inconsistent with counters is corrupt",
			file:    `{"version":2,"checked":1,"registered":2,"coverage":0.9}`,
			wantErr: "does not match checked/registered",
		},
		{
			name:    "coverage outside range is corrupt",
			file:    `{"version":2,"checked":2,"registered":2,"coverage":1.5}`,
			wantErr: "outside [0,1]",
		},
		{
			name:    "checked above registered is corrupt",
			file:    `{"version":2,"checked":3,"registered":2,"coverage":1.5}`,
			wantErr: "exceeds registered",
		},
		{
			name:    "negative finding count is corrupt",
			file:    `{"version":2,"checked":2,"registered":2,"coverage":1,"findings":{"HW-4":-1}}`,
			wantErr: "is negative",
		},
		{
			name:    "unparseable JSON fails closed",
			file:    `{"version":2,`,
			wantErr: "is not valid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := hw4Module(t)
			baselinePath := filepath.Join(app, DefaultBaselinePath)
			must(t, os.WriteFile(baselinePath, []byte(tt.file), 0o644))

			var out, errOut bytes.Buffer

			code := Run(Options{
				Patterns: []string{"./..."}, Dir: app, Version: "test",
				Env:          []string{"GOFLAGS=-mod=mod"},
				BaselinePath: baselinePath,
				Stdout:       &out, Stderr: &errOut,
			})
			if code != 1 {
				t.Fatalf("invalid baseline exit = %d, want 1 (fail closed); stderr: %s", code, errOut.String())
			}

			if !strings.Contains(errOut.String(), tt.wantErr) {
				t.Errorf("stderr missing %q; got: %s", tt.wantErr, errOut.String())
			}
		})
	}
}

// TestCoverageMinGate: --coverage-min fails below threshold.
func TestCoverageMinGate(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns:    []string{"./..."},
		Dir:         app,
		Version:     "test",
		Stdout:      &out,
		Stderr:      &errOut,
		CoverageMin: 0.5,
	})
	if code != 1 {
		t.Fatalf("coverage-min gate exit = %d, want 1; out: %s err: %s", code, out.String(), errOut.String())
	}
}

// TestCoverageMinComposesWithBaseline: passing --coverage-min must not
// short-circuit baseline validation or the per-rule ratchet — the gates
// compose. Discrimination: both scenarios below exit 2 (triage-only) on the
// old early-return shape, where the baseline file was never read.
func TestCoverageMinComposesWithBaseline(t *testing.T) {
	t.Parallel()

	app := hw4Module(t)
	baselinePath := filepath.Join(app, DefaultBaselinePath)

	var out, errOut bytes.Buffer

	// The absolute floor passes (2/2 = 100% >= 50%), but a v1 baseline is
	// corrupt by generation: validation must fail the gate anyway.
	must(t, os.WriteFile(baselinePath,
		[]byte(`{"version":1,"checked":2,"registered":2,"coverage":1}`), 0o644))

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		CoverageMin:  0.5,
		BaselinePath: baselinePath,
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 1 {
		t.Fatalf("coverage-min + v1 baseline exit = %d, want 1 (baseline must still be validated); stderr: %s",
			code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "predates v2") {
		t.Errorf("stderr missing v1 migration message: %s", errOut.String())
	}

	// The rule floor is part of the committed baseline too: one committed
	// HW-4 site against two scanned regresses the gate even under
	// coverage-min with flat coverage.
	regressed, _ := json.Marshal(baseline{
		Version: 2, Checked: 2, Registered: 2, Coverage: 1,
		Findings: map[string]int{"HW-4": 1},
	}, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	must(t, os.WriteFile(baselinePath, append(regressed, '\n'), 0o644))

	out.Reset()
	errOut.Reset()

	code = Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		CoverageMin:  0.5,
		BaselinePath: baselinePath,
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 1 {
		t.Fatalf("coverage-min + rule regression exit = %d, want 1; stderr: %s", code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "HW-4 findings 2 exceed the committed baseline 1") {
		t.Errorf("stderr missing per-rule regression message: %s", errOut.String())
	}
}

// TestCheckForcesZeroThroughFailedGates: --check is advisory in EVERY case —
// even when both gates fail (impossible coverage floor + corrupt baseline).
// The gate diagnostics stay on stderr, so machine output on stdout remains
// parseable. On the old shape (advisory forcing only finding-tier codes, or
// gates skipping baseline validation) this exits 1 or 0-with-garbage.
func TestCheckForcesZeroThroughFailedGates(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)
	baselinePath := filepath.Join(app, DefaultBaselinePath)
	must(t, os.WriteFile(baselinePath,
		[]byte(`{"version":1,"checked":1,"registered":3,"coverage":0.5}`), 0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		CoverageMin:  0.99,
		BaselinePath: baselinePath,
		Check:        true,
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("check + failed gates exit = %d, want 0 (advisory in every case); stderr: %s",
			code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "predates v2") {
		t.Errorf("stderr must still diagnose the corrupt baseline: %s", errOut.String())
	}

	if !strings.Contains(errOut.String(), "below the required minimum") {
		t.Errorf("stderr must still diagnose the coverage miss: %s", errOut.String())
	}

	out.Reset()
	errOut.Reset()

	code = Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		CoverageMin:  0.99,
		BaselinePath: baselinePath,
		Check:        true,
		JSON:         true,
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("check + json + failed gates exit = %d, want 0; stderr: %s", code, errOut.String())
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(lastJSONLine(out.String()))), &parsed); err != nil {
		t.Fatalf("json output corrupted by gate lines: %v\n%s", err, out.String())
	}
}

// TestJSONAndSARIF: machine outputs are produced on demand.
func TestJSONAndSARIF(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Stdout: &out, Stderr: &errOut, JSON: true,
		MinConfidence: finding.ConfidenceHigh,
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(lastJSONLine(out.String()))), &parsed); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, out.String())
	}

	out.Reset()

	code = Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Stdout: &out, Stderr: &errOut, SARIF: true,
		MinConfidence: finding.ConfidenceHigh,
	})
	if code != 1 {
		t.Fatalf("sarif exit = %d, want 1", code)
	}

	if !strings.Contains(out.String(), "sarif") {
		t.Errorf("sarif output unexpected")
	}
}

// TestCleanModuleExitsZero: an honest module exits 0 with no findings.
func TestCleanModuleExitsZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stub := filepath.Join(dir, "dostub")
	must(t, os.MkdirAll(stub, 0o755))
	must(t, os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644))
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644))

	app := filepath.Join(dir, "app")
	must(t, os.MkdirAll(app, 0o755))
	must(t, os.WriteFile(
		filepath.Join(app, "go.mod"),
		[]byte(
			"module example.com/clean\n\ngo 1.26\n\n"+
				"require github.com/samber/do/v2 v2.1.0\n\n"+
				"replace github.com/samber/do/v2 => ../dostub\n",
		),
		0o644,
	))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(`package main

import (
	"context"

	do "github.com/samber/do/v2"
)

type DB struct{}

func (d *DB) Shutdown(context.Context) {}

func (d *DB) HealthCheck(context.Context) error { return d.probe() }

func (d *DB) probe() error { return nil }

type Cfg struct{ Addr string }

func main() {
	do.ProvideValue(nil, &DB{})
	do.ProvideValue(nil, Cfg{Addr: ":80"})
}
`), 0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Stdout: &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("clean module exit = %d, want 0; out: %s err: %s", code, out.String(), errOut.String())
	}

	if !strings.Contains(out.String(), "no health-washing found") {
		t.Errorf("expected clean summary; out:\n%s", out.String())
	}
}

func lastJSONLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range slices.Backward(lines) {
		if strings.HasPrefix(line, "{") {
			return line
		}
	}

	return text
}

// TestHW7CrossPackage: the nil-body verdict is computed in the package
// DECLARING the service and consumed at the registration site in another
// package. Discriminates the two-sweep fact store from a single-sweep
// regression: on the old shape (facts never shared across packages) the
// nil-body service renders clean and this test fails.
func TestHW7CrossPackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stub := filepath.Join(dir, "dostub")
	must(t, os.MkdirAll(stub, 0o755))
	must(t, os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644))
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644))

	app := filepath.Join(dir, "app")
	must(t, os.MkdirAll(filepath.Join(app, "services"), 0o755))
	must(t, os.WriteFile(
		filepath.Join(app, "go.mod"),
		[]byte(
			"module example.com/cross\n\ngo 1.26\n\n"+
				"require github.com/samber/do/v2 v2.1.0\n\n"+
				"replace github.com/samber/do/v2 => ../dostub\n",
		),
		0o644,
	))
	must(t, os.WriteFile(filepath.Join(app, "services", "services.go"), []byte(`package services

import "context"

type NilCheck struct{}

func (n *NilCheck) HealthCheck(context.Context) error { return nil }

type RealCheck struct{}

func (r *RealCheck) HealthCheck(context.Context) error { return r.probe() }

func (r *RealCheck) probe() error { return nil }
`), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(`package main

import (
	do "github.com/samber/do/v2"

	"example.com/cross/services"
)

func main() {
	do.ProvideValue(nil, &services.NilCheck{})
	do.ProvideValue(nil, &services.RealCheck{})
}
`), 0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Stdout: &out, Stderr: &errOut,
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (HW-7 is Full confidence); stderr: %s", code, errOut.String())
	}

	if !strings.Contains(out.String(), "HW-7") {
		t.Errorf("cross-package nil body must be reported:\n%s", out.String())
	}

	if strings.Count(out.String(), "HW-7") != 1 {
		t.Errorf("HW-7 must fire exactly once (RealCheck delegates):\n%s", out.String())
	}
}

// TestCheckAdvisoryMode: --check reports findings but always exits 0, so CI
// annotation pipelines can parse output without failing the build.
func TestCheckAdvisoryMode(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Check:  true,
		Stdout: &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("advisory exit = %d, want 0; stderr: %s", code, errOut.String())
	}

	if !strings.Contains(out.String(), "HW-1") {
		t.Errorf("advisory run must still report findings; out:\n%s", out.String())
	}

	if !strings.Contains(out.String(), "--check: advisory run; exit code forced to 0") {
		t.Errorf("advisory run must say so; out:\n%s", out.String())
	}
}

// TestCheckAdvisorySilentInMachineFormats: the --check advisory note is human
// UI. --json/--sarif and the structured --output formats are parsed by
// machines and must not carry a trailing unparseable line; only human-facing
// presentations (plain text, table, markdown) keep the note.
func TestCheckAdvisorySilentInMachineFormats(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	tests := []struct {
		name     string
		opts     Options
		wantNote bool
	}{
		{name: "plain text keeps the note", opts: Options{}, wantNote: true},
		{name: "json suppresses the note", opts: Options{JSON: true}},
		{name: "sarif suppresses the note", opts: Options{SARIF: true}},
		{name: "csv suppresses the note", opts: Options{OutputFormat: "csv"}},
		{name: "table keeps the note", opts: Options{OutputFormat: "table"}, wantNote: true},
		{name: "markdown keeps the note", opts: Options{OutputFormat: "markdown"}, wantNote: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.opts
			opts.Patterns = []string{"./..."}
			opts.Dir = app
			opts.Env = []string{"GOFLAGS=-mod=mod"}
			opts.Version = "test"
			opts.Check = true

			var out, errOut bytes.Buffer

			opts.Stdout, opts.Stderr = &out, &errOut

			if code := Run(opts); code != 0 {
				t.Fatalf("advisory exit = %d, want 0; stderr: %s", code, errOut.String())
			}

			got := strings.Contains(out.String(), "--check: advisory run; exit code forced to 0")
			if got != tt.wantNote {
				t.Errorf("advisory note present = %v, want %v; out:\n%s", got, tt.wantNote, out.String())
			}
		})
	}
}

// TestDisableRules: the CLI --disable flag mutes rules the way the analyzer
// flag does (the golangci plugin path was already covered by wiring).
func TestDisableRules(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:          []string{"GOFLAGS=-mod=mod"},
		DisableRules: "HW-1",
		Stdout:       &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("disabled HW-1 exit = %d, want 0; out: %s err: %s", code, out.String(), errOut.String())
	}

	if strings.Contains(out.String(), "HW-1") {
		t.Errorf("disabled rule still reported; out:\n%s", out.String())
	}
}

// TestAllowlistEmptyPathPatternCoversProject: an entry without pathPattern is
// the project-wide form, not a silent no-op; a rule-less entry warns and
// stays inert.
func TestAllowlistEmptyPathPatternCoversProject(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)
	cfg := filepath.Join(app, "allow.json")

	must(t, os.WriteFile(cfg, []byte(
		`{"allow":[{"rule":"HW-1","reason":"accepted: legacy store, ticket HW-1-101"}]}`),
		0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:        []string{"GOFLAGS=-mod=mod"},
		ConfigPath: cfg,
		Stdout:     &out, Stderr: &errOut,
	})
	if code != 0 {
		t.Fatalf("allowlisted HW-1 exit = %d, want 0; out: %s err: %s", code, out.String(), errOut.String())
	}

	if strings.Contains(errOut.String(), "lacks a rule") {
		t.Errorf("well-formed entry must not warn: %s", errOut.String())
	}

	// A rule-less entry is inert and warns.
	must(t, os.WriteFile(cfg, []byte(
		`{"allow":[{"pathPattern":"**","reason":"blanket entry with no rule"}]}`), 0o644))

	out.Reset()
	errOut.Reset()

	code = Run(Options{
		Patterns: []string{"./..."}, Dir: app, Version: "test",
		Env:        []string{"GOFLAGS=-mod=mod"},
		ConfigPath: cfg,
		Stdout:     &out, Stderr: &errOut,
	})
	if code != 1 {
		t.Fatalf("inert entry exit = %d, want 1 (finding survives); err: %s", code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "lacks a rule") {
		t.Errorf("rule-less entry must warn; err: %s", errOut.String())
	}
}

// TestLoadFailureExitsTwo: exit 1 is reserved for findings; a run that cannot
// load anything has none and must say 2.
func TestLoadFailureExitsTwo(t *testing.T) {
	t.Parallel()

	broken := t.TempDir()
	must(t, os.WriteFile(filepath.Join(broken, "go.mod"),
		[]byte("mod ule example.com/broken\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(broken, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		Patterns: []string{"./..."}, Dir: broken, Version: "test",
		Env:    []string{"GOFLAGS=-mod=mod"},
		Stdout: &out, Stderr: &errOut,
	})
	if code != 2 {
		t.Fatalf("load failure exit = %d, want 2; stderr: %s", code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "load failed") {
		t.Errorf("load failure must explain itself; stderr: %s", errOut.String())
	}
}

// TestGoWorkMultiModule: a go.work workspace spanning an app module and a
// local samber/do stub module (the shape real consumers like go.work-based
// monorepos present) must load and analyze across module boundaries.
func TestGoWorkMultiModule(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	stub := filepath.Join(dir, "dostub")
	must(t, os.MkdirAll(stub, 0o755))
	must(t, os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644))
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644))

	app := filepath.Join(dir, "app")
	must(t, os.MkdirAll(app, 0o755))
	must(t, os.WriteFile(filepath.Join(app, "go.mod"),
		[]byte("module example.com/app\n\ngo 1.26\n\nrequire github.com/samber/do/v2 v2.1.0\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(`package main

import (
	"context"

	do "github.com/samber/do/v2"
)

type Store struct{}

func (s *Store) Shutdown(context.Context) {}

func NewStore(i do.Injector) (*Store, error) { return &Store{}, nil }

func main() {
	do.Provide(nil, NewStore)
}
`), 0o644))

	must(t, os.WriteFile(filepath.Join(dir, "go.work"),
		[]byte("go 1.26\n\nuse (\n\t./app\n\t./dostub\n)\n"), 0o644))

	var out, errOut bytes.Buffer

	code := Run(Options{
		// In workspace mode `all` means every module listed in go.work;
		// `./...` from the workspace root errors because the root itself is
		// not a module. README documents `all` as the workspace pattern.
		Patterns: []string{"all"}, Dir: dir, Version: "test",
		// Workspace mode forbids -mod=mod; every module is local via use,
		// so readonly loading resolves everything.
		Stdout: &out, Stderr: &errOut,
	})
	if code != 1 {
		t.Fatalf("go.work exit = %d, want 1 (HW-1 in app module); out: %s err: %s",
			code, out.String(), errOut.String())
	}

	if !strings.Contains(out.String(), "HW-1") {
		t.Errorf("go.work analysis lost the HW-1 finding; out:\n%s err:\n%s", out.String(), errOut.String())
	}
}
