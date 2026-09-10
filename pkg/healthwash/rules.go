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

	var serviceType types.Type
	var unresolved bool
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
				message: fmt.Sprintf("%s: %s at %s; analyze the concrete type or suppress with //samber-linter:allow %s <reason>",
					RuleUnresolved, MessageUnresolv, orDash(rel), RuleCodeUnres),
				pos: call.Pos(),
			})
		}
		return rec, reps
	}

	stored := serviceType // as registered: T (value) or *T (pointer)
	valueReg := !isPointer(stored)
	ptrToBase := types.NewPointer(stored) // for value regs this is *T

	anyCheckStored := lc.typeImplementsAnyCheck(stored)
	bareCheckStored := lc.typeImplementsBareCheck(stored)
	ctxCheckStored := lc.typeImplementsCtxCheck(stored)
	anyCheckPtr := lc.typeImplementsAnyCheck(ptrToBase)
	shutdownStored := lc.typeImplementsAnyShutdown(stored)

	rec.ImplementsCheck = anyCheckStored

	var reps []siteReport
	add := func(rule, msg string) {
		reps = append(reps, siteReport{rule: rule, message: rule + ": " + msg, pos: call.Pos()})
	}

	switch kind {
	case KindTransient:
		if anyCheckStored {
			add(RuleHW3, fmt.Sprintf(
				"%s implements a Healthchecker variant but is registered transiently (%s); the transient healthcheck is an upstream TODO and always returns nil, so the check can never execute. Register as a singleton or drop the dead implementation. Suppress with //samber-linter:allow %s <reason>",
				rel, fnName, RuleCodeHW3))
		}
		if bareCheckStored && !ctxCheckStored {
			add(RuleHW2, fmt.Sprintf(
				"%s implements HealthCheck() without a context variant; a hung bare check cannot be cancelled and degrades the whole sweep. Prefer HealthCheck(context.Context) error. Suppress with //samber-linter:allow %s <reason>",
				rel, RuleCodeHW2))
		}
	case KindLazy, KindEager:
		if valueReg && anyCheckPtr && !anyCheckStored {
			add(RuleHW5, fmt.Sprintf(
				"%s declares its health check on receiver *T but is registered as value %s; the sweep type-asserts the stored value, so the implementation exists and never runs. Register the pointer or move the receiver to T. Suppress with //samber-linter:allow %s <reason>",
				rel, rel, RuleCodeHW5))
			return rec, reps
		}
		if shutdownStored && !anyCheckStored {
			add(RuleHW1, fmt.Sprintf(
				"%s implements do.Shutdowner but no Healthchecker; it renders an unconditional \"pass\" on health dashboards. Implement HealthCheck(context.Context) error or suppress with a reason: //samber-linter:allow %s <reason>",
				rel, RuleCodeHW1))
		}
		if anyCheckStored {
			if kind == KindLazy {
				add(RuleHW4, fmt.Sprintf(
					"%s implements a Healthchecker variant but is registered lazily (%s); until first resolution it reports green without ever having been constructed. Register eagerly when boot-critical or suppress with a reason: //samber-linter:allow %s <reason>",
					rel, fnName, RuleCodeHW4))
			}
			if bareCheckStored && !ctxCheckStored {
				add(RuleHW2, fmt.Sprintf(
					"%s implements HealthCheck() without a context variant; a hung bare check cannot be cancelled and degrades the whole sweep. Prefer HealthCheck(context.Context) error. Suppress with //samber-linter:allow %s <reason>",
					rel, RuleCodeHW2))
			}
		}
	}

	return rec, reps
}

// evalAlias records As/AsNamed rows for HW-6. Aliases delegate their
// healthcheck to the target wrapper (service_alias.go:114-126), so they are
// never attributed and never counted in the ratchet denominator.
func evalAlias(pass *analysis.Pass, call *ast.CallExpr) ServiceRecord {
	name := ""
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if inst, ok2 := pass.TypesInfo.Instances[sel.Sel]; ok2 && inst.TypeArgs != nil && inst.TypeArgs.Len() >= 2 {
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
