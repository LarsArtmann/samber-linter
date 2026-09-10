// Package driver wires the healthwash analyzer into a CLI: it loads the
// target packages, runs the analyzer, converts diagnostics into go-finding
// findings (severity AND confidence), applies the config allowlist, computes
// the HW-6 coverage ratchet as a project-level post-pass, persists baselines
// atomically, and maps the outcome to the 0/1/2 confidence exit contract.
package driver

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"os"
	"path"
	"runtime"
	"slices"
	"strings"

	atomicwrite "github.com/larsartmann/go-atomic-write"
	"github.com/larsartmann/go-finding"
	linter "github.com/larsartmann/go-linter-sdk"
	output "github.com/larsartmann/go-output"
	"github.com/larsartmann/samber-linter/pkg/healthwash"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// ToolName is the short namespace users write in suppressions and configs.
const ToolName = "samber-linter"

// DefaultBaselinePath is where --set-baseline persists the ratchet floor.
const DefaultBaselinePath = ".samber-linter-baseline.json"

// VerifiedDoVersions lists the samber/do versions whose mechanism behavior
// the rules encode (see README §11). Targets outside this set get a warning.
func VerifiedDoVersions() []string {
	return []string{"v2.0.0", "v2.1.0"}
}

// isVerifiedDoVersion reports whether v is inside the verified set.
func isVerifiedDoVersion(v string) bool {
	return slices.Contains(VerifiedDoVersions(), v)
}

// Options configures one driver run.
type Options struct {
	Patterns []string
	Strict   bool
	JSON     bool
	SARIF    bool
	Check    bool // advisory mode: report everything, always exit 0

	// OutputFormat selects the findings presentation format via --output
	// (empty = the default plain text lines). Machine formats stay on
	// --json/--sarif.
	OutputFormat output.Format

	CoverageMin  float64 // negative disables the gate
	SetBaseline  bool
	BaselinePath string
	ConfigPath   string
	DisableRules string // comma-separated rule IDs passed to the analyzer's disable flag

	MinConfidence finding.Confidence
	Version       string
	Dir           string   // working directory for package loading (tests)
	Env           []string // extra env for package loading (tests: GOFLAGS=-mod=mod)

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
//
//nolint:gochecknoglobals // read-only rule metadata table (severity/confidence per rule)
var ruleMetaByRule = map[string]ruleMeta{
	healthwash.RuleHW1: {
		severity:   finding.SeverityWarning,
		confidence: finding.ConfidenceFull,
	},
	healthwash.RuleHW2: {severity: finding.SeverityInfo, confidence: finding.ConfidenceHigh},
	healthwash.RuleHW3: {
		severity:   finding.SeverityWarning,
		confidence: finding.ConfidenceFull,
	},
	healthwash.RuleHW4: {
		severity:   finding.SeverityInfo,
		confidence: finding.ConfidenceMedium,
	},
	healthwash.RuleHW5: {
		severity:   finding.SeverityWarning,
		confidence: finding.ConfidenceFull,
	},
	healthwash.RuleHW0: {
		severity:   finding.SeverityWarning,
		confidence: finding.ConfidenceFull,
	},
	healthwash.RuleUnresolved: {
		severity:   finding.SeverityInfo,
		confidence: finding.ConfidenceMedium,
	},
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
	Version    int     `json:"version"`
	Checked    int     `json:"checked"`
	Registered int     `json:"registered"`
	Coverage   float64 `json:"coverage"`
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

		return 2 // exit 1 is reserved for findings; a tool that cannot load has none
	}

	analyzer := buildAnalyzer(opts)

	findings, records := analyzePackages(analyzer, pkgs, errw)
	findings = applyAllowlist(findings, opts.ConfigPath, errw)

	report := finding.NewReportFromFindings(
		finding.ToolInfo{Name: ToolName, Version: opts.Version}, findings)

	if !emitOutputs(out, errw, report, findings, opts) {
		return 1
	}

	code := applyGates(out, report, records, pkgs, opts)
	if opts.Check {
		fmt.Fprintln(out, "--check: advisory run; exit code forced to 0")

		return 0
	}

	return code
}

// buildAnalyzer constructs the analyzer with the driver-level flags applied.
func buildAnalyzer(opts Options) *analysis.Analyzer {
	analyzer := healthwash.New()
	if opts.Strict {
		_ = analyzer.Flags.Set("strict", "true")
	}

	if opts.DisableRules != "" {
		_ = analyzer.Flags.Set("disable", opts.DisableRules)
	}

	return analyzer
}

// analyzePackages runs the analyzer over every loaded package, converting
// diagnostics to findings and collecting registration records for HW-6.
func analyzePackages(
	analyzer *analysis.Analyzer, pkgs []*packages.Package, errw io.Writer,
) ([]finding.Finding, []healthwash.ServiceRecord) {
	var (
		records  []healthwash.ServiceRecord
		findings []finding.Finding
		loadErrs int
	)

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Fprintf(errw, "%s: %s\n", ToolName, e)

			loadErrs++
		}

		diags, recs := runAnalyzer(analyzer, pkg)

		records = append(records, recs...)
		for _, d := range diags {
			findings = append(findings, toFinding(analyzer, d, pkg.Fset))
		}
	}

	if loadErrs > 0 {
		fmt.Fprintf(errw, "%s: %d package error(s) during load; results may be incomplete\n",
			ToolName, loadErrs)
	}

	return findings, records
}

