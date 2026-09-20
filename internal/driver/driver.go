// Package driver wires the healthwash analyzer into a CLI: it loads the
// target packages, runs the analyzer, converts diagnostics into go-finding
// findings (severity AND confidence), applies the config allowlist, computes
// the HW-6 coverage ratchet as a project-level post-pass, persists baselines
// atomically, and maps the outcome to the 0/1/2 confidence exit contract.
package driver

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"maps"
	"math"
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
	healthwash.RuleHW7: {
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

// baseline is the committed ratchet floor. Version 2 adds per-rule finding
// counts (aggregate coverage alone can hide a single-rule regression);
// validateBaseline rejects every other schema loudly — a ratchet that
// silently degrades is worse than a failed run.
const baselineSchemaVersion = 2

type baseline struct {
	Version    int            `json:"version"`
	Checked    int            `json:"checked"`
	Registered int            `json:"registered"`
	Coverage   float64        `json:"coverage"`
	Findings   map[string]int `json:"findings,omitempty"` // rule ID → count; absent rule means floor 0
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

	pkgs, err := load(context.Background(), opts)
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

	code := applyGates(out, report, findings, records, pkgs, opts)
	if opts.Check {
		// Machine consumers (--json/--sarif and the structured --output
		// formats) get exactly one parseable shape; the advisory note is
		// human UI and would corrupt their parsers.
		if !opts.JSON && !opts.SARIF && !IsMachineFormat(opts.OutputFormat) {
			fmt.Fprintln(out, "--check: advisory run; exit code forced to 0")
		}

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
//
// Two sweeps: HW-7's NilBodyFacts must be complete before any registration
// site imports them, and package order is load order, not dependency order.
// Sweep 1 runs everything with a silent report and collects only facts;
// sweep 2 produces the authoritative findings and records. The analyzer is a
// cheap per-package AST walk, so the duplicate execution is noise next to
// loading.
func analyzePackages(
	analyzer *analysis.Analyzer, pkgs []*packages.Package, errw io.Writer,
) ([]finding.Finding, []healthwash.ServiceRecord) {
	store := newFactStore()
	for _, pkg := range pkgs {
		runAnalyzer(analyzer, pkg, store, false)
	}

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

		diags, recs := runAnalyzer(analyzer, pkg, store, true)

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

// factStore carries NilBodyFacts between the two per-package sweeps of one
// driver pass. Keyed by object identity: packages.Load type-checks the whole
// graph from source against a shared type universe, so a method object seen
// through an importer is the same instance the declaring package produced.
type factStore struct {
	facts map[types.Object]bool
}

func newFactStore() *factStore {
	return &factStore{facts: map[types.Object]bool{}}
}

func (s *factStore) exportFrom(obj types.Object, fact analysis.Fact) {
	if _, ok := fact.(*healthwash.NilBodyFact); ok {
		s.facts[obj] = true
	}
}

func (s *factStore) importInto(obj types.Object, fact analysis.Fact) bool {
	if _, ok := fact.(*healthwash.NilBodyFact); !ok {
		return false
	}

	return s.facts[obj]
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
	out io.Writer, report *finding.Report, findings []finding.Finding,
	records []healthwash.ServiceRecord, pkgs []*packages.Package, opts Options,
) int {
	gateFailed := false
	human := opts.humanPresentation()
	coverage := reportCoverage(out, findings, records, opts, &gateFailed, human)
	warnDover(out, pkgs)

	code := linter.ExitCodeByConfidence(report, opts.MinConfidence)
	// A failed gate is a must-fix outcome regardless of the finding tiers:
	// exit 2 means "advisory, please triage", and a regressed ratchet or an
	// unreadable baseline is never advisory.
	if gateFailed {
		code = 1
	}

	_ = coverage

	return code
}

// load loads the target packages for one run. ctx may be nil (x/tools
// defaults it): the CLI passes no deadline, in-process callers pass theirs so
// a hung `go list` cannot outlive it. The inherited GOFLAGS is sanitized
// (see loadEnv); explicit opts.Env entries are appended last and therefore
// win, so callers can always override.
func load(ctx context.Context, opts Options) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedDeps | packages.NeedImports |
			packages.NeedModule,
		Dir:     opts.Dir,
		Env:     append(loadEnv(), opts.Env...),
		Tests:   false, // composition roots are what dashboards see; DO-3 keeps Override* in tests
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	return pkgs, nil
}

// loadEnv builds the child-process env for package loading: the inherited
// environment with GOFLAGS stripped of -mod tokens. Neither inherited extreme
// is safe: -mod=vendor fails when the analyzed repo has no vendor directory,
// and -mod=mod is illegal in workspace mode. The go command deduplicates env
// with the LAST occurrence winning, so the sanitized entry is always present
// (empty when nothing was stripped) and later explicit entries shadow it.
func loadEnv() []string {
	inherited := os.Environ()
	env := make([]string, 0, len(inherited)+1)
	goFlags := ""

	for _, entry := range inherited {
		if key, value, ok := strings.Cut(entry, "="); ok && key == "GOFLAGS" {
			goFlags = value

			continue
		}

		env = append(env, entry)
	}

	return append(env, "GOFLAGS="+stripModTokens(goFlags))
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

// runAnalyzer builds an analysis.Pass by hand (the x/tools checker internals
// are internal) and runs the analyzer over one package. On the collecting
// sweep it yields diagnostics and HW-6 records; every sweep shares the
// factStore so NilBodyFacts cross package boundaries.
func runAnalyzer(
	analyzer *analysis.Analyzer, pkg *packages.Package, store *factStore, collect bool,
) (
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
		ImportObjectFact:  store.importInto,
		ExportObjectFact:  store.exportFrom,
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

	if collect {
		pass.Report = func(d analysis.Diagnostic) { diags = append(diags, d) }
		pass.ExportPackageFact = func(fact analysis.Fact) {
			if pf, ok := fact.(*healthwash.PackageFacts); ok {
				records = append(records, pf.Records...)
			}
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

// ruleCounts counts the post-allowlist findings per rule ID: the same set
// the confidence exit gate sees, so allowlist-accepted findings never
// ratchet.
func ruleCounts(findings []finding.Finding) map[string]int {
	counts := make(map[string]int, len(findings))
	for _, f := range findings {
		counts[string(f.Rule)]++
	}

	return counts
}

// humanPresentation reports whether stdout is carrying human UI (plain text
// or a human-oriented --output table). Machine consumers --json/--sarif and
// the structured formats get exactly one parseable shape: gate VERDICTS
// travel via the exit code and stderr, never as trailing stdout lines.
func (o Options) humanPresentation() bool {
	return !o.JSON && !o.SARIF && !IsMachineFormat(o.OutputFormat)
}

// reportCoverage prints the HW-6 coverage picture and applies its gates.
// Human-only lines (coverage, baseline acknowledgement, the --strict
// unresolved summary) are suppressed on machine presentations; failures
// always reach stderr and always flip gateFailed.
func reportCoverage(
	out io.Writer,
	findings []finding.Finding,
	records []healthwash.ServiceRecord,
	opts Options,
	gateFailed *bool,
	human bool,
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

	counts := ruleCounts(findings)

	baselinePath := opts.BaselinePath
	if opts.SetBaseline {
		writeBaseline(out, opts.Stderr, baselinePath, baseline{
			Version:    baselineSchemaVersion,
			Checked:    checked,
			Registered: registered,
			Coverage:   coverage,
			Findings:   counts,
		}, gateFailed, human)

		return coverage
	}

	// The gates compose, never short-circuit: --coverage-min is an absolute
	// floor, and a baseline file that exists is still validated and enforced
	// as a ratchet. A 2026-09-20 consumer incident (CV) sailed a stale v1
	// baseline through a -coverage-min gate because the early return here
	// skipped every baseline check — a gate that reads its own instrument
	// must also validate that instrument.
	if opts.CoverageMin > 0 {
		enforceCoverageMin(out, opts.Stderr, checked, registered, coverage, opts.CoverageMin, gateFailed, human)
	}

	enforceBaselineRatchet(out, opts.Stderr, baselinePath, counts, checked, registered, coverage, gateFailed, human)

	reportStrictUnresolved(out, records, opts, human)

	return coverage
}

// reportStrictUnresolved surfaces the silent-by-default unresolved
// registrations when --strict is on: recall the max-recall profile can see
// without a single new detection rule. Human presentations only — machines
// see HW-unresolved findings through the normal finding stream.
func reportStrictUnresolved(out io.Writer, records []healthwash.ServiceRecord, opts Options, human bool) {
	if !opts.Strict || !human {
		return
	}

	unresolved := 0
	for _, record := range records {
		if record.Unresolved {
			unresolved++
		}
	}

	if unresolved > 0 {
		fmt.Fprintf(out,
			"strict: %d registration(s) could not be resolved statically; analyze the concrete type or suppress with //samber-linter:allow hw-unresolved <reason>\n",
			unresolved)
	}
}

// writeBaseline persists the current coverage as the ratchet floor.
func writeBaseline(out, errw io.Writer, path string, b baseline, gateFailed *bool, human bool) {
	data, _ := json.Marshal(b, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if _, err := atomicwrite.WriteIfChanged(path, append(data, '\n')); err != nil {
		fmt.Fprintf(errw, "%s: baseline write failed: %v\n", ToolName, err)

		*gateFailed = true

		return
	}

	if human {
		fmt.Fprintf(out, "baseline written: %s (coverage %d/%d = %.0f%%)\n",
			path, b.Checked, b.Registered, b.Coverage*100)
	}
}

// enforceCoverageMin applies the absolute --coverage-min gate.
func enforceCoverageMin(
	out, errw io.Writer, checked, registered int, coverage, minCoverage float64,
	gateFailed *bool, human bool,
) {
	if human {
		fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (threshold: %.0f%%)\n",
			checked, registered, coverage*100, minCoverage*100)
	}

	if registered > 0 && coverage < minCoverage {
		fmt.Fprintf(errw, "%s: coverage %.0f%% is below the required minimum %.0f%%\n",
			ToolName, coverage*100, minCoverage*100)

		*gateFailed = true
	}
}

// enforceBaselineRatchet applies the committed-baseline regression gate:
// coverage must not drop and no rule may exceed its committed finding count.
// A missing baseline file only informs; a present-but-invalid one fails the
// gate loudly (fail closed — a ratchet that quietly stops reading is worse
// than a red build).
func enforceBaselineRatchet(
	out, errw io.Writer, baselinePath string, counts map[string]int,
	checked, registered int, coverage float64, gateFailed *bool, human bool,
) {
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		if registered > 0 && human {
			fmt.Fprintf(
				out,
				"health-coverage: %d/%d = %.0f%% (no baseline; use --set-baseline to start the ratchet)\n",
				checked,
				registered,
				coverage*100,
			)
		}

		return
	}

	var b baseline
	if err := json.Unmarshal(data, &b); err != nil {
		fmt.Fprintf(errw, "%s: baseline %s is not valid JSON: %v\n", ToolName, baselinePath, err)

		*gateFailed = true

		return
	}

	if b.Registered <= 0 {
		if registered > 0 && human {
			fmt.Fprintf(
				out,
				"health-coverage: %d/%d = %.0f%% (no baseline; use --set-baseline to start the ratchet)\n",
				checked,
				registered,
				coverage*100,
			)
		}

		return
	}

	if err := validateBaseline(b); err != nil {
		fmt.Fprintf(errw, "%s: baseline %s: %v\n", ToolName, baselinePath, err)

		*gateFailed = true

		return
	}

	if human {
		fmt.Fprintf(out, "health-coverage: %d/%d = %.0f%% (baseline: %.0f%%)\n",
			checked, registered, coverage*100, b.Coverage*100)
	}

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

	enforceRuleRatchet(errw, b.Findings, counts, gateFailed)
}

// Baseline validation failure classes. validateBaseline wraps these with the
// specific numbers, so callers can errors.Is the class while the stderr line
// stays precise.
var (
	ErrBaselineSchema  = errors.New("baseline schema version not supported")
	ErrBaselineCount   = errors.New("baseline service counters are inconsistent")
	ErrBaselineRules   = errors.New("baseline per-rule counts are malformed")
	ErrBaselineFloated = errors.New("baseline coverage is out of range")
)

// coverageRatioTolerance is the float slack between stored coverage and
// checked/registered: the file is tool-written, so anything beyond float
// rounding means it was hand-edited or corrupted.
const coverageRatioTolerance = 1e-9

// validateBaseline rejects baseline files this run cannot enforce honestly:
// wrong schema generation, negative or inconsistent counters, out-of-range
// coverage, or malformed per-rule counts.
func validateBaseline(b baseline) error {
	if err := validateBaselineSchema(b); err != nil {
		return err
	}

	if err := validateBaselineCounters(b); err != nil {
		return err
	}

	for rule, count := range b.Findings {
		if rule == "" {
			return fmt.Errorf("%w: findings map has an empty rule name", ErrBaselineRules)
		}

		if count < 0 {
			return fmt.Errorf("%w: findings[%q] = %d is negative", ErrBaselineRules, rule, count)
		}
	}

	return nil
}

// validateBaselineSchema rejects schema generations this binary cannot read
// or enforce: older files lack the per-rule floors, newer files may enforce
// rules this binary does not know.
func validateBaselineSchema(b baseline) error {
	switch {
	case b.Version < baselineSchemaVersion:
		return fmt.Errorf(
			"%w: v%d predates v%d; re-run --set-baseline to migrate (v2 adds per-rule finding counts)",
			ErrBaselineSchema, b.Version, baselineSchemaVersion)
	case b.Version > baselineSchemaVersion:
		return fmt.Errorf(
			"%w: v%d is newer than this tool supports (v%d); upgrade samber-linter",
			ErrBaselineSchema, b.Version, baselineSchemaVersion)
	}

	return nil
}

// validateBaselineCounters rejects impossible or hand-edited counter sets.
func validateBaselineCounters(b baseline) error {
	if b.Checked < 0 || b.Registered < 0 {
		return fmt.Errorf("%w: negative counts (checked=%d registered=%d)", ErrBaselineCount, b.Checked, b.Registered)
	}

	if b.Checked > b.Registered {
		return fmt.Errorf("%w: checked %d exceeds registered %d", ErrBaselineCount, b.Checked, b.Registered)
	}

	if b.Coverage < 0 || b.Coverage > 1 {
		return fmt.Errorf("%w: coverage %v outside [0,1]", ErrBaselineFloated, b.Coverage)
	}

	if b.Registered > 0 {
		want := float64(b.Checked) / float64(b.Registered)
		if math.Abs(b.Coverage-want) > coverageRatioTolerance {
			return fmt.Errorf(
				"%w: coverage %v does not match checked/registered = %d/%d = %v; the file is corrupt or hand-edited",
				ErrBaselineFloated, b.Coverage, b.Checked, b.Registered, want)
		}
	}

	return nil
}

// enforceRuleRatchet fails the gate when any rule produces more findings than
// its committed count: aggregate coverage can stay flat while a single rule
// regresses. Improvements pass silently; --set-baseline locks them in.
func enforceRuleRatchet(errw io.Writer, base, current map[string]int, gateFailed *bool) {
	for _, rule := range slices.Sorted(maps.Keys(current)) {
		if current[rule] <= base[rule] {
			continue
		}

		fmt.Fprintf(
			errw,
			"%s: %s findings %d exceed the committed baseline %d; "+
				"fix the regressions or explicitly re-baseline with --set-baseline\n",
			ToolName,
			rule,
			current[rule],
			base[rule],
		)

		*gateFailed = true
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
