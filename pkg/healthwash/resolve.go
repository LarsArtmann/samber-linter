package healthwash

import (
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
