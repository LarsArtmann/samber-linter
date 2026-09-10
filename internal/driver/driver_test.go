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

	// Committed higher floor -> regression gate fails (exit 1).
	b.Coverage = 0.9
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
			"module example.com/clean\n\ngo 1.26\n\n" +
				"require github.com/samber/do/v2 v2.1.0\n\n" +
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

func lastJSONLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range slices.Backward(lines) {
		if strings.HasPrefix(line, "{") {
			return line
		}
	}

	return text
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
