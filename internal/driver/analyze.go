package driver

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/larsartmann/go-finding"
	"golang.org/x/tools/go/packages"
)

// Analyze runs one analysis pass programmatically: load → analyze → allowlist.
// Unlike Run it renders no presentation output and maps no exit codes —
// in-process consumers (BuildFlow's DAG provider) own presentation, confidence
// policy, and gating. The HW-6 baseline ratchet and coverage gate stay
// CLI-only: they are release gates, not per-detection concerns.
//
// A load failure or ANY package error aborts with an error. A run that cannot
// load every package has incomplete registration facts, and reporting the
// remainder as clean would be a false green — the exact failure mode this
// linter exists to kill. A nil ctx is tolerated — x/tools defaults it.
func Analyze(ctx context.Context, opts Options) ([]finding.Finding, error) {
	if len(opts.Patterns) == 0 {
		opts.Patterns = []string{"./..."}
	}

	pkgs, err := loadWithCtx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ToolName, err)
	}

	if err := firstLoadError(pkgs); err != nil {
		return nil, err
	}

	errw := opts.Stderr
	if errw == nil {
		errw = io.Discard
	}

	findings, _ := analyzePackages(buildAnalyzer(opts), pkgs, errw)

	return applyAllowlist(findings, opts.ConfigPath, errw), nil
}

// loadWithCtx is load() with a cancellable context so a hung `go list` (dead
// cache mount, wedged toolchain) cannot outlive the caller's deadline. The
// packages.Config must stay in sync with load() in driver.go — the CLI keeps
// its own copy because that file owns the process exit contract.
func loadWithCtx(ctx context.Context, opts Options) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedDeps | packages.NeedImports |
			packages.NeedModule,
		Dir:     opts.Dir,
		Env:     append(os.Environ(), opts.Env...),
		Tests:   false, // composition roots are what dashboards see; DO-3 keeps Override* in tests
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	return pkgs, nil
}

// firstLoadError aggregates package load errors into one error carrying the
// first offender and the total count.
func firstLoadError(pkgs []*packages.Package) error {
	var (
		first error
		count int
	)

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			if first == nil {
				first = fmt.Errorf("%s: %w", pkg.PkgPath, e)
			}

			count++
		}
	}

	if first == nil {
		return nil
	}

	return fmt.Errorf(
		"%w; aborting: %d package error(s) during load — "+
			"incomplete registration facts are never reported as clean",
		first, count,
	)
}
