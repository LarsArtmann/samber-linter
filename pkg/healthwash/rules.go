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
//	HW-2  bare check: uncancellable, degrades the sweep              (info)
func evalSite(
	pass *analysis.Pass,
	fnName string,
	kind ServiceKind,
	call *ast.CallExpr,
	lc ifaces,
	strict bool,
) (ServiceRecord, []siteReport) {
	if kind == KindAlias {
		return evalAlias(pass, call), nil
	}

	idx, ok := providerArgIndex(fnName)
	if !ok || idx >= len(call.Args) {
		return ServiceRecord{Kind: kind}, nil
	}

	var (
		serviceType types.Type
		unresolved  bool
	)

	if kind == KindEager {
		serviceType = pass.TypesInfo.TypeOf(call.Args[idx])
		if serviceType == nil {
			unresolved = true
		}
	} else {
		pt := pass.TypesInfo.TypeOf(call.Args[idx])
		if pt == nil {
			unresolved = true
		} else if sig, ok := pt.Underlying().(*types.Signature); ok &&
			sig.Results() != nil && sig.Results().Len() >= 1 {
			serviceType = sig.Results().At(0).Type()
		} else {
			unresolved = true
		}
	}

	if serviceType != nil {
		if _, isIface := serviceType.Underlying().(*types.Interface); isIface {
			// Interface-typed closure result: the sweep asserts the stored
			// concrete instance, which is statically unknowable here.
			unresolved = true
		}
	}

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

	facts := typeFacts{
		anyCheck:   lc.typeImplementsAnyCheck(stored),
		bareCheck:  lc.typeImplementsBareCheck(stored),
		ctxCheck:   lc.typeImplementsCtxCheck(stored),
		anyCheckP:  lc.typeImplementsAnyCheck(ptrToBase),
		shutdown:   lc.typeImplementsAnyShutdown(stored),
		valueReg:   valueReg,
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

// typeFacts is what the sweep can see about the stored instance. All fields
// derive from interface satisfaction of the stored type (and its pointer for
// value registrations).
type typeFacts struct {
	anyCheck  bool // any Healthchecker variant on the stored type
	bareCheck bool // bare HealthCheck() on the stored type
	ctxCheck  bool // HealthCheck(context.Context) on the stored type
	anyCheckP bool // any Healthchecker variant on *T (value registrations)
	shutdown  bool // any Shutdowner variant on the stored type
	valueReg  bool // registered as a value, not a pointer
}

// reportTransientRules fires for transient registrations: the upstream
// transient healthcheck is a TODO that always returns nil, so a stored check
// can never execute (HW-3); a bare check would degrade the sweep if the
// upstream ever dispatches (HW-2).
func reportTransientRules(add func(rule, msg string), rel, fnName string, f typeFacts) {
	if f.anyCheck {
		add(RuleHW3, fmt.Sprintf(
			"%s implements a Healthchecker variant but is registered transiently (%s); the transient healthcheck is an upstream TODO and always returns nil, so the check can never execute. Register as a singleton or drop the dead implementation. Suppress with //samber-linter:allow %s <reason>",
			rel,
			fnName,
			RuleCodeHW3,
		))
	}

	if f.bareCheck && !f.ctxCheck {
		addBareCheckRule(add, rel)
	}
}

// reportSweepRules fires for lazy and eager registrations in precedence
// order. HW-5 short-circuits: it and HW-1 share the single remedy (register
// the pointer), so reporting both would double-count one fixable cause.
func reportSweepRules(add func(rule, msg string), rel, fnName string, kind ServiceKind, f typeFacts) {
	if f.valueReg && f.anyCheckP && !f.anyCheck {
		add(RuleHW5, fmt.Sprintf(
			"%s declares its health check on receiver *T but is registered as value %s; the sweep type-asserts the stored value, so the implementation exists and never runs. Register the pointer or move the receiver to T. Suppress with //samber-linter:allow %s <reason>",
			rel,
			rel,
			RuleCodeHW5,
		))

		return
	}

	if f.shutdown && !f.anyCheck {
		add(RuleHW1, fmt.Sprintf(
			"%s implements do.Shutdowner but no Healthchecker; it renders an unconditional %q on health dashboards. Implement HealthCheck(context.Context) error or suppress with a reason: //samber-linter:allow %s <reason>",
			rel,
			"pass",
			RuleCodeHW1,
		))
	}

	if f.anyCheck {
		if kind == KindLazy {
			add(RuleHW4, fmt.Sprintf(
				"%s implements a Healthchecker variant but is registered lazily (%s); until first resolution it reports green without ever having been constructed. Register eagerly when boot-critical or suppress with a reason: //samber-linter:allow %s <reason>",
				rel,
				fnName,
				RuleCodeHW4,
			))
		}

		if f.bareCheck && !f.ctxCheck {
			addBareCheckRule(add, rel)
		}
	}
}

// addBareCheckRule is the shared HW-2 message for bare checks.
func addBareCheckRule(add func(rule, msg string), rel string) {
	add(RuleHW2, fmt.Sprintf(
		"%s implements HealthCheck() without a context variant; a hung bare check cannot be cancelled and degrades the whole sweep. Prefer HealthCheck(context.Context) error. Suppress with //samber-linter:allow %s <reason>",
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

func orDash(s string) string {
	if s == "" {
		return "<unknown>"
	}

	return s
}
