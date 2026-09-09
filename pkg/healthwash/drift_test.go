package healthwash

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The drift matrix re-verifies the README §2 mechanism assertions against the
// REAL samber/do sources in the module cache, for every pinned version. If a
// new samber/do release changes the mechanism, this test fails loudly instead
// of letting the rules silently rot (README §11 drift guard).

const doModulePath = "github.com/samber/do"

func moduleCacheDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir; cannot locate module cache")
	}
	return filepath.Join(home, "go", "pkg", "mod")
}

func doSourceDir(t *testing.T, version string) string {
	t.Helper()
	dir := filepath.Join(moduleCacheDir(t), doModulePath, "v2@"+version)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Skipf("samber/do %s not in module cache", version)
	}
	return dir
}

func parseFile(t *testing.T, dir, name string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("%s missing in %s", name, dir)
		}
		t.Fatalf("parse %s/%s: %v", dir, name, err)
	}
	return fset, f
}

// funcBodies collects the raw text of every method/function body with the
// given receiver-or-name.
func funcBodies(t *testing.T, f *ast.File, names ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, n := range names {
			if fd.Name.Name == n {
				var b strings.Builder
				// Render the body statements crudely but deterministically.
				dumpNode(&b, fd.Body)
				out[n] = b.String()
			}
		}
	}
	return out
}

func dumpNode(b *strings.Builder, n ast.Node) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *ast.BlockStmt:
		for _, s := range v.List {
			dumpStmt(b, s)
		}
	}
}

func dumpStmt(b *strings.Builder, s ast.Stmt) {
	switch v := s.(type) {
	case *ast.ReturnStmt:
		if len(v.Results) == 0 {
			b.WriteString("return;")
		} else if ident, ok := v.Results[0].(*ast.Ident); ok {
			b.WriteString("return " + ident.Name + ";")
		} else {
			b.WriteString("return <expr>;")
		}
	case *ast.IfStmt:
		b.WriteString("if(")
		dumpExpr(b, v.Cond)
		b.WriteString("){")
		dumpNode(b, v.Body)
		if v.Else != nil {
			b.WriteString("else{")
			dumpStmt(b, v.Else)
			b.WriteString("}")
		}
		b.WriteString("}")
	case *ast.SwitchStmt:
		b.WriteString("switch{")
		dumpNode(b, v.Body)
		b.WriteString("}")
	case *ast.TypeSwitchStmt:
		b.WriteString("typeswitch{")
		dumpNode(b, v.Body)
		b.WriteString("}")
	case *ast.CaseClause:
		b.WriteString("case[")
		for i, e := range v.List {
			if i > 0 {
				b.WriteString(",")
			}
			dumpExpr(b, e)
		}
		b.WriteString("]:")
		for _, inner := range v.Body {
			dumpStmt(b, inner)
		}
	case *ast.ExprStmt:
		dumpExpr(b, v.X)
		b.WriteString(";")
	}
}

func dumpExpr(b *strings.Builder, e ast.Expr) {
	switch v := e.(type) {
	case *ast.Ident:
		b.WriteString(v.Name)
	case *ast.UnaryExpr:
		b.WriteString("!")
		dumpExpr(b, v.X)
	case *ast.TypeAssertExpr:
		b.WriteString("assert(")
		dumpExpr(b, v.X)
		b.WriteString(")")
	case *ast.CallExpr:
		dumpExpr(b, v.Fun)
		b.WriteString("(")
		for i, a := range v.Args {
			if i > 0 {
				b.WriteString(",")
			}
			dumpExpr(b, a)
		}
		b.WriteString(")")
	case *ast.SelectorExpr:
		dumpExpr(b, v.X)
		b.WriteString("." + v.Sel.Name)
	}
}

// ifaceMethods maps interface name -> method signature strings.
func ifaceMethods(t *testing.T, f *ast.File) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			var methods []string
			for _, m := range iface.Methods.List {
				names := ""
				for i, n := range m.Names {
					if i > 0 {
						names += ","
					}
					names += n.Name
				}
				methods = append(methods, names)
			}
			out[ts.Name.Name] = methods
		}
	}
	return out
}

func TestDriftV210(t *testing.T) {
	dir := doSourceDir(t, "v2.1.0")
	assertMechanism(t, dir)
}

func TestDriftV200(t *testing.T) {
	dir := doSourceDir(t, "v2.0.0")
	assertMechanism(t, dir)
}

