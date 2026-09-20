package healthwash

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// evalSite classifies one registration call site. It returns the HW-6 record
// for the site plus any rule violations found there.
//
// Rule precedence, all evaluated on the STORED instance (the value the sweep
// actually type-asserts — service_eager.go:88,94, service_lazy.go:136):
//
//	HW-5  registration/value mismatch: check reachable only on *T   (warn)
//	HW-3  transient wrapper: the check can never execute             (warn)
//	HW-1  shutdown-without-check: green by construction              (warn)
//	HW-4  lazy wrapper: green until first resolution                 (info)
//	HW-7  check body is exactly `return nil`: green forever          (warn)
//	HW-2  bare check: uncancellable, degrades the sweep              (info)
func evalSite(
	pass *analysis.Pass,
	fnName string,
	kind ServiceKind,
	call *ast.CallExpr,
	doIfaces ifaces,
	strict bool,
	methodDecls map[*types.Func]*ast.FuncDecl,
) (ServiceRecord, []siteReport) {
	if kind == KindAlias {
		return evalAlias(pass, call), nil
	}

	idx, ok := providerArgIndex(fnName)
	if !ok || idx >= len(call.Args) {
		return ServiceRecord{Kind: kind}, nil
	}

	serviceType, unresolved := resolveStoredType(pass, kind, call.Args[idx])

	rel := types.TypeString(serviceType, types.RelativeTo(pass.Pkg))
	full := types.TypeString(serviceType, nil)
	endLine := pass.Fset.Position(call.End()).Line
	rec := ServiceRecord{
		Name: full,
		Type: rel,
		Kind: kind,
	}

	if unresolved || serviceType == nil {
		rec.Unresolved = true
		rec.Name = rec.Type

		var reps []siteReport
		if strict {
			reps = append(reps, siteReport{
				rule: RuleUnresolved,
				message: fmt.Sprintf(
					"%s: %s at %s; analyze the concrete type or suppress with //samber-linter:allow %s <reason>",
					RuleUnresolved,
					MessageUnresolv,
					orDash(rel),
					RuleCodeUnres,
				),
				pos:     call.Pos(),
				endLine: endLine,
			})
		}

		return rec, reps
	}

	stored := serviceType // as registered: T (value) or *T (pointer)
	valueReg := !isPointer(stored)
	ptrToBase := types.NewPointer(stored) // for value regs this is *T

	// HW-7/HW-8 read the reachable check's body, which only exists for
	// sweep-dispatched kinds: transients never dispatch (HW-3 owns the whole
	// story) and aliases delegate to their target.
	var nilBody, emptyBody bool
	if kind == KindLazy || kind == KindEager {
		nilBody = hasSoleNilReturnCheck(pass, stored, methodDecls)
		emptyBody = hasNakedReturnCheck(pass, stored, methodDecls)
	}

	facts := typeFacts{
		anyCheck:  doIfaces.typeImplementsAnyCheck(stored),
		bareCheck: doIfaces.typeImplementsBareCheck(stored),
		ctxCheck:  doIfaces.typeImplementsCtxCheck(stored),
		anyCheckP: doIfaces.typeImplementsAnyCheck(ptrToBase),
		shutdown:  doIfaces.typeImplementsAnyShutdown(stored),
		valueReg:  valueReg,
		nilBody:   nilBody,
		emptyBody: emptyBody,
	}

	rec.ImplementsCheck = facts.anyCheck

	var reps []siteReport

	add := func(rule, msg string) {
		reps = append(
			reps,
			siteReport{rule: rule, message: rule + ": " + msg, pos: call.Pos(), endLine: endLine},
		)
	}

	// KindAlias never reaches the switch body: evalSite handles it before
	// provider-argument resolution, and alias rows carry no rule findings.
	switch kind {
	case KindAlias:
		// defensive: kept exhaustive for future ServiceKind values.

	case KindTransient:
		reportTransientRules(add, rel, fnName, facts)
	case KindLazy, KindEager:
		reportSweepRules(add, rel, fnName, kind, facts)

		return rec, reps
	}

	return rec, reps
}

// resolveStoredType extracts the type the sweep will store for one
// registration argument: the value itself for eager registrations, the
// provider's first result otherwise. unresolved is true when that type is not
// statically knowable (nil type, result-less provider, or an interface-typed
// closure result whose concrete instance the analyzer cannot see).
func resolveStoredType(
	pass *analysis.Pass, kind ServiceKind, arg ast.Expr,
) (types.Type, bool) {
	if kind == KindEager {
		serviceType := pass.TypesInfo.TypeOf(arg)

		return serviceType, serviceType == nil
	}

	pt := pass.TypesInfo.TypeOf(arg)
	if pt == nil {
		return nil, true
	}

	sig, isSig := pt.Underlying().(*types.Signature)
	if !isSig || sig.Results() == nil || sig.Results().Len() < 1 {
		return nil, true
	}

	serviceType := sig.Results().At(0).Type()
	if _, isIface := serviceType.Underlying().(*types.Interface); isIface {
		// Interface-typed closure result: the sweep asserts the stored
		// concrete instance, which is statically unknowable here.
		return nil, true
	}

	return serviceType, false
}