// emitOutputs writes the findings in every requested representation. It
// returns false when a requested presentation cannot be rendered (the run
// must then fail with exit 1).
func emitOutputs(
	out, errw io.Writer, report *finding.Report, findings []finding.Finding, opts Options,
) bool {
	if opts.OutputFormat != "" {
		if err := renderFindings(out, findings, opts.OutputFormat); err != nil {
			fmt.Fprintf(errw, "%s: %v\n", ToolName, err)

			return false
		}
	} else {
		printText(out, findings)
	}

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

	return true
}

// applyGates computes the HW-6 ratchet output, warns about the samber/do
// version, and maps the outcome to the confidence exit contract.
func applyGates(
	out io.Writer, report *finding.Report, records []healthwash.ServiceRecord,
	pkgs []*packages.Package, opts Options,
) int {
	gateFailed := false
	coverage := reportCoverage(out, records, opts, &gateFailed)
	warnDover(out, pkgs)

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
		Env:   append(os.Environ(), opts.Env...),
		Tests: false, // composition roots are what dashboards see; DO-3 keeps Override* in tests
	}

	pkgs, err := packages.Load(cfg, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	return pkgs, nil
}

// runAnalyzer builds an analysis.Pass by hand (the x/tools checker internals
// are internal) and runs the analyzer over one package. Facts are collected
// for the HW-6 post-pass; no cross-package fact imports are needed because
// registration facts are self-contained per package.
func runAnalyzer(analyzer *analysis.Analyzer, pkg *packages.Package) (
	[]analysis.Diagnostic, []healthwash.ServiceRecord,
) {
	pass := &analysis.Pass{
		Analyzer:          analyzer,
		Fset:              pkg.Fset,
		Files:             pkg.Syntax,
		OtherFiles:        pkg.OtherFiles,
		IgnoredFiles:      pkg.IgnoredFiles,
		Pkg:               pkg.Types,
		TypesInfo:         pkg.TypesInfo,
		TypesSizes:        types.SizesFor("gc", runtime.GOARCH),
		Report:            func(analysis.Diagnostic) {},
		ImportObjectFact:  func(obj types.Object, fact analysis.Fact) bool { return false },
		ExportObjectFact:  func(obj types.Object, fact analysis.Fact) {},
		ImportPackageFact: func(p *types.Package, fact analysis.Fact) bool { return false },
		ExportPackageFact: func(fact analysis.Fact) {},
		AllObjectFacts:    func() []analysis.ObjectFact { return nil },
		AllPackageFacts:   func() []analysis.PackageFact { return nil },
		ResultOf:          map[*analysis.Analyzer]any{},
	}

	var (
		diags   []analysis.Diagnostic
		records []healthwash.ServiceRecord
	)

	pass.Report = func(d analysis.Diagnostic) { diags = append(diags, d) }
	pass.ExportPackageFact = func(fact analysis.Fact) {
		if pf, ok := fact.(*healthwash.PackageFacts); ok {
			records = append(records, pf.Records...)
		}
	}

	_, _ = analyzer.Run(pass)

	return diags, records
}

func toFinding(
	analyzer *analysis.Analyzer,
	diag analysis.Diagnostic,
	fset *token.FileSet,
) finding.Finding {
	rule := diag.Category
	if rule == "" {
		rule = analyzer.Name
	}

	meta, ok := ruleMetaByRule[rule]
	if !ok {
		meta = ruleMeta{severity: finding.SeverityWarning, confidence: finding.ConfidenceMedium}
	}

	pos := fset.Position(diag.Pos)

	return finding.NewBuilder(
		finding.RuleName(rule),
		finding.ToolName(ToolName),
		diag.Message,
		meta.severity,
		finding.Position{
			File:   finding.FilePath(pos.Filename),
			Line:   pos.Line,
			Column: pos.Column,
			Offset: pos.Offset,
		},
	).WithConfidence(meta.confidence).MustBuild()
}

func applyAllowlist(
	findings []finding.Finding,
	configPath string,
	errw io.Writer,
) []finding.Finding {
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

	warnMalformedAllowlist(cfg.Allow, errw)

	usable := usableAllowEntries(cfg.Allow)

	kept := findings[:0]
	for _, entry := range findings {
		suppressed := false

		for _, e := range usable {
			if entryCovers(e, entry) {
				suppressed = true

				break
			}
		}

		if !suppressed {
			kept = append(kept, entry)
		}
	}

	return kept
}

