package healthwash

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"time"

	"golang.org/x/tools/go/analysis"
)

// DoPath is the samber/do v2 package path. Matching is by package path,
// never identifier text — resilient to dot-imports and renames.
const DoPath = "github.com/samber/do/v2"

// Rule codes. Stable forever once shipped.
const (
	RuleHW0        = "HW-0"
	RuleHW1        = "HW-1"
	RuleHW2        = "HW-2"
	RuleHW3        = "HW-3"
	RuleHW4        = "HW-4"
	RuleHW5        = "HW-5"
	RuleUnresolved = "HW-unresolved"

	RuleCodeHW0     = "hw-0"
	RuleCodeHW1     = "hw-1"
	RuleCodeHW2     = "hw-2"
	RuleCodeHW3     = "hw-3"
	RuleCodeHW4     = "hw-4"
	RuleCodeHW5     = "hw-5"
	RuleCodeUnres   = "hw-unresolved"
	RuleCodeAll     = "all"
	RuleMessageHW0  = "suppression directive without a reason; unexplained suppressions rot into permanent darkness"
	MessageUnresolv = "service type could not be resolved statically"
)

// regKindOf maps a samber/do registration function name to the wrapper kind
// its call creates (verified against samber/do v2.1.0 di.go:57-251).
func regKindOf(fn string) (ServiceKind, bool) {
	switch fn {
	case "Provide", "ProvideNamed", "Override", "OverrideNamed":
		return KindLazy, true
	case "ProvideValue", "ProvideNamedValue", "OverrideValue", "OverrideNamedValue":
		return KindEager, true
	case "ProvideTransient", "ProvideNamedTransient", "OverrideTransient", "OverrideNamedTransient":
		return KindTransient, true
	case "As", "AsNamed":
		return KindAlias, true
	}

	return "", false
}

// providerArgIndex is the index of the provider/value argument within the
// registration call (0-based).
func providerArgIndex(fn string) (int, bool) {
	switch fn {
	case "Provide", "Override", "ProvideValue", "OverrideValue",
		"ProvideTransient", "OverrideTransient":
		return 1, true
	case "ProvideNamed", "OverrideNamed", "ProvideNamedValue", "OverrideNamedValue",
		"ProvideNamedTransient", "OverrideNamedTransient":
		return 2, true
	}

	return 0, false
}

// New returns the samber-linter analyzer. It is a plain *analysis.Analyzer so
// it doubles as the golangci-lint plugin contract; the CLI driver layers
// findings, confidence, suppression reporting, and the HW-6 ratchet on top.
func New() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "healthwash",
		Doc: "detects health-washing in samber/do v2 containers: services that " +
			"render green 'pass' on health dashboards but cannot actually fail",
		URL:       "https://github.com/LarsArtmann/samber-linter",
		Run:       run,
		FactTypes: []analysis.Fact{(*PackageFacts)(nil)},
	}
	a.Flags.Bool(
		"strict",
		false,
		"report HW-unresolved for unresolvable service types instead of staying silent",
	)
	a.Flags.String(
		"disable",
		"",
		"comma-separated rule IDs to skip (e.g. HW-1,HW-4) — testing/migration aid",
	)

	return a
}

type siteReport struct {
	rule    string // "HW-1" …
	message string
	pos     token.Pos
	// endLine is the last source line of the registration call. Suppression
	// directives may sit anywhere in that span: the common style puts the
	// directive on the line above the call, but long provider closures often
	// carry it inside the argument list.
	endLine int
}

func run(pass *analysis.Pass) (any, error) {
	doPkg := findDoPackage(pass)
	if doPkg == nil {
		return nil, nil // target does not use samber/do v2
	}

	ifaces := loadInterfaces(doPkg)
	strict := pass.Analyzer.Flags.Lookup("strict") != nil &&
		pass.Analyzer.Flags.Lookup("strict").Value.String() == "true"

	directives := collectDirectives(pass.Fset, pass.Files)
	disabled := parseDisabledRules(pass)

	var (
		records []ServiceRecord
		reports []siteReport
	)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			rec, reps, isReg := inspectRegistration(pass, n, ifaces, strict)
			if isReg {
				records = append(records, rec)
				reports = append(reports, reps...)
			}

			return true
		})
	}

	reportedDirectives := reportFindings(pass, reports, directives, disabled)
	reportOrphanedDirectives(pass, directives, reportedDirectives)

	pass.ExportPackageFact(&PackageFacts{Records: records})

	return nil, nil
}

// parseDisabledRules reads the analyzer's disable flag into a lookup set.
func parseDisabledRules(pass *analysis.Pass) map[string]bool {
	disabled := map[string]bool{}

	f := pass.Analyzer.Flags.Lookup("disable")
	if f == nil {
		return disabled
	}

	for r := range strings.SplitSeq(f.Value.String(), ",") {
		r = strings.ToUpper(strings.TrimSpace(r))
		if r != "" {
			disabled[r] = true
		}
	}

	return disabled
}

