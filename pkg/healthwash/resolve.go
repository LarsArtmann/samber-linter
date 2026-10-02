package healthwash

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ifaces holds the six samber/do lifecycle interfaces resolved from the
// analyzed package graph. Signatures verified against samber/do v2.1.0
// di_lifecycle.go:21,42,62,82,102,122 — all four Shutdowner* variants share
// the method name `Shutdown`, differing only in parameters/returns, so
// interface satisfaction (not method names) is the only sound test.
type ifaces struct {
	healthchecker    *types.Interface
	healthcheckerCtx *types.Interface
	shutdowner       *types.Interface
	shutdownerErr    *types.Interface
	shutdownerCtx    *types.Interface
	shutdownerCtxErr *types.Interface
}

func findDoPackage(pass *analysis.Pass) *types.Package {
	if pass.Pkg.Path() == DoPath {
		return pass.Pkg
	}

	for _, imp := range pass.Pkg.Imports() {
		if imp.Path() == DoPath {
			return imp
		}
	}

	return nil
}

func loadInterfaces(doPkg *types.Package) ifaces {
	i := ifaces{}
	i.healthchecker = lookupInterface(doPkg, "Healthchecker")
	i.healthcheckerCtx = lookupInterface(doPkg, "HealthcheckerWithContext")
	i.shutdowner = lookupInterface(doPkg, "Shutdowner")
	i.shutdownerErr = lookupInterface(doPkg, "ShutdownerWithError")
	i.shutdownerCtx = lookupInterface(doPkg, "ShutdownerWithContext")
	i.shutdownerCtxErr = lookupInterface(doPkg, "ShutdownerWithContextAndError")

	return i
}

func lookupInterface(pkg *types.Package, name string) *types.Interface {
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil
	}

	tn, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	return tn
}

// implements reports whether t (as stored in the container, i.e. the value
// the sweep type-asserts) satisfies iface. Unnamed types never carry methods
// of their own; only named types and pointers-to-named can satisfy
// samber/do's lifecycle interfaces.
func implements(t types.Type, iface *types.Interface) bool {
	if iface == nil || t == nil {
		return false
	}

	switch u := t.(type) {
	case *types.Named:
		return types.Implements(u, iface)
	case *types.Pointer:
		if _, ok := u.Elem().(*types.Named); ok {
			return types.Implements(u, iface)
		}
	}

	return false
}

// typeImplementsAnyCheck mirrors the sweep's wrapper assertion: it checks
// HealthcheckerWithContext first, then Healthchecker (service_eager.go:87-101,
// service_lazy.go:118-123) — either match means the sweep dispatches.
func (i ifaces) typeImplementsAnyCheck(t types.Type) bool {
	return implements(t, i.healthcheckerCtx) || implements(t, i.healthchecker)
}

func (i ifaces) typeImplementsBareCheck(t types.Type) bool {
	return implements(t, i.healthchecker)
}

func (i ifaces) typeImplementsCtxCheck(t types.Type) bool {
	return implements(t, i.healthcheckerCtx)
}

func (i ifaces) typeImplementsAnyShutdown(t types.Type) bool {
	return implements(t, i.shutdowner) || implements(t, i.shutdownerErr) ||
		implements(t, i.shutdownerCtx) || implements(t, i.shutdownerCtxErr)
}

// isConcrete reports whether t is a concrete type whose instance the sweep
// could type-assert: named/pointer/basic types yes; interfaces, type
// parameters, and untyped nil no (README §4 step 2: unresolvable classes).
func isConcrete(t types.Type) bool {
	if t == nil {
		return false
	}

	if b, isBasic := t.(*types.Basic); isBasic && b.Info()&types.IsUntyped != 0 {
		return false
	}

	if _, isTypeParam := t.(*types.TypeParam); isTypeParam {
		return false
	}

	_, isIface := t.Underlying().(*types.Interface)

	return !isIface
}

// providerBody returns the body of a registration's provider argument when it
// is statically visible in the analyzed package: a closure literal, or a
// function/method referenced by identifier or method value and declared in
// this package. Bodies declared in other packages are invisible (the same
// doctrine as HW-7's method bodies).
func providerBody(pass *analysis.Pass, arg ast.Expr) *ast.BlockStmt {
	var fnIdent *ast.Ident

	switch a := arg.(type) {
	case *ast.FuncLit:
		return a.Body
	case *ast.Ident:
		fnIdent = a
	case *ast.SelectorExpr:
		fnIdent = a.Sel
	}

	if fnIdent == nil {
		return nil
	}

	fn, isFn := pass.TypesInfo.Uses[fnIdent].(*types.Func)
	if !isFn {
		return nil
	}

	return funcDeclBody(pass, fn)
}

// funcDeclBody finds fn's declared body among the analyzed package's files.
// Object identity (one shared type-checker pass) makes the lookup exact.
func funcDeclBody(pass *analysis.Pass, fn *types.Func) *ast.BlockStmt {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fd, isFnDecl := decl.(*ast.FuncDecl)
			if !isFnDecl || fd.Body == nil {
				continue
			}

			if obj, isFunc := pass.TypesInfo.Defs[fd.Name].(*types.Func); isFunc && obj == fn {
				return fd.Body
			}
		}
	}

	return nil
}

// concreteInstanceFromReturns unifies the first-result types of a provider
// body's return statements into the one concrete instance type the runtime
// sweep will store. Returns nil when no return carries a concrete type, or
// when returns diverge (README §4: multiple implementations are
// unresolvable). Returns whose result is untyped nil (error paths), an
// interface-typed expression, or a type parameter carry no concrete evidence
// and are skipped.
func concreteInstanceFromReturns(pass *analysis.Pass, body *ast.BlockStmt) types.Type {
	var instance types.Type

	for _, ret := range returnStmts(body) {
		if len(ret.Results) == 0 {
			continue
		}

		t := pass.TypesInfo.TypeOf(ret.Results[0])
		if !isConcrete(t) {
			continue
		}

		if instance == nil {
			instance = t

			continue
		}

		if !types.Identical(instance, t) {
			return nil
		}
	}

	return instance
}

// returnStmts collects the return statements of a function body without
// descending into nested function literals — a closure's returns leave the
// closure, not the provider.
func returnStmts(body *ast.BlockStmt) []*ast.ReturnStmt {
	var out []*ast.ReturnStmt

	ast.Inspect(body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			out = append(out, v)
		}

		return true
	})

	return out
}
