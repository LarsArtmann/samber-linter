package healthwash

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// doRegistrationCall classifies one call node as a direct samber/do
// registration: a selector whose callee is a known registration function of
// the do package (package-path match, resilient to dot-imports and renames).
func doRegistrationCall(pass *analysis.Pass, call *ast.CallExpr) (string, ServiceKind, bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return "", "", false
	}

	fn, isFn := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !isFn || fn.Pkg() == nil || fn.Pkg().Path() != DoPath {
		return "", "", false
	}

	kind, known := regKindOf(fn.Name())
	if !known {
		return "", "", false
	}

	return fn.Name(), kind, true
}

// wrapperInfo describes one resolvable repo-local wrapper: a package-level
// function whose body performs exactly one samber/do registration with a
// bare parameter as its provider/value argument.
type wrapperInfo struct {
	innerFnName string      // the wrapped do.* function ("ProvideNamed")
	kind        ServiceKind // the wrapper kind that call creates
	paramIdx    int         // which wrapper parameter feeds the provider slot
}

// collectWrappers indexes every resolvable wrapper in the package and returns
// the set of registration call nodes that live inside those wrapper bodies.
// Those body calls are parameterized — never concrete registrations — so they
// must not be evaluated or counted on their own: their concrete meaning is
// attributed at each call site instead. Bodies with zero or multiple
// registrations, non-parameter providers (a plain delegating helper's call
// stays concrete and is reported directly, as before), or unresolvable
// provider slots are not wrappers and keep their previous behavior.
func collectWrappers(pass *analysis.Pass) (
	map[*types.Func]*wrapperInfo, map[*ast.CallExpr]bool,
) {
	wrappers := map[*types.Func]*wrapperInfo{}
	innerSkips := map[*ast.CallExpr]bool{}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			obj, info, inner := wrapperOfDecl(pass, decl)
			if info == nil {
				continue
			}

			wrappers[obj] = info
			innerSkips[inner] = true
		}
	}

	return wrappers, innerSkips
}

// wrapperOfDecl decides whether one top-level declaration is a resolvable
// wrapper. A nil info means it is not (with the reason documented at each
// guard); the returned inner call is the body registration to skip.
func wrapperOfDecl(pass *analysis.Pass, decl ast.Decl) (*types.Func, *wrapperInfo, *ast.CallExpr) {
	fnDecl, isFn := decl.(*ast.FuncDecl)
	if !isFn || fnDecl.Body == nil {
		return nil, nil, nil
	}

	obj, isFunc := pass.TypesInfo.Defs[fnDecl.Name].(*types.Func)
	if !isFunc {
		return nil, nil, nil
	}

	sig, isSig := obj.Type().(*types.Signature)
	if !isSig {
		return nil, nil, nil
	}

	inner := soleRegistrationCall(pass, fnDecl.Body)
	if inner == nil {
		return nil, nil, nil // zero or multiple registrations: not a wrapper
	}

	innerName, kind, _ := doRegistrationCall(pass, inner)

	idx, hasProvider := providerArgIndex(innerName)
	if !hasProvider || idx >= len(inner.Args) {
		// As/AsNamed aliases carry no provider argument to map.
		return nil, nil, nil
	}

	paramIdent, isIdent := inner.Args[idx].(*ast.Ident)
	if !isIdent {
		// Concrete provider: the body call IS the registration.
		return nil, nil, nil
	}

	paramVar, isVar := pass.TypesInfo.Uses[paramIdent].(*types.Var)
	if !isVar {
		return nil, nil, nil
	}

	paramIdx := paramIndex(sig, paramVar)
	if paramIdx < 0 {
		// Parameter of a nested closure, not this wrapper.
		return nil, nil, nil
	}

	return obj, &wrapperInfo{innerFnName: innerName, kind: kind, paramIdx: paramIdx}, inner
}

// soleRegistrationCall returns the single samber/do registration call in
// body, or nil when there are none or several.
func soleRegistrationCall(pass *analysis.Pass, body *ast.BlockStmt) *ast.CallExpr {
	var calls []*ast.CallExpr

	ast.Inspect(body, func(n ast.Node) bool {
		if call, isCall := n.(*ast.CallExpr); isCall {
			if _, _, known := doRegistrationCall(pass, call); known {
				calls = append(calls, call)
			}
		}

		return true
	})

	if len(calls) != 1 {
		return nil
	}

	return calls[0]
}

// paramIndex locates variable v in sig's parameter tuple by object identity;
// -1 when v is not a parameter of this signature.
func paramIndex(sig *types.Signature, v *types.Var) int {
	for k := range sig.Params().Len() {
		if sig.Params().At(k) == v {
			return k
		}
	}

	return -1
}

// calleeIdent extracts the plain function identifier from a call expression,
// unwrapping explicit generic instantiation (f[T] and f[T1, T2]; inference
// sites are plain idents already). Selector callees (other packages, local
// methods) are not wrapper candidates in v1.
func calleeIdent(fun ast.Expr) (*ast.Ident, bool) {
	switch candidate := fun.(type) {
	case *ast.Ident:
		return candidate, true
	case *ast.IndexExpr:
		return calleeIdent(candidate.X)
	case *ast.IndexListExpr:
		return calleeIdent(candidate.X)
	}

	return nil, false
}