// typeFacts is what the sweep can see about the stored instance. All fields
// derive from interface satisfaction of the stored type (and its pointer for
// value registrations); nilBody/emptyBody are the syntactic facts (HW-7/HW-8).
type typeFacts struct {
	anyCheck  bool // any Healthchecker variant on the stored type
	bareCheck bool // bare HealthCheck() on the stored type
	ctxCheck  bool // HealthCheck(context.Context) on the stored type
	anyCheckP bool // any Healthchecker variant on *T (value registrations)
	shutdown  bool // any Shutdowner variant on the stored type
	valueReg  bool // registered as a value, not a pointer
	nilBody   bool // the reachable check body is exactly `return nil`
	emptyBody bool // the reachable check body is a single naked return
}

// reportTransientRules fires for transient registrations: the upstream
// transient healthcheck is a TODO that always returns nil, so a stored check
// can never execute (HW-3); a bare check would degrade the sweep if the
// upstream ever dispatches (HW-2).
func reportTransientRules(add func(rule, msg string), rel, fnName string, facts typeFacts) {
	if facts.anyCheck {
		add(RuleHW3, fmt.Sprintf(
			"%s implements a Healthchecker variant but is registered transiently (%s); "+
				"the transient healthcheck is an upstream TODO and always returns nil, so the check can never execute. "+
				"Register as a singleton or drop the dead implementation. "+
				"Suppress with //samber-linter:allow %s <reason>",
			rel,
			fnName,
			RuleCodeHW3,
		))
	}

	if facts.bareCheck && !facts.ctxCheck {
		addBareCheckRule(add, rel)
	}
}

// reportSweepRules fires for lazy and eager registrations in precedence
// order. HW-5 short-circuits: it and HW-1 share the single remedy (register
// the pointer), so reporting both would double-count one fixable cause.
func reportSweepRules(add func(rule, msg string), rel, fnName string, kind ServiceKind, facts typeFacts) {
	if facts.valueReg && facts.anyCheckP && !facts.anyCheck {
		add(RuleHW5, fmt.Sprintf(
			"%s declares its health check on receiver *T but is registered as value %s; "+
				"the sweep type-asserts the stored value, so the implementation exists and never runs. "+
				"Register the pointer or move the receiver to T. "+
				"Suppress with //samber-linter:allow %s <reason>",
			rel,
			rel,
			RuleCodeHW5,
		))

		return
	}

	if facts.shutdown && !facts.anyCheck {
		add(RuleHW1, fmt.Sprintf(
			"%s implements do.Shutdowner but no Healthchecker; "+
				"it renders an unconditional %q on health dashboards. "+
				"Implement HealthCheck(context.Context) error or suppress with a reason: "+
				"//samber-linter:allow %s <reason>",
			rel,
			"pass",
			RuleCodeHW1,
		))
	}

	if facts.anyCheck {
		if kind == KindLazy {
			add(RuleHW4, fmt.Sprintf(
				"%s implements a Healthchecker variant but is registered lazily (%s); "+
					"until first resolution it reports green without ever having been constructed. "+
					"Register eagerly when boot-critical or suppress with a reason: "+
					"//samber-linter:allow %s <reason>",
				rel,
				fnName,
				RuleCodeHW4,
			))
		}

		if facts.bareCheck && !facts.ctxCheck {
			addBareCheckRule(add, rel)
		}

		if facts.nilBody {
			add(RuleHW7, fmt.Sprintf(
				"%s's health check body is exactly \"return nil\"; "+
					"the check can never fail and always renders \"pass\" on health dashboards. "+
					"Implement a real check or suppress with a reason: "+
					"//samber-linter:allow %s <reason>",
				rel,
				RuleCodeHW7,
			))
		}

		if facts.emptyBody {
			add(RuleHW8, fmt.Sprintf(
				"%s's health check body is empty (named result, implicit nil); "+
					"the check can never fail and always renders \"pass\" on health dashboards. "+
					"Implement a real check or suppress with a reason: "+
					"//samber-linter:allow %s <reason>",
				rel,
				RuleCodeHW8,
			))
		}
	}
}

