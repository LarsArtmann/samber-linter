// Package sdk exposes samber-linter's static analysis as a programmatic API
// for in-process consumers such as BuildFlow's DAG provider. It wraps the
// internal driver without exposing the CLI's exit-code contract: callers get
// findings or an error, never stdout to parse.
package sdk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// Analyze loads the Go packages under dir (pattern "./...", test files
// excluded — composition roots are what dashboards see) and returns the
// healthwash findings, with the repo's inline //samber-linter:allow
// suppressions already applied. dir "" means the process working directory.
// A load failure or any package error is returned as an error: incomplete
// results are never reported as clean.
func Analyze(ctx context.Context, dir string) ([]finding.Finding, error) {
	findings, err := driver.Analyze(ctx, driver.Options{Dir: dir, Env: sanitizedGoFlagsEnv()})
	if err != nil {
		return nil, fmt.Errorf("analyze %s: %w", dir, err)
	}

	return findings, nil
}

// sanitizedGoFlagsEnv returns a GOFLAGS env entry with any -mod token
// stripped. Neither inherited extreme is safe for package loading in BOTH
// directions: -mod=vendor fails when the analyzed repo has no vendor
// directory, and -mod=mod is illegal in workspace mode ("go: -mod may only
// be set to readonly or vendor when in workspace mode"). Stripping lets the
// go command negotiate per target: readonly, or vendor when vendor/ exists.
//
// The entry is ALWAYS appended (even when empty): child processes
// deduplicate env with the LAST occurrence winning, so an empty GOFLAGS
// reliably shadows an inherited one, while non-mod flags are preserved.
func sanitizedGoFlagsEnv() []string {
	return []string{"GOFLAGS=" + stripModTokens(os.Getenv("GOFLAGS"))}
}

// stripModTokens removes "-mod=..." (and a bare "-mod") from a
// space-separated GOFLAGS value.
func stripModTokens(flags string) string {
	tokens := strings.Fields(flags)

	kept := tokens[:0:0]

	for _, tok := range tokens {
		if tok == "-mod" || strings.HasPrefix(tok, "-mod=") {
			continue
		}

		kept = append(kept, tok)
	}

	return strings.Join(kept, " ")
}
