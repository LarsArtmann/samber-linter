package sdk_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/pkg/sdk"
)

// writeDoStub materializes the offline samber/do v2 stub (via replace) so
// loading works without the module proxy.
func writeDoStub(t *testing.T, dir string) {
	t.Helper()

	stub := filepath.Join(dir, "dostub")
	must(t, os.MkdirAll(stub, 0o755))
	must(t, os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644))
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644))
}

// e2eModule writes a self-contained module in dir: one HW-1 site (Shutdowner
// without Healthchecker), one honest service, one clean handler, and a local
// samber/do v2 stub (via replace) so loading works offline.
func e2eModule(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeDoStub(t, dir)

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
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(mainGo), 0o644))

	return app
}

// hw7CrossModule writes a two-package module: lib declares the services,
// main registers them. HW-7's evidence (a `return nil` check body) lives in
// lib and must cross the package boundary as an object fact — the invariant
// the driver's two-sweep exists to guarantee.
func hw7CrossModule(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeDoStub(t, dir)

	app := filepath.Join(dir, "app")
	must(t, os.MkdirAll(filepath.Join(app, "lib"), 0o755))
	goMod := "module example.com/app\n\ngo 1.26\n\n" +
		"require github.com/samber/do/v2 v2.1.0\n\n" +
		"replace github.com/samber/do/v2 => ../dostub\n"
	must(t, os.WriteFile(filepath.Join(app, "go.mod"), []byte(goMod), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "lib", "lib.go"), []byte(`package lib

import "context"

type ForeignNil struct{}

func (f *ForeignNil) HealthCheck(context.Context) error { return nil }

type ForeignReal struct{}

func (f *ForeignReal) HealthCheck(context.Context) error { return f.probe() }

func (f *ForeignReal) probe() error { return nil }
`), 0o644))
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(`package main

import (
	"example.com/app/lib"

	do "github.com/samber/do/v2"
)

func main() {
	do.ProvideValue(nil, &lib.ForeignNil{})  // HW-7, evidence in another package
	do.ProvideValue(nil, &lib.ForeignReal{}) // clean
}
`), 0o644))

	return app
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func TestAnalyze_FindsHW1(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	findings, err := sdk.Analyze(context.Background(), app)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	var hw1 []finding.Finding

	for _, f := range findings {
		if f.Rule == "HW-1" {
			hw1 = append(hw1, f)
		}
	}

	if len(hw1) != 1 {
		t.Fatalf("HW-1 findings = %d, want 1; all rules: %v", len(hw1), rules(findings))
	}

	if hw1[0].Severity != finding.SeverityWarning {
		t.Errorf("HW-1 severity = %q, want %q", hw1[0].Severity, finding.SeverityWarning)
	}
}

func TestAnalyze_CleanModuleReturnsNoFindings(t *testing.T) {
	t.Parallel()

	app := e2eModule(t)

	findings, err := sdk.Analyze(context.Background(), app)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	for _, f := range findings {
		if f.Rule != "HW-1" {
			t.Errorf("unexpected finding %q on the HW-1-only fixture: %s", f.Rule, f.Message)
		}
	}
}

func TestAnalyze_HW7CrossPackage(t *testing.T) {
	t.Parallel()

	app := hw7CrossModule(t)

	findings, err := sdk.Analyze(context.Background(), app)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	var hw7 []finding.Finding

	for _, f := range findings {
		if f.Rule == "HW-7" {
			hw7 = append(hw7, f)
		}
	}

	if len(hw7) != 1 {
		t.Fatalf("HW-7 findings = %d, want 1 (declared in lib, registered in main); all rules: %v", len(hw7), rules(findings))
	}

	if hw7[0].Severity != finding.SeverityWarning {
		t.Errorf("HW-7 severity = %q, want %q", hw7[0].Severity, finding.SeverityWarning)
	}
}

func TestAnalyze_LoadFailureIsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/broken\n\ngo 1.26\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "broken.go"),
		[]byte("package main\n\nfunc broken() {\n\tthis is not go source\n}\n"), 0o644))

	findings, err := sdk.Analyze(context.Background(), dir)
	if err == nil {
		t.Fatalf("Analyze on a broken module returned (%d findings, nil error); want a load error", len(findings))
	}

	if len(findings) != 0 {
		t.Errorf("findings = %d on a load failure, want 0", len(findings))
	}
}

func rules(findings []finding.Finding) []finding.RuleName {
	out := make([]finding.RuleName, 0, len(findings))

	for _, f := range findings {
		out = append(out, f.Rule)
	}

	return out
}