// warnMalformedAllowlist explains why an entry is ignored: unexplained
// suppressions rot, and silent no-ops are worse than errors.
func warnMalformedAllowlist(entries []allowEntry, errw io.Writer) {
	for _, entry := range entries {
		if entry.Reason == "" {
			fmt.Fprintf(
				errw,
				"%s: config allowlist entry for %s lacks a reason; ignoring entry (unexplained suppressions rot)\n",
				ToolName,
				entry.Rule,
			)
		}

		if strings.TrimSpace(entry.Rule) == "" {
			fmt.Fprintf(
				errw,
				"%s: config allowlist entry lacks a rule; ignoring entry (name the rule, or \"all\")\n",
				ToolName,
			)
		}
	}
}

// usableAllowEntries keeps only entries that can ever match: reason present,
// rule named.
func usableAllowEntries(entries []allowEntry) []allowEntry {
	usable := make([]allowEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.Reason != "" && strings.TrimSpace(entry.Rule) != "" {
			usable = append(usable, entry)
		}
	}

	return usable
}

// entryCovers reports whether one allowlist entry suppresses one finding. An
// empty pathPattern means the entry covers every file: it is the project-wide
// form, not a silent no-op.
func entryCovers(entry allowEntry, fnd finding.Finding) bool {
	if !allowMatches(entry.Rule, string(fnd.Rule)) {
		return false
	}

	if entry.PathPattern == "" {
		return true
	}

	matched, _ := path.Match(entry.PathPattern, string(fnd.Position.File))

	return matched
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

func reportCoverage(
	out io.Writer,
	records []healthwash.ServiceRecord,
	opts Options,
	gateFailed *bool,
) float64 {
	uniq := map[string]healthwash.ServiceRecord{}

	for _, record := range records {
		if !record.AffectsHW6() {
			continue // alias rows delegate to their target; never counted
		}

		if prev, ok := uniq[record.Name]; !ok ||
			(record.Kind != healthwash.KindTransient && prev.ImplementsCheck != record.ImplementsCheck) {
			uniq[record.Name] = record
		}
	}

	registered := len(uniq)
	checked := 0

	for _, record := range uniq {
		// Transients are skipped, never checked — the sweep never dispatches
		// to them even when the type implements a check (README §2.4, §7).
		if record.ImplementsCheck && record.Kind != healthwash.KindTransient {
			checked++
		}
	}

	coverage := 0.0
	if registered > 0 {
		coverage = float64(checked) / float64(registered)
	}

	baselinePath := opts.BaselinePath
	if opts.SetBaseline {
		writeBaseline(out, opts.Stderr, baselinePath, baseline{
			Version: 1, Checked: checked, Registered: registered, Coverage: coverage,
		}, gateFailed)

		return coverage
	}

	if opts.CoverageMin > 0 {
		enforceCoverageMin(out, opts.Stderr, checked, registered, coverage, opts.CoverageMin, gateFailed)

		return coverage
	}

	enforceBaselineRatchet(out, opts.Stderr, baselinePath, checked, registered, coverage, gateFailed)

	return coverage
}

// writeBaseline persists the current coverage as the ratchet floor.
func writeBaseline(out, errw io.Writer, path string, b baseline, gateFailed *bool) {
	data, _ := json.Marshal(b, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if _, err := atomicwrite.WriteIfChanged(path, append(data, '\n')); err != nil {
		fmt.Fprintf(errw, "%s: baseline write failed: %v\n", ToolName, err)

		*gateFailed = true

		return
	}

	fmt.Fprintf(out, "baseline written: %s (coverage %d/%d = %.0f%%)\n",
		path, b.Checked, b.Registered, b.Coverage*100)
}

// enforceCoverageMin applies the absolute --coverage-min gate.
func enforceCoverageMin(
	out, errw io.Writer, checked, registered int, coverage, minCoverage float64, gateFailed *bool,
) {
	fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (threshold: %.0f%%)\n",
		checked, registered, coverage*100, minCoverage*100)

	if registered > 0 && coverage < minCoverage {
		fmt.Fprintf(errw, "%s: coverage %.0f%% is below the required minimum %.0f%%\n",
			ToolName, coverage*100, minCoverage*100)

		*gateFailed = true
	}
}

// enforceBaselineRatchet applies the committed-baseline regression gate.
func enforceBaselineRatchet(
	out, errw io.Writer, baselinePath string, checked, registered int, coverage float64, gateFailed *bool,
) {
	if data, err := os.ReadFile(baselinePath); err == nil {
		var b baseline
		if json.Unmarshal(data, &b) == nil && b.Registered > 0 {
			fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (baseline: %.0f%%)\n",
				checked, registered, coverage*100, b.Coverage*100)

			if coverage < b.Coverage {
				fmt.Fprintf(
					errw,
					"%s: coverage %.0f%% regressed below the committed baseline %.0f%%; "+
						"fix the regressions or explicitly re-baseline with --set-baseline\n",
					ToolName,
					coverage*100,
					b.Coverage*100,
				)

				*gateFailed = true
			} else if coverage > b.Coverage {
				fmt.Fprintln(out, "coverage improved; lock in the gain with --set-baseline")
			}

			return
		}
	}

	if registered > 0 {
		fmt.Fprintf(
			out,
			"health-coverage: %d/%d = %.0f%% (no baseline; use --set-baseline to start the ratchet)\n",
			checked,
			registered,
			coverage*100,
		)
	}
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
