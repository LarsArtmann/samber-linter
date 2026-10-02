// Command samber-linter detects health-washing in samber/do v2 containers:
// services that render green "pass" on health dashboards but cannot actually
// fail. See README.md for the verified mechanism anatomy and rule table.
//
// The CLI is built on cmdguard in its single-command shape: a typed flag
// struct registered on the root command. The driver owns every user-facing
// line (findings, load errors, gate diagnostics), so driver exit codes 1/2
// become exit-code-only errors that fang must not re-print (see
// reportOnlyUnreportedErrors).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/fang/v2"
	v4 "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
	"github.com/spf13/cobra"
)

// defaultMinConfidence is the exit-1 threshold for the confidence gate. It is
// mirrored by the min-confidence flag's default tag; flags_test.go fails the
// suite when the two drift apart.
const defaultMinConfidence = 0.75

// linterFlags is the complete flag surface of the driver. Field tags are the
// contract: names, defaults, and help must match the stdlib-flag CLI this
// command replaced (README §1/§7). Unlike stdlib flag, pflag parses
// interspersed flags, so `samber-linter ./... --json` and a trailing `--help`
// work natively (the wantsHelp workaround from the 2026-10-02 webphone report
// became obsolete). flags_test.go pins the dynamic parts (output format list,
// baseline default) to the driver package.
type linterFlags struct {
	JSON          bool    `flag:"json" help:"emit the go-finding JSON report"`
	SARIF         bool    `flag:"sarif" help:"emit a SARIF 2.1 report for code scanning"`
	OutputFormat  string  `flag:"output" default:"" help:"findings presentation format; plain text lines by default"`
	Strict        bool    `flag:"strict" help:"report HW-unresolved for statically unresolvable service types"`
	DisableRules  string  `flag:"disable" default:"" help:"comma-separated rule IDs to skip (e.g. HW-1,HW-4); testing/migration aid"`
	Check         bool    `flag:"check" help:"advisory mode: report everything but always exit 0 (for CI annotation pipelines)"`
	CoverageMin   float64 `flag:"coverage-min" default:"-1" help:"fail when health coverage is below this fraction (0..1)"`
	SetBaseline   bool    `flag:"set-baseline" help:"write the current coverage as the ratchet floor and pass"`
	BaselinePath  string  `flag:"baseline" default:".samber-linter-baseline.json" help:"path of the committed coverage baseline file"`
	ConfigPath    string  `flag:"config" default:"" help:"path of the allowlist config for recurring suppression categories"`
	MinConfidence float64 `flag:"min-confidence" default:"0.75" help:"exit 1 when any finding is at or above this confidence (0..1)"`
	ShowVersion   bool    `flag:"version" help:"print the tool version"`
}

func main() {
	if version == "" {
		version = resolveVersion()
	}

	cli, err := v4.NewCLI[linterFlags](
		"samber-linter",
		"detects health-washing in samber/do v2 containers",
		linterFlags{},
		v4.WithCLILong(longDescription),
		v4.WithHelpTransform(dynamicOutputHelp),
		// fang would inject its own --version flag backed by the module
		// version; the tool's --version prints the build-resolved version
		// string exactly (consumers gate on it), so fang's must stay off.
		v4.WithFangOptions(fang.WithoutVersion()),
		v4.WithFangErrorHandler(reportOnlyUnreportedErrors),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "samber-linter: building CLI: %v\n", err)
		os.Exit(2)
	}

	root := cli.RootCommand()
	root.Use = "samber-linter [flags] <packages...>"

	// cobra auto-adds help/completion subcommands; cobra's default root arg
	// policy then rejects positional package patterns as "unknown command".
	root.Args = cobra.ArbitraryArgs

	root.RunE = func(_ *cobra.Command, args []string) error {
		return runLinter(cli.Config(), args)
	}

	cli.ExecuteAndExit(context.Background())
}

// runLinter maps the parsed flag struct onto one driver pass and converts the
// driver's exit code into cmdguard's error contract: 0 stays nil, 1/2 become
// ExitCoder errors (silent ones — the driver already reported).
func runLinter(cfg *linterFlags, args []string) error {
	if cfg.ShowVersion {
		fmt.Fprintln(os.Stdout, version)

		return nil
	}

	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	outputFormat, err := driver.ParseOutputFormat(cfg.OutputFormat)
	if err != nil {
		// Actionable and not yet reported: keep the message (fang prints it)
		// and stay on the documented tri-state (2 = no findings produced).
		return newReportedExitError(2, err)
	}

	code := driver.Run(driver.Options{
		Patterns:      patterns,
		Strict:        cfg.Strict,
		JSON:          cfg.JSON,
		SARIF:         cfg.SARIF,
		Check:         cfg.Check,
		OutputFormat:  outputFormat,
		CoverageMin:   cfg.CoverageMin,
		SetBaseline:   cfg.SetBaseline,
		BaselinePath:  cfg.BaselinePath,
		ConfigPath:    cfg.ConfigPath,
		DisableRules:  cfg.DisableRules,
		MinConfidence: finding.Confidence(cfg.MinConfidence),
		Version:       version,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	})
	if code == 0 {
		return nil
	}

	return newSilentExitError(code)
}

// newSilentExitError wraps a driver exit code whose user-facing report the
// driver has already printed.
func newSilentExitError(code int) error {
	exitErr, err := v4.NewExitError(code, nil)
	if err != nil {
		return fmt.Errorf("mapping driver exit code %d: %w", code, err)
	}

	return exitErr
}

// newReportedExitError wraps an error fang should still print, pinned to a
// specific exit code.
func newReportedExitError(code int, cause error) error {
	exitErr, err := v4.NewExitError(code, cause)
	if err != nil {
		return fmt.Errorf("mapping exit code %d: %w", code, err)
	}

	return exitErr
}

// reportOnlyUnreportedErrors is the fang error handler: silent for exit-code
// errors with no cause (the driver already printed everything), styled output
// for every other error (flag misuse, invalid flag values).
func reportOnlyUnreportedErrors(w io.Writer, styles fang.Styles, err error) {
	var exitErr *v4.ExitError
	if errors.As(err, &exitErr) && exitErr.Err == nil {
		return
	}

	fang.DefaultErrorHandler(w, styles, err)
}

// dynamicOutputHelp rewrites the --output flag's help at execution time from
// the driver's supported-format list, so the help text can never drift from
// the formats ParseOutputFormat actually accepts.
func dynamicOutputHelp(root *cobra.Command) {
	outputFlag := root.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		return
	}

	outputFlag.Usage = fmt.Sprintf(
		"findings presentation format (%s); plain text lines by default",
		strings.Join(driver.SupportedOutputFormatNames(), ", "))
}

const longDescription = `Detects health-washing in samber/do v2 containers: services that render
green "pass" on health dashboards but cannot actually fail.

Exit codes: 0 clean, 1 findings (or a coverage gate failure), 2 load failure
or triage-only findings. See README.md for the rule table (HW-1..HW-6), the
mechanism anatomy pinned to samber/do v2.1.0, and the suppression syntax.`
