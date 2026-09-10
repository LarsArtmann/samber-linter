package driver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// e2eModule writes a self-contained module in dir: an app with one HW-1 site,
// one honest checker, one clean handler, and a local samber/do v2 stub (via
// replace) so loading works offline. Coverage: 1/3 checked.
func e2eModule(t *testing.T) string {
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
	mainGo := `package main

import (
	"context"

	do "github.com/samber/do/v2"
)

// Store: Shutdowner without Healthchecker -> HW-1.
type Store struct{}

func (s *Store) Shutdown(context.Context) {}

func NewStore(i do.Injector) (*Store, error) { return &Store{}, nil }

// Honest: both interfaces -> clean.
type Honest struct{}

func (h *Honest) Shutdown(context.Context)          {}
func (h *Honest) HealthCheck(context.Context) error { return nil }

// Handler: nothing -> clean.
type Handler struct{}

func NewHandler(i do.Injector) (*Handler, error) { return &Handler{}, nil }

func main() {
	do.Provide(nil, NewStore) // HW-1
	do.ProvideValue(nil, &Honest{})
	do.Provide(nil, NewHandler)
}
`
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(mainGo), 0o644))
	return app
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunEndToEnd(t *testing.T) {
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
	o := out.String()
	if !strings.Contains(o, "HW-1") {
		t.Errorf("output missing HW-1 finding:\n%s", o)
	}
	if !strings.Contains(o, "health-coverage: 1/3 = 33%") {
		t.Errorf("output missing coverage line 1/3:\n%s", o)
	}
}

// TestSetBaselineAndRatchet: --set-baseline writes the floor atomically; a
// committed higher floor fails on regression.
func TestSetBaselineAndRatchet(t *testing.T) {
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

	// Committed higher floor -> regression gate fails (exit 1).
	b.Coverage = 0.9
	higher, _ := json.MarshalIndent(b, "", "  ")
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

// TestCoverageMinGate: --coverage-min fails below threshold.
func TestCoverageMinGate(t *testing.T) {
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

// TestJSONAndSARIF: machine outputs are produced on demand.
func TestJSONAndSARIF(t *testing.T) {
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
	var parsed map[string]interface{}
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
		[]byte("module example.com/clean\n\ngo 1.26\n\nrequire github.com/samber/do/v2 v2.1.0\n\nreplace github.com/samber/do/v2 => ../dostub\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(`package main

import (
	"context"

	do "github.com/samber/do/v2"
)

type DB struct{}

func (d *DB) Shutdown(context.Context)          {}
func (d *DB) HealthCheck(context.Context) error { return nil }

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

func lastJSONLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "{") {
			return lines[i]
		}
	}
	return s
}
