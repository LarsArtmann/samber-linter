// Package sdk exposes samber-linter's static analysis as a programmatic API
// for in-process consumers such as BuildFlow's DAG provider. It wraps the
// internal driver without exposing the CLI's exit-code contract: callers get
// findings or an error, never stdout to parse.
package sdk

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// goFlagsMod overrides an inherited GOFLAGS=-mod=vendor: package loading must
// resolve through the module graph because the analyzed repo may not have a
// vendor directory (and -mod=vendor fails loudly when it does not). Child
// processes deduplicate env with the LAST occurrence winning, so appending
// here shadows any inherited value.
const goFlagsMod = "GOFLAGS=-mod=mod"

// Analyze loads the Go packages under dir (pattern "./...", test files
// excluded — composition roots are what dashboards see) and returns the
// healthwash findings, with the repo's inline //samber-linter:allow
// suppressions already applied. dir "" means the process working directory.
// A load failure or any package error is returned as an error: incomplete
// results are never reported as clean.
func Analyze(ctx context.Context, dir string) ([]finding.Finding, error) {
	findings, err := driver.Analyze(ctx, driver.Options{Dir: dir, Env: []string{goFlagsMod}})
	if err != nil {
		return nil, fmt.Errorf("analyze %s: %w", dir, err)
	}

	return findings, nil
}
