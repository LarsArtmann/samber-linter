// Command samber-linter detects health-washing in samber/do v2 containers:
// services that render green "pass" on health dashboards but cannot actually
// fail. See README.md for the verified mechanism anatomy and rule table.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// defaultMinConfidence is the exit-1 threshold for the confidence gate.
const defaultMinConfidence = 0.75

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: samber-linter [flags] <packages...>")
	fmt.Fprintln(w, "flags may appear before or after the package patterns")
}

// wantsHelp reports whether the invocation asks for usage anywhere on the
// command line. flag.Parse stops at the first positional argument, so a
// trailing `./... --help` would otherwise load "--help" as an import path
// (the 2026-10-02 webphone report) instead of printing usage.
func wantsHelp(args []string) bool {
	return slices.ContainsFunc(args, func(a string) bool {
		return a == "-h" || a == "-help" || a == "--help"
	})
}

func main() {
	if version == "" {
		version = resolveVersion()
	}

	if wantsHelp(os.Args[1:]) {
		printUsage(os.Stdout)

		return
	}

	flagSet := flag.NewFlagSet("samber-linter", flag.ExitOnError)
	jsonOut := flagSet.Bool("json", false, "emit the go-finding JSON report")
	sarifOut := flagSet.Bool("sarif", false, "emit a SARIF 2.1 report for code scanning")
	outputFlag := flagSet.String("output", "", fmt.Sprintf(
		"findings presentation format (%s); plain text lines by default",
		strings.Join(driver.SupportedOutputFormatNames(), ", ")))
	strict := flagSet.Bool(
		"strict",
		false,
		"report HW-unresolved for statically unresolvable service types",
	)
	disable := flagSet.String(
		"disable",
		"",
		"comma-separated rule IDs to skip (e.g. HW-1,HW-4); testing/migration aid",
	)
	check := flagSet.Bool(
		"check",
		false,
		"advisory mode: report everything but always exit 0 (for CI annotation pipelines)",
	)
	coverageMin := flagSet.Float64(
		"coverage-min",
		-1,
		"fail when health coverage is below this fraction (0..1)",
	)
	setBaseline := flagSet.Bool(
		"set-baseline",
		false,
		"write the current coverage as the ratchet floor and pass",
	)
	baselinePath := flagSet.String(
		"baseline",
		driver.DefaultBaselinePath,
		"path of the committed coverage baseline file",
	)
	configPath := flagSet.String(
		"config",
		"",
		"path of the allowlist config for recurring suppression categories",
	)
	minConf := flagSet.Float64(
		"min-confidence",
		defaultMinConfidence,
		"exit 1 when any finding is at or above this confidence (0..1)",
	)
	showVersion := flagSet.Bool("version", false, "print the tool version")
	flagSet.Usage = func() {
		printUsage(flagSet.Output())
	}
	_ = flagSet.Parse(os.Args[1:])

	if *showVersion {
		fmt.Fprintln(os.Stdout, version)

		return
	}

	patterns := flagSet.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	outputFormat, err := driver.ParseOutputFormat(*outputFlag)
	if err != nil {
		fmt.Fprintln(flagSet.Output(), err)
		flagSet.Usage()

		os.Exit(2)
	}

	var minRequired finding.Confidence
	if minConf != nil {
		minRequired = finding.Confidence(*minConf)
	}

	os.Exit(driver.Run(driver.Options{
		Patterns:      patterns,
		Strict:        *strict,
		JSON:          *jsonOut,
		SARIF:         *sarifOut,
		Check:         *check,
		OutputFormat:  outputFormat,
		CoverageMin:   *coverageMin,
		SetBaseline:   *setBaseline,
		BaselinePath:  *baselinePath,
		ConfigPath:    *configPath,
		DisableRules:  *disable,
		MinConfidence: minRequired,
		Version:       version,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}))
}
