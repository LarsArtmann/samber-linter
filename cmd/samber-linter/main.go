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
	"slices"
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
// contract: flag names and semantics must match the stdlib-flag CLI this
// command replaced (README §1/§7); flags_test.go pins the surface (names,
// defaults, help wiring) to the driver package. Help texts stay short so the
// tagalign columns fit the 120-column lll budget; --output's full format list
// is injected at runtime by dynamicOutputHelp. Unlike stdlib flag, pflag
// parses interspersed flags, so `samber-linter ./... --json` and a trailing
// `--help` work natively (the wantsHelp workaround from the 2026-10-02
// webphone report became obsolete).
type linterFlags struct {
	JSON          bool    `flag:"json"           help:"emit the go-finding JSON report"`
	SARIF         bool    `flag:"sarif"          help:"emit a SARIF 2.1 report for code scanning"`
	OutputFormat  string  `flag:"output"         help:"findings presentation format"                default:""`
	Strict        bool    `flag:"strict"         help:"report HW-unresolved for unresolvable types"`
	DisableRules  string  `flag:"disable"        help:"comma-separated rule IDs to skip"            default:""`
	Check         bool    `flag:"check"          help:"advisory mode: always exit 0"`
	CoverageMin   float64 `flag:"coverage-min"   help:"fail below this coverage (0..1)"             default:"-1"`
	SetBaseline   bool    `flag:"set-baseline"   help:"write current coverage as ratchet floor"`
	BaselinePath  string  `flag:"baseline"       help:"committed coverage baseline file"            default:""`
	ConfigPath    string  `flag:"config"         help:"allowlist config for suppressions"           default:""`
	MinConfidence float64 `flag:"min-confidence" help:"exit 1 at/above this (0..1)"                 default:"0.75"`
	ShowVersion   bool    `flag:"version"        help:"print the tool version"`
}

func main() {
	if version == "" {
		version = resolveVersion()
	}

	cli, err := newCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "samber-linter: building CLI: %v\n", err)
		os.Exit(2)
	}

	if err := cli.ExecuteWithArgs(context.Background(), normalizeHelpFlag(os.Args[1:])); err != nil {
		os.Exit(v4.ExitCode(err))
	}
}

// normalizeHelpFlag rewrites the bare "-help" token to "--help" before cobra
// parses it: pflag reads single-dash tokens as shorthand clusters, so "-help"
// would die on unknown shorthand 'e' (styled error, exit 1) instead of
// printing help the way the stdlib-flag CLI this tool shipped as v0.1..v0.3
// did. Only the exact standalone token is rewritten; genuine flag misuse
// (including "-help=x") stays an error.
func normalizeHelpFlag(args []string) []string {
	for i, arg := range args {
		if arg == "-help" {
			rewritten := slices.Clone(args)
			rewritten[i] = "--help"

			return rewritten
		}
	}

	return args
}

// newCLI builds the complete CLI exactly as main runs it: cmdguard
// construction, the root command's Use line, ArbitraryArgs (cobra's default
// root arg policy would reject positional package patterns as "unknown
// command"), and the RunE bridge into runLinter. Extracted from main so the
// contract tests execute the real wiring, not a parallel copy of it.
func newCLI() (*v4.CLI[linterFlags], error) {
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
		return nil, err
	}

	root := cli.RootCommand()
	root.Use = "samber-linter [flags] <packages...>"

	root.Args = cobra.ArbitraryArgs

	root.RunE = func(_ *cobra.Command, args []string) error {
		return runLinter(cli.Config(), args)
	}

	return cli, nil
}

// runLinter maps the parsed flag struct onto one driver pass and converts the
// driver's exit code into cmdguard's error contract: 0 stays nil, 1/2 become
// ExitCoder errors (silent ones — the driver already reported).
func runLinter(cfg *linterFlags, args []string) error {
	if cfg.ShowVersion {
		fmt.Fprintln(os.Stdout, version)

		return nil
	}

	opts, err := buildDriverOptions(cfg, args, version)
	if err != nil {
		// Actionable and not yet reported: keep the message (fang prints it)
		// and stay on the documented tri-state (2 = no findings produced).
		return newReportedExitError(2, err)
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr

	code := driver.Run(opts)
	if code == 0 {
		return nil
	}

	return newSilentExitError(code)
}

// buildDriverOptions maps the parsed flag struct onto driver.Options, owning
// the two defaults the flag tags cannot express: the ./... pattern fallback
// and the empty --baseline fallback to the driver's documented default path
// (tag values are compile-time strings; the constant lives in the driver).
func buildDriverOptions(cfg *linterFlags, args []string, toolVersion string) (driver.Options, error) {
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	outputFormat, err := driver.ParseOutputFormat(cfg.OutputFormat)
	if err != nil {
		return driver.Options{}, err
	}

	baselinePath := cfg.BaselinePath
	if baselinePath == "" {
		baselinePath = driver.DefaultBaselinePath
	}

	return driver.Options{
		Patterns:      patterns,
		Strict:        cfg.Strict,
		JSON:          cfg.JSON,
		SARIF:         cfg.SARIF,
		Check:         cfg.Check,
		OutputFormat:  outputFormat,
		CoverageMin:   cfg.CoverageMin,
		SetBaseline:   cfg.SetBaseline,
		BaselinePath:  baselinePath,
		ConfigPath:    cfg.ConfigPath,
		DisableRules:  cfg.DisableRules,
		MinConfidence: finding.Confidence(cfg.MinConfidence),
		Version:       toolVersion,
	}, nil
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
func reportOnlyUnreportedErrors(out io.Writer, styles fang.Styles, err error) {
	var exitErr *v4.ExitError
	if errors.As(err, &exitErr) && exitErr.Err == nil {
		return
	}

	fang.DefaultErrorHandler(out, styles, err)
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
