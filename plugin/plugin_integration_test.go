package plugin_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
)

// TestPluginRegisteredWithGolangciLint verifies that the init() in plugin.go
// registers the healthwash linter with plugin-module-register — the same
// lookup golangci-lint v2 performs when discovering module plugins. If this
// passes, a custom-gcl binary built via `golangci-lint custom` will discover
// healthwash when the target .golangci.yml registers it under
// linters.settings.custom.
func TestPluginRegisteredWithGolangciLint(t *testing.T) {
	t.Parallel()

	constructor, err := register.GetPlugin("healthwash")
	if err != nil {
		t.Fatalf("healthwash not registered with plugin-module-register: %v", err)
	}

	plugin, err := constructor(nil)
	if err != nil {
		t.Fatalf("constructor(nil) failed: %v", err)
	}

	if plugin.GetLoadMode() != register.LoadModeSyntax {
		t.Errorf("load mode = %q, want %q", plugin.GetLoadMode(), register.LoadModeSyntax)
	}

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers failed: %v", err)
	}

	if len(analyzers) != 1 {
		t.Fatalf("analyzers = %d, want 1", len(analyzers))
	}

	if analyzers[0].Name != "healthwash" {
		t.Errorf("analyzer name = %q, want healthwash", analyzers[0].Name)
	}

	if analyzers[0].Run == nil {
		t.Error("analyzer Run is nil")
	}
}

// TestPluginSettingsPassThrough verifies the strict and disable settings from
// .golangci.yml reach the analyzer flags.
func TestPluginSettingsPassThrough(t *testing.T) {
	t.Parallel()

	constructor, err := register.GetPlugin("healthwash")
	if err != nil {
		t.Fatalf("healthwash not registered: %v", err)
	}

	plugin, err := constructor(map[string]any{
		"strict":  "true",
		"disable": "HW-4",
	})
	if err != nil {
		t.Fatalf("constructor with settings failed: %v", err)
	}

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers failed: %v", err)
	}

	if len(analyzers) != 1 {
		t.Fatalf("analyzers = %d, want 1", len(analyzers))
	}

	if f := analyzers[0].Flags.Lookup("strict"); f == nil || f.Value.String() != "true" {
		t.Errorf("strict flag not set from plugin settings")
	}

	if f := analyzers[0].Flags.Lookup("disable"); f == nil || f.Value.String() != "HW-4" {
		t.Errorf("disable flag not set from plugin settings")
	}
}

// TestCustomGCLIntegration builds the custom-gcl binary and runs it against a
// module with a known HW-1 site, proving the full plugin path end to end.
// Requires network (golangci-lint custom clones the golangci-lint source);
// skipped in short mode and when golangci-lint is absent.
func TestCustomGCLIntegration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping custom-gcl integration test in short mode (requires network + git clone)")
	}

	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not found in PATH")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	projectRoot := filepath.Dir(wd)

	t.Log("building custom-gcl binary (this may take a while)...")

	buildCmd := exec.CommandContext(t.Context(), "golangci-lint", "custom")
	buildCmd.Dir = projectRoot

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("golangci-lint custom failed: %v\n%s", err, output)
	}

	customGCL := filepath.Join(projectRoot, "custom-gcl")

	if _, err := os.Stat(customGCL); err != nil {
		t.Fatalf("custom-gcl binary not found after build: %v", err)
	}

	t.Cleanup(func() { _ = os.Remove(customGCL) })

	app := t.TempDir()
	stub := filepath.Join(app, "dostub")

	src, err := os.ReadFile(filepath.Join(projectRoot, "testdata", "src", "github.com", "samber", "do", "v2", "do.go"))
	if err != nil {
		t.Fatalf("read do stub: %v", err)
	}

	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(stub, "go.mod"),
		[]byte("module github.com/samber/do/v2\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(stub, "do.go"), src, 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(app, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(target, "go.mod"),
		[]byte(
			"module example.com/target\n\ngo 1.26\n\n"+
				"require github.com/samber/do/v2 v2.1.0\n\n"+
				"replace github.com/samber/do/v2 => ../dostub\n",
		),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(target, "main.go"), []byte(`package main

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
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// The registration section is CRITICAL: without linters.settings.custom,
	// golangci-lint reports "unknown linters: healthwash" even though the
	// plugin is compiled into the binary (verified 2026-09-10).
	if err := os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte(`version: "2"
linters:
  enable:
    - healthwash
  settings:
    custom:
      healthwash:
        type: module
        description: "Detect health-washing in samber/do v2 containers"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	runCmd := exec.CommandContext(t.Context(), customGCL, "run", "./...")
	runCmd.Dir = target

	out, err := runCmd.CombinedOutput()
	// golangci-lint exits 1 when it finds issues — which is the success case
	// here; the HW-1 in the output is the assertion.
	if err != nil && !strings.Contains(string(out), "HW-1") {
		t.Fatalf("custom-gcl run failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "HW-1") {
		t.Errorf("custom-gcl output missing HW-1 diagnostic:\n%s", out)
	}
}
