package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v4 "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
)

// The exit-code contract in one table: this harness executes the REAL CLI
// wiring (newCLI, not a copy) against self-contained scaffold modules, so a
// regression in flag parsing, help handling, or exit-code mapping fails here
// before it ships. It replaces the manual smoke loop from the cmdguard
// adoption session (status 2026-10-02 11:42).

// captureStd swaps os.Stdout/os.Stderr for pipes and returns their buffers,
// restored via t.Cleanup. The CLI and the driver write to the process fds
// directly, so fd-level capture is the only faithful vantage point. Tests
// using it must not run in parallel.
func captureStd(t *testing.T) (stdout, stderr *bytes.Buffer) {
	t.Helper()

	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}

	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	os.Stdout, os.Stderr = wOut, wErr

	doneOut := make(chan struct{})
	go func() {
		_, _ = io.Copy(stdout, rOut)
		close(doneOut)
	}()

	doneErr := make(chan struct{})
	go func() {
		_, _ = io.Copy(stderr, rErr)
		close(doneErr)
	}()

	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		_ = wOut.Close()
		_ = wErr.Close()
		<-doneOut
		<-doneErr
	})

	return stdout, stderr
}

// writePlainModule scaffolds a standalone module (valid go.mod + main.go),
// for scans that need no samber/do dependency.
func writePlainModule(t *testing.T, mainGo string) string {
	t.Helper()

	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/app\n\ngo 1.26\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644))

	return dir
}

// writeConsumerModule scaffolds a module with one HW-1 registration site and
// a local samber/do v2 stub via replace, so loading works offline (the same
// scaffold internal/driver's e2e tests use).
func writeConsumerModule(t *testing.T) string {
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

// Honest: both interfaces, a body that can fail -> clean.
type Honest struct{}

func (h *Honest) Shutdown(context.Context) {}

func (h *Honest) HealthCheck(context.Context) error { return h.probe() }

func (h *Honest) probe() error { return nil }

func main() {
	do.Provide(nil, NewStore) // HW-1
	do.ProvideValue(nil, &Honest{})
}
`
	must(t, os.WriteFile(filepath.Join(app, "main.go"), []byte(mainGo), 0o644))

	return app
}

// writeBrokenModule scaffolds a module whose go.mod cannot parse: the load
// fails before analysis, which is the documented exit-2 path.
func writeBrokenModule(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("mod ule example.com/broken\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644))

	return dir
}

func TestExitCodeContract(t *testing.T) {
	cleanModule := func(t *testing.T) string {
		return writePlainModule(t, "package main\n\nfunc main() {}\n")
	}

	cases := []struct {
		name    string
		args    []string
		module  func(*testing.T) string
		wantMsg string // substring asserted against combined output
	}{
		// Help: every spelling, including after positional patterns (the
		// webphone regression), must print help and exit 0.
		{name: "help long", args: []string{"--help"}, module: cleanModule, wantMsg: "samber-linter"},
		{name: "help short", args: []string{"-h"}, module: cleanModule, wantMsg: "samber-linter"},
		{name: "help single-dash-long", args: []string{"-help"}, module: cleanModule, wantMsg: "samber-linter"},
		{name: "help trailing after pattern", args: []string{"./...", "--help"}, module: cleanModule, wantMsg: "samber-linter"},
		// Cobra auto-adds help/completion subcommands; they must stay
		// invocable (and never reach the linter) — ArbitraryArgs keeps them
		// from swallowing positional package patterns.
		{name: "help subcommand", args: []string{"help"}, module: cleanModule, wantMsg: "samber-linter"},
		{name: "completion bash", args: []string{"completion", "bash"}, module: cleanModule},
		{name: "version", args: []string{"--version"}, module: cleanModule, wantMsg: "test-version"},
		// Flag misuse: cobra convention exits 1 (stdlib flag exited 2 — the
		// one documented delta of the cmdguard adoption).
		{name: "unknown flag exits 1", args: []string{"--bogus"}, module: cleanModule},
		// Invalid --output keeps the documented tri-state: exit 2, no scan.
		{name: "invalid output exits 2", args: []string{"--output", "bogus", "./..."}, module: cleanModule},
		// The driver tri-state, through the real wiring.
		{name: "clean scan exits 0", args: []string{"./..."}, module: cleanModule},
		{name: "findings exit 1", args: []string{"./..."}, module: writeConsumerModule, wantMsg: "HW-1"},
		{name: "check forces 0 on findings", args: []string{"--check", "./..."}, module: writeConsumerModule},
		{name: "load failure exits 2", args: []string{"./..."}, module: writeBrokenModule},
		{name: "check forces 0 on load failure", args: []string{"--check", "./..."}, module: writeBrokenModule},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(tc.module(t))

			version = "test-version"
			t.Cleanup(func() { version = "" })

			stdout, stderr := captureStd(t)

			cli, err := newCLI()
			if err != nil {
				t.Fatalf("building CLI: %v", err)
			}

			execErr := cli.ExecuteWithArgs(context.Background(), tc.args)
			if code := v4.ExitCode(execErr); code < 0 || code > 255 {
				t.Fatalf("v4.ExitCode(%v) = %d, want a valid exit code", execErr, code)
			}

			if code := v4.ExitCode(execErr); code != exitCodeFor(tc.name) {
				t.Errorf("exit code = %d, want %d (stdout:\n%s\nstderr:\n%s)",
					code, exitCodeFor(tc.name), stdout.String(), stderr.String())
			}

			if tc.wantMsg != "" {
				combined := stdout.String() + stderr.String()
				if !strings.Contains(combined, tc.wantMsg) {
					t.Errorf("output missing %q (stdout:\n%s\nstderr:\n%s)",
						tc.wantMsg, stdout.String(), stderr.String())
				}
			}
		})
	}
}

// exitCodeFor is the expected-code table, kept beside the cases so a mismatch
// reads as one line per contract point.
func exitCodeFor(name string) int {
	switch name {
	case "unknown flag exits 1":
		return 1
	case "findings exit 1":
		return 1
	case "invalid output exits 2":
		return 2
	case "load failure exits 2":
		return 2
	default:
		return 0
	}
}