// addBareCheckRule is the shared HW-2 message for bare checks.
func addBareCheckRule(add func(rule, msg string), rel string) {
	add(RuleHW2, fmt.Sprintf(
		"%s implements HealthCheck() without a context variant; "+
			"a hung bare check cannot be cancelled and degrades the whole sweep. "+
			"Prefer HealthCheck(context.Context) error. "+
			"Suppress with //samber-linter:allow %s <reason>",
		rel,
		RuleCodeHW2,
	))
}

// evalAlias records As/AsNamed rows for HW-6. Aliases delegate their
// healthcheck to the target wrapper (service_alias.go:114-126), so they are
// never attributed and never counted in the ratchet denominator.
func evalAlias(pass *analysis.Pass, call *ast.CallExpr) ServiceRecord {
	name := ""

	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if inst, ok2 := pass.TypesInfo.Instances[sel.Sel]; ok2 && inst.TypeArgs != nil &&
			inst.TypeArgs.Len() >= 2 {
			name = types.TypeString(inst.TypeArgs.At(1), nil)
		}
	}

	return ServiceRecord{Name: name, Type: name, Kind: KindAlias}
}

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)

	return ok
}

// collectMethodDecls maps every method declared in the analyzed package to
// its AST declaration, keyed by the checker's method object. HW-7 resolves
// the stored type's HealthCheck through this map; a check declared in another
// package has no entry (the body is not visible from here) and is silently
// skipped — the same doctrine as unresolvable registrations.
func collectMethodDecls(pass *analysis.Pass) map[*types.Func]*ast.FuncDecl {
	out := map[*types.Func]*ast.FuncDecl{}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}

			obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			if !ok {
				continue
			}

			out[obj] = fn
		}
	}

	return out
}

// hasSoleNilReturnCheck is HW-7's predicate: the stored type's reachable
// HealthCheck method is declared in this package and its body's only
// statement is `return nil`. The method is resolved by interface-method
// lookup (promoted methods included) — never by name heuristics on the AST.
func hasSoleNilReturnCheck(
	pass *analysis.Pass, stored types.Type, methodDecls map[*types.Func]*ast.FuncDecl,
) bool {
	named := baseNamed(stored)
	if named == nil {
		return false
	}

	// addressable=true: for a stored *T the method set includes both receiver
	// forms; for a stored T the reachable check is a value-receiver method.
	obj, _, _ := types.LookupFieldOrMethod(named, true, pass.Pkg, "HealthCheck")

	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	if decl, declared := methodDecls[fn]; declared {
		return isSoleNilReturn(decl.Body)
	}

	// Declared in another package: that package exported the verdict when it
	// was analyzed. Objects are identical across the package graph (one
	// type-checker pass over the source), so the lookup keys match.
	var nilBody NilBodyFact

	return pass.ImportObjectFact(fn, &nilBody)
}

// hasNakedReturnCheck is HW-8's predicate: the stored type's reachable
// HealthCheck method's body is a single naked `return` on a named result —
// the implicit zero value cannot fail. Same-package bodies are read directly;
// foreign ones come in through NakedReturnFact, mirroring HW-7.
func hasNakedReturnCheck(
	pass *analysis.Pass, stored types.Type, methodDecls map[*types.Func]*ast.FuncDecl,
) bool {
	named := baseNamed(stored)
	if named == nil {
		return false
	}

	obj, _, _ := types.LookupFieldOrMethod(named, true, pass.Pkg, "HealthCheck")

	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	if decl, declared := methodDecls[fn]; declared {
		return isSoleNakedReturn(decl.Body)
	}

	var nakedReturn NakedReturnFact

	return pass.ImportObjectFact(fn, &nakedReturn)
}

// isSoleNakedReturn reports whether the block's only statement is a return
// with no results. That compiles only with named results, whose implicit zero
// value is nil: the check cannot fail. A naked return among other statements
// is NOT this — err may have been set.
func isSoleNakedReturn(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}

	ret, ok := body.List[0].(*ast.ReturnStmt)

	return ok && len(ret.Results) == 0
}

// baseNamed unwraps a stored type to its named base: *T → T. Method sets and
// declarations live on the named type.
func baseNamed(t types.Type) *types.Named {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}

	named, _ := t.(*types.Named)

	return named
}

// isSoleNilReturn reports whether the block's only statement is a single
// `return nil`. Comments are ignored (a commented body is still nil); naked
// returns on named results and multi-result returns are not HW-7 — they can
// carry a real error value.
func isSoleNilReturn(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}

	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}

	nilIdent, ok := ret.Results[0].(*ast.Ident)

	return ok && nilIdent.Name == "nil"
}

func orDash(s string) string {
	if s == "" {
		return "<unknown>"
	}

	return s
}