// inspectRegistration classifies one AST node; isReg is true only when the
// node is a samber/do registration call.
func inspectRegistration(
	pass *analysis.Pass, n ast.Node, doIfaces ifaces, strict bool,
) (ServiceRecord, []siteReport, bool) {
	call, isCall := n.(*ast.CallExpr)
	if !isCall {
		return ServiceRecord{}, nil, false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return ServiceRecord{}, nil, false
	}

	fn, isFn := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !isFn || fn.Pkg() == nil || fn.Pkg().Path() != DoPath {
		return ServiceRecord{}, nil, false
	}

	kind, known := regKindOf(fn.Name())
	if !known {
		return ServiceRecord{}, nil, false
	}

	rec, reps := evalSite(pass, fn.Name(), kind, call, doIfaces, strict)

	return rec, reps, true
}

// reportFindings applies suppression directives (honored anywhere in the
// site's line span, or on the line above) and the disable list, then reports
// the survivors. It returns the set of directive positions it already
// reported as HW-0 so the orphan scan does not double-report them.
func reportFindings(
	pass *analysis.Pass, reports []siteReport,
	directives map[dirKey][]foundDirective, disabled map[string]bool,
) map[token.Pos]bool {
	reportedSites := map[token.Pos]bool{}
	reportedDirectives := map[token.Pos]bool{}

	reportHW0 := func(pos token.Pos) {
		pass.Report(analysis.Diagnostic{
			Pos:      pos,
			Category: RuleHW0,
			Message:  fmt.Sprintf("%s: %s", RuleHW0, RuleMessageHW0),
		})
	}

	for _, site := range reports {
		pos := pass.Fset.Position(site.pos)
		suppressed := false

		for line := pos.Line - 1; line <= site.endLine; line++ {
			for _, directive := range directives[dirKey{pos.Filename, line}] {
				if directive.invalid {
					// A suppression without a reason is itself a finding,
					// attributed to the suppressible site (where the fix
					// lands). It never suppresses — and it stays reported
					// even once the underlying violation is gone (the
					// orphan scan in reportOrphanedDirectives).
					if !reportedSites[site.pos] {
						reportedSites[site.pos] = true
						reportedDirectives[directive.pos] = true
						reportHW0(site.pos)
					}

					continue
				}

				if !directive.expired && matchesRule(directive.rule, site.rule) {
					suppressed = true
				}
			}
		}

		if suppressed || disabled[strings.ToUpper(site.rule)] {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Pos:      site.pos,
			Category: site.rule,
			Message:  site.message,
		})
	}

	return reportedDirectives
}

// reportOrphanedDirectives surfaces malformed directives that no finding
// referenced, at the comment itself: unexplained suppressions rot into
// permanent darkness even after the underlying violation is fixed.
func reportOrphanedDirectives(
	pass *analysis.Pass, directives map[dirKey][]foundDirective, alreadyReported map[token.Pos]bool,
) {
	for _, found := range directives {
		for _, directive := range found {
			if directive.invalid && !alreadyReported[directive.pos] {
				pass.Report(analysis.Diagnostic{
					Pos:      directive.pos,
					Category: RuleHW0,
					Message:  fmt.Sprintf("%s: %s", RuleHW0, RuleMessageHW0),
				})
			}
		}
	}
}

type dirKey struct {
	file string
	line int
}

type foundDirective struct {
	directive

	pos token.Pos
}

// directive is the analyzer-side view; invalid marks a malformed directive
// (missing rule token or missing reason), which must itself trigger HW-0.
// expired marks a directive whose `until` re-review deadline has passed: it
// no longer suppresses (the finding resurfaces for re-review).
type directive struct {
	rule    string // lowercased token, "" when missing
	reason  string
	invalid bool
	expired bool
}

func matchesRule(token, rule string) bool {
	tok := strings.ToLower(strings.TrimSpace(token))
	if tok == RuleCodeAll {
		return true
	}

	return tok == strings.ToLower(rule)
}

func collectDirectives(fset *token.FileSet, files []*ast.File) map[dirKey][]foundDirective {
	out := map[dirKey][]foundDirective{}

	for _, f := range files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				d, ok := parseDirectiveComment(c.Text)
				if !ok {
					continue
				}

				pos := fset.Position(c.Pos())
				k := dirKey{pos.Filename, pos.Line}
				out[k] = append(out[k], foundDirective{directive: d, pos: c.Pos()})
			}
		}
	}

	return out
}

func parseDirectiveComment(text string) (directive, bool) {
	if !strings.Contains(text, DirectivePrefix) {
		return directive{}, false
	}

	d, ok := ParseDirective(text)
	if !ok {
		return directive{}, false
	}

	res := directive{rule: d.Rule, reason: d.Reason}
	if d.Rule == "" || d.Reason == "" {
		res.invalid = true
	}

	if d.Expires != nil && d.Expired(time.Now()) {
		res.expired = true
	}

	return res, true
}
