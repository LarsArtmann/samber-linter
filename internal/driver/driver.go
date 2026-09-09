// Package driver wires the healthwash analyzer into a CLI: it loads the
// target packages, runs the analyzer, converts diagnostics into go-finding
// findings (severity AND confidence), applies the config allowlist, computes
// the HW-6 coverage ratchet as a project-level post-pass, persists baselines
// atomically, and maps the outcome to the 0/1/2 confidence exit contract.
package driver

import (
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/larsartmann/go-atomic-write"
	"github.com/larsartmann/go-finding"
	linter "github.com/larsartmann/go-linter-sdk"
	"github.com/larsartmann/samber-linter/pkg/healthwash"
	"golang.org/x/tools/go/packages"

	"golang.org/x/tools/go/analysis"
)

// ToolName is the short namespace users write in suppressions and configs.
const ToolName = "samber-linter"

// DefaultBaselinePath is where --set-baseline persists the ratchet floor.
const DefaultBaselinePath = ".samber-linter-baseline.json"

// VerifiedDover lists the samber/do versions whose mechanism behavior the
// rules encode (see README §11). Targets outside this set get a warning.
var VerifiedDover = map[string]bool{"v2.0.0": true, "v2.1.0": true}

// Options configures one driver run.
type Options struct {
	Patterns []string
	Strict   bool
	JSON     bool
	SARIF    bool

	CoverageMin  float64 // negative disables the gate
	SetBaseline  bool
	BaselinePath string
	ConfigPath   string

	MinConfidence finding.Confidence
	Version       string
	Dir           string // working directory for package loading (tests)

	Stdout io.Writer
	Stderr io.Writer
}

type ruleMeta struct {
	severity   finding.Severity
	confidence finding.Confidence
}

// severity != confidence: HW-1/3/5 are type facts (Full), HW-2 a strong
// convention (High), HW-4 a judgment call (Medium). Exit codes key on
// confidence only.
var ruleMetaByRule = map[string]ruleMeta{
	healthwash.RuleHW1:        {severity: finding.SeverityWarning, confidence: finding.ConfidenceFull},
	healthwash.RuleHW2:        {severity: finding.SeverityInfo, confidence: finding.ConfidenceHigh},
	healthwash.RuleHW3:        {severity: finding.SeverityWarning, confidence: finding.ConfidenceFull},
	healthwash.RuleHW4:        {severity: finding.SeverityInfo, confidence: finding.ConfidenceMedium},
	healthwash.RuleHW5:        {severity: finding.SeverityWarning, confidence: finding.ConfidenceFull},
	healthwash.RuleHW0:        {severity: finding.SeverityWarning, confidence: finding.ConfidenceFull},
	healthwash.RuleUnresolved: {severity: finding.SeverityInfo, confidence: finding.ConfidenceMedium},
}

// allowlistConfig is the --config file: recurring suppression categories,
// kept separate from inline directives. Reasons are mandatory here too.
type allowlistConfig struct {
	Allow []allowEntry `json:"allow"`
}

type allowEntry struct {
	Rule        string `json:"rule"`        // "HW-1" etc, or "all"
	PathPattern string `json:"pathPattern"` // path.Match pattern on file paths
	Reason      string `json:"reason"`
}

// baseline is the committed ratchet floor.
type baseline struct {
	Version   int     `json:"version"`
	Checked   int     `json:"checked"`
	Registered int    `json:"registered"`
	Coverage  float64 `json:"coverage"`
}

