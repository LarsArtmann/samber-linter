package healthwash

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// doRegistrationCall classifies one call node as a direct samber/do
// registration: a selector whose callee is a known registration function of
// the do package (package-path match, resilient to dot-imports and renames).
func doRegistrationCall(pass *analysis.Pass, call *ast.CallExpr) (fnName string, kind ServiceKind, ok bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return "", "", false
	}

	fn, isFn := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !isFn || fn.Pkg() == nil || fn.Pkg().Path() != DoPath {
		return "", "", false
	}

	k, known := regKindOf(fn.Name())
	if !known {
		return "", "", false
	}

	return fn.Name(), k, true
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
			fnDecl, isFn := decl.(*ast.FuncDecl)
			if !isFn || fnDecl.Body == nil {
				continue
			}

			obj, isFunc := pass.TypesInfo.Defs[fnDecl.Name].(*types.Func)
			if !isFunc {
				continue
			}

			sig, isSig := obj.Type().(*types.Signature)
			if !isSig {
				continue
			}

			var regCalls []*ast.CallExpr

			ast.Inspect(fnDecl.Body, func(n ast.Node) bool {
				if call, isCall := n.(*ast.CallExpr); isCall {
					if _, _, known := doRegistrationCall(pass, call); known {
						regCalls = append(regCalls, call)
					}
				}

				return true
			})

			if len(regCalls) != 1 {
				continue
			}

			inner := regCalls[0]
			innerName, kind, _ := doRegistrationCall(pass, inner)

			idx, hasProvider := providerArgIndex(innerName)
			if !hasProvider || idx >= len(inner.Args) {
				continue // As/AsNamed aliases carry no provider argument to map
			}

			paramIdent, isIdent := inner.Args[idx].(*ast.Ident)
			if !isIdent {
				continue // concrete provider: the body call IS the registration
			}

			paramVar, isVar := pass.TypesInfo.Uses[paramIdent].(*types.Var)
			if !isVar {
				continue
			}

			paramIdx := -1
			for k := range sig.Params().Len() {
				if sig.Params().At(k) == paramVar {
					paramIdx = k

					break
				}
			}

			if paramIdx < 0 {
				continue // parameter of a nested closure, not this wrapper
			}

			wrappers[obj] = &wrapperInfo{innerFnName: innerName, kind: kind, paramIdx: paramIdx}
			innerSkips[inner] = true
		}
	}

	return wrappers, innerSkips
}

// calleeIdent extracts the plain function identifier from a call expression,
// unwrapping explicit generic instantiation (f[T] and f[T1, T2]; inference
// sites are plain idents already). Selector callees (other packages, local
// methods) are not wrapper candidates in v1.
func calleeIdent(fun ast.Expr) (*ast.Ident, bool) {
	switch f := fun.(type) {
	case *ast.Ident:
		return f, true
	case *ast.IndexExpr:
		return calleeIdent(f.X)
	case *ast.IndexListExpr:
		return calleeIdent(f.X)
	}

	return nil, false
}
