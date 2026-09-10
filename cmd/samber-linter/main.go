// Command samber-linter detects health-washing in samber/do v2 containers:
// services that render green "pass" on health dashboards but cannot actually
// fail. See README.md for the verified mechanism anatomy and rule table.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/samber-linter/internal/driver"
)

// version is the tool version reported in findings and SARIF exports.
var version = "0.1.0"

// defaultMinConfidence is the exit-1 threshold for the confidence gate.
const defaultMinConfidence = 0.75

func main() {
	fs := flag.NewFlagSet("samber-linter", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit the go-finding JSON report")
	sarifOut := fs.Bool("sarif", false, "emit a SARIF 2.1 report for code scanning")
	outputFlag := fs.String("output", "", fmt.Sprintf(
		"findings presentation format (%s); plain text lines by default",
		strings.Join(driver.SupportedOutputFormatNames(), ", ")))
	strict := fs.Bool(
		"strict",
		false,
		"report HW-unresolved for statically unresolvable service types",
	)
	disable := fs.String(
		"disable",
		"",
		"comma-separated rule IDs to skip (e.g. HW-1,HW-4); testing/migration aid",
	)
	check := fs.Bool(
		"check",
		false,
		"advisory mode: report everything but always exit 0 (for CI annotation pipelines)",
	)
	coverageMin := fs.Float64(
		"coverage-min",
		-1,
		"fail when health coverage is below this fraction (0..1)",
	)
	setBaseline := fs.Bool(
		"set-baseline",
		false,
		"write the current coverage as the ratchet floor and pass",
	)
	baselinePath := fs.String(
		"baseline",
		driver.DefaultBaselinePath,
		"path of the committed coverage baseline file",
	)
	configPath := fs.String(
		"config",
		"",
		"path of the allowlist config for recurring suppression categories",
	)
	minConf := fs.Float64(
		"min-confidence",
		defaultMinConfidence,
		"exit 1 when any finding is at or above this confidence (0..1)",
	)
	showVersion := fs.Bool("version", false, "print the tool version")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: samber-linter [flags] <packages...>")
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(version)

		return
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	outputFormat, err := driver.ParseOutputFormat(*outputFlag)
	if err != nil {
		fmt.Fprintln(fs.Output(), err)
		fs.Usage()

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
