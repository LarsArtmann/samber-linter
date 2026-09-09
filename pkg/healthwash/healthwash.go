package healthwash

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

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
		FactTypes: []analysis.Fact{PackageFacts{}},
	}
	a.Flags.Bool("strict", false, "report HW-unresolved for unresolvable service types instead of staying silent")
	return a
}

type siteReport struct {
	rule    string // "HW-1" …
	message string
	pos     token.Pos
}

func run(pass *analysis.Pass) (interface{}, error) {
	doPkg := findDoPackage(pass)
	if doPkg == nil {
		return nil, nil // target does not use samber/do v2
	}
	ifaces := loadInterfaces(doPkg)
	strict := pass.Analyzer.Flags.Lookup("strict") != nil &&
		pass.Analyzer.Flags.Lookup("strict").Value.String() == "true"

	directives := collectDirectives(pass.Fset, pass.Files)

	var records []ServiceRecord
	var reports []siteReport

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			obj := pass.TypesInfo.Uses[sel.Sel]
			fn, ok := obj.(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != DoPath {
				return true
			}
			kind, known := regKindOf(fn.Name())
			if !known {
				return true
			}

			rec, reps := evalSite(pass, fn.Name(), kind, call, ifaces, strict)
			records = append(records, rec)
			reports = append(reports, reps...)
			return true
		})
	}

	for _, d := range reports {
		pos := pass.Fset.Position(d.pos)
		suppressed := false
		for _, line := range []int{pos.Line - 1, pos.Line} {
			for _, fd := range directives[dirKey{pos.Filename, line}] {
				if fd.invalid {
					pass.Report(analysis.Diagnostic{
						Pos:      fd.pos,
						Category: RuleHW0,
						Message:  fmt.Sprintf("%s: %s", RuleHW0, RuleMessageHW0),
					})
					continue
				}
				if matchesRule(fd.rule, d.rule) {
					suppressed = true
				}
			}
		}
		if suppressed {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos:      d.pos,
			Category: d.rule,
			Message:  d.message,
		})
	}

	pass.ExportPackageFact(PackageFacts{Records: records})
	return nil, nil
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
type directive struct {
	rule    string // lowercased token, "" when missing
	reason  string
	invalid bool
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
	return res, true
}