// Run executes one analysis pass and returns the process exit code.
func Run(opts Options) int {
	out, errw := opts.Stdout, opts.Stderr
	if out == nil {
		out = os.Stdout
	}
	if errw == nil {
		errw = os.Stderr
	}
	if opts.MinConfidence == 0 {
		opts.MinConfidence = finding.ConfidenceHigh
	}
	if opts.BaselinePath == "" {
		opts.BaselinePath = DefaultBaselinePath
	}

	pkgs, err := load(opts)
	if err != nil {
		fmt.Fprintf(errw, "%s: load failed: %v\n", ToolName, err)
		return 1
	}

	analyzer := healthwash.New()
	if opts.Strict {
		_ = analyzer.Flags.Set("strict", "true")
	}

	var records []healthwash.ServiceRecord
	var findings []finding.Finding
	var loadErrs int

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Fprintf(errw, "%s: %s\n", ToolName, e)
			loadErrs++
		}
		diags, recs := runAnalyzer(analyzer, pkg)
		records = append(records, recs...)
		for _, d := range diags {
			findings = append(findings, toFinding(analyzer, d, pkg.Fset, opts.Version))
		}
	}

	findings = applyAllowlist(findings, opts.ConfigPath, errw)

	report := finding.NewReportFromFindings(
		finding.ToolInfo{Name: ToolName, Version: opts.Version}, findings)

	printText(out, findings)
	if opts.JSON {
		s, _ := report.JSON()
		fmt.Fprintln(out, s)
	}
	if opts.SARIF {
		b, err := report.ToSARIF()
		if err != nil {
			fmt.Fprintf(errw, "%s: sarif export failed: %v\n", ToolName, err)
		} else {
			fmt.Fprintln(out, string(b))
		}
	}

	gateFailed := false
	coverage := reportCoverage(out, records, opts, &gateFailed)
	warnDover(out, pkgs)

	if loadErrs > 0 {
		fmt.Fprintf(errw, "%s: %d package error(s) during load; results may be incomplete\n",
			ToolName, loadErrs)
	}

	code := linter.ExitCodeByConfidence(report, opts.MinConfidence)
	if gateFailed && code == 0 {
		code = 1
	}
	_ = coverage
	return code
}

func load(opts Options) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedDeps | packages.NeedImports |
			packages.NeedModule,
		Dir:   opts.Dir,
		Tests: false, // composition roots are what dashboards see; DO-3 keeps Override* in tests
	}
	return packages.Load(cfg, opts.Patterns...)
}

// runAnalyzer builds an analysis.Pass by hand (the x/tools checker internals
// are internal) and runs the analyzer over one package. Facts are collected
// for the HW-6 post-pass; no cross-package fact imports are needed because
// registration facts are self-contained per package.
func runAnalyzer(analyzer *analysis.Analyzer, pkg *packages.Package) (
	[]analysis.Diagnostic, []healthwash.ServiceRecord,
) {
	pass := &analysis.Pass{
		Analyzer:    analyzer,
		Fset:        pkg.Fset,
		Files:       pkg.Syntax,
		OtherFiles:  pkg.OtherFiles,
		IgnoredFiles: pkg.IgnoredFiles,
		Pkg:         pkg.Types,
		TypesInfo:   pkg.TypesInfo,
		TypesSizes:  types.SizesFor("gc", runtime.GOARCH),
		Report:      func(analysis.Diagnostic) {},
		ImportObjectFact: func(obj types.Object, fact analysis.Fact) bool { return false },
		ExportObjectFact: func(obj types.Object, fact analysis.Fact) {},
		ImportPackageFact: func(p *types.Package, fact analysis.Fact) bool { return false },
		ExportPackageFact: func(fact analysis.Fact) {},
		AllObjectFacts:   func() []analysis.ObjectFact { return nil },
		AllPackageFacts:  func() []analysis.PackageFact { return nil },
		ResultOf:         map[*analysis.Analyzer]interface{}{},
	}

	var diags []analysis.Diagnostic
	var records []healthwash.ServiceRecord
	pass.Report = func(d analysis.Diagnostic) { diags = append(diags, d) }
	pass.ExportPackageFact = func(fact analysis.Fact) {
		if pf, ok := fact.(*healthwash.PackageFacts); ok {
			records = append(records, pf.Records...)
		}
	}

	_, _ = analyzer.Run(pass)
	return diags, records
}

func toFinding(analyzer *analysis.Analyzer, d analysis.Diagnostic, fset *token.FileSet, version string) finding.Finding {
	rule := d.Category
	if rule == "" {
		rule = analyzer.Name
	}
	meta, ok := ruleMetaByRule[rule]
	if !ok {
		meta = ruleMeta{severity: finding.SeverityWarning, confidence: finding.ConfidenceMedium}
	}
	pos := fset.Position(d.Pos)
	return finding.NewBuilder(
		finding.RuleName(rule),
		finding.ToolName(ToolName),
		d.Message,
		meta.severity,
		finding.Position{File: finding.FilePath(pos.Filename), Line: pos.Line, Column: pos.Column, Offset: pos.Offset},
	).WithConfidence(meta.confidence).MustBuild()
}