// assertMechanism encodes the README §2 pins as executable assertions. Each
// one documents which rule would silently break if the pin regressed.
func assertMechanism(t *testing.T, dir string) {
	t.Helper()

	// §2.5 lifecycle interfaces: names and method sets.
	fsetLc, fLc := parseFile(t, dir, "di_lifecycle.go")
	_ = fsetLc
	ifaces := ifaceMethods(t, fLc)
	for _, name := range []string{
		"Healthchecker", "HealthcheckerWithContext",
		"Shutdowner", "ShutdownerWithError",
		"ShutdownerWithContext", "ShutdownerWithContextAndError",
	} {
		if _, ok := ifaces[name]; !ok {
			t.Errorf("v-pinned interface %s missing from di_lifecycle.go", name)
		}
	}
	if ms := ifaces["Healthchecker"]; len(ms) != 1 || ms[0] != "HealthCheck" {
		t.Errorf("Healthchecker method set changed: %v — HW-2/HW-5 method matching must be revisited", ms)
	}
	if ms := ifaces["ShutdownerWithError"]; len(ms) != 1 || ms[0] != "Shutdown" {
		t.Errorf("ShutdownerWithError method set changed: %v (it is `Shutdown() error`, NOT `ShutdownWithError()`)", ms)
	}

	// §2.4 transient healthcheck: upstream TODO, unconditional nil; and
	// isHealthchecker() unconditional false.
	_, fTr := parseFile(t, dir, "service_transient.go")
	hc := funcBodies(t, fTr, "healthcheck", "isHealthchecker")
	if !strings.Contains(hc["healthcheck"], "return nil;") {
		t.Errorf("transient healthcheck no longer unconditionally returns nil — HW-3's rationale is void")
	}
	if !strings.Contains(hc["isHealthchecker"], "return false;") {
		t.Errorf("transient isHealthchecker no longer unconditional false — §7 runtime metrics spec is void")
	}
	hasTODO := false
	for _, cg := range fTr.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "@TODO: implement healthcheck") {
				hasTODO = true
			}
		}
	}
	if !hasTODO {
		t.Errorf("transient healthcheck TODO comment gone — upstream may have implemented checks; revisit HW-3")
	}

	// §2.2 eager wrapper: type-asserts the stored instance against both
	// Healthchecker variants (ctx first) with a fallthrough nil — the exact
	// "healthy or doesn't implement" collapse.
	_, fEager := parseFile(t, dir, "service_eager.go")
	eager := funcBodies(t, fEager, "healthcheck")
	if !strings.Contains(eager["healthcheck"], "case[HealthcheckerWithContext]:") ||
		!strings.Contains(eager["healthcheck"], "case[Healthchecker]:") {
		t.Errorf("eager healthcheck no longer dispatches on HealthcheckerWithContext then Healthchecker — HW-1/HW-5 method-set logic is void: %s", eager["healthcheck"])
	}
	if !strings.Contains(eager["healthcheck"], "return nil;") {
		t.Errorf("eager healthcheck fallthrough nil gone — non-implementers would surface differently: %s", eager["healthcheck"])
	}

	// §2.3 lazy wrapper: !s.built → nil before any dispatch.
	_, fLazy := parseFile(t, dir, "service_lazy.go")
	lazy := funcBodies(t, fLazy, "healthcheck")
	if !strings.Contains(lazy["healthcheck"], "if(!s.built){return nil;}") {
		t.Errorf("lazy unbuilt fast-path changed — HW-4's rationale is void: %s", lazy["healthcheck"])
	}

	// §2.6 registration functions create the named wrappers; Override*
	// mirrors Provide*.
	_, fDi := parseFile(t, dir, "di.go")
	di := map[string]bool{}
	for _, decl := range fDi.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			di[fd.Name.Name] = true
		}
	}
	for _, fn := range []string{
		"Provide", "ProvideNamed", "ProvideValue", "ProvideNamedValue",
		"ProvideTransient", "ProvideNamedTransient",
		"Override", "OverrideNamed", "OverrideValue", "OverrideNamedValue",
		"OverrideTransient", "OverrideNamedTransient",
		"As", "AsNamed",
	} {
		if !di[fn] {
			t.Errorf("registration function %s missing from di.go — the matcher surface must be revisited", fn)
		}
	}
}