func applyAllowlist(findings []finding.Finding, configPath string, errw io.Writer) []finding.Finding {
	if configPath == "" {
		return findings
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(errw, "%s: config: %v\n", ToolName, err)
		return findings
	}
	var cfg allowlistConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(errw, "%s: config parse: %v\n", ToolName, err)
		return findings
	}
	for _, e := range cfg.Allow {
		if e.Reason == "" {
			fmt.Fprintf(errw, "%s: config allowlist entry for %s lacks a reason; ignoring entry (unexplained suppressions rot)\n",
				ToolName, e.Rule)
		}
	}
	kept := findings[:0]
	for _, f := range findings {
		suppressed := false
		for _, e := range cfg.Allow {
			if e.Reason == "" {
				continue
			}
			if !allowMatches(e.Rule, string(f.Rule)) {
				continue
			}
			if ok, _ := path.Match(e.PathPattern, string(f.Position.File)); ok {
				suppressed = true
				break
			}
		}
		if !suppressed {
			kept = append(kept, f)
		}
	}
	return kept
}

// allowMatches reports whether an allowlist entry covers a rule. "all"
// matches everything; comparison is case-insensitive on the HW code.
func allowMatches(entry, rule string) bool {
	e := strings.ToLower(strings.TrimSpace(entry))
	if e == "all" {
		return true
	}
	return e == strings.ToLower(rule)
}

func reportCoverage(out io.Writer, records []healthwash.ServiceRecord, opts Options, gateFailed *bool) float64 {
	uniq := map[string]healthwash.ServiceRecord{}
	for _, r := range records {
		if !r.AffectsHW6() {
			continue // alias rows delegate to their target; never counted
		}
		if prev, ok := uniq[r.Name]; !ok || (r.Kind != healthwash.KindTransient && prev.ImplementsCheck != r.ImplementsCheck) {
			uniq[r.Name] = r
		}
	}
	registered := len(uniq)
	checked := 0
	for _, r := range uniq {
		// Transients are skipped, never checked — the sweep never dispatches
		// to them even when the type implements a check (README §2.4, §7).
		if r.ImplementsCheck && r.Kind != healthwash.KindTransient {
			checked++
		}
	}
	coverage := 0.0
	if registered > 0 {
		coverage = float64(checked) / float64(registered)
	}

	baselinePath := opts.BaselinePath
	if opts.SetBaseline {
		b := baseline{Version: 1, Checked: checked, Registered: registered, Coverage: coverage}
		data, _ := json.MarshalIndent(b, "", "  ")
		if _, err := atomicwrite.WriteIfChanged(baselinePath, append(data, '\n')); err != nil {
			fmt.Fprintf(opts.Stderr, "%s: baseline write failed: %v\n", ToolName, err)
			*gateFailed = true
		} else {
			fmt.Fprintf(out, "baseline written: %s (coverage %d/%d = %.0f%%)\n",
				baselinePath, checked, registered, coverage*100)
		}
		return coverage
	}

	if opts.CoverageMin >= 0 {
		fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (threshold: %.0f%%)\n",
			checked, registered, coverage*100, opts.CoverageMin*100)
		if registered > 0 && coverage < opts.CoverageMin {
			fmt.Fprintf(opts.Stderr, "%s: coverage %.0f%% is below the required minimum %.0f%%\n",
				ToolName, coverage*100, opts.CoverageMin*100)
			*gateFailed = true
		}
		return coverage
	}

	if data, err := os.ReadFile(baselinePath); err == nil {
		var b baseline
		if json.Unmarshal(data, &b) == nil && b.Registered > 0 {
			fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (baseline: %.0f%%)\n",
				checked, registered, coverage*100, b.Coverage*100)
			if coverage < b.Coverage {
				fmt.Fprintf(opts.Stderr, "%s: coverage %.0f%% regressed below the committed baseline %.0f%%; fix the regressions or explicitly re-baseline with --set-baseline\n",
					ToolName, coverage*100, b.Coverage*100)
				*gateFailed = true
			} else if coverage > b.Coverage {
				fmt.Fprintf(out, "coverage improved; lock in the gain with --set-baseline\n")
			}
			return coverage
		}
	}

	if registered > 0 {
		fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (no baseline; use --set-baseline to start the ratchet)\n",
			checked, registered, coverage*100)
	}
	return coverage
}

func printText(out io.Writer, findings []finding.Finding) {
	for _, f := range findings {
		fmt.Fprintf(out, "%s:%d:%d: %s\n",
			f.Position.File, f.Position.Line, f.Position.Column, f.Message)
	}
	if len(findings) == 0 {
		fmt.Fprintln(out, "no health-washing found")
	}
}
