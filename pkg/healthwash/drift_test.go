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

// parseSrc parses one file of dir and returns the file and raw source.
func parseSrc(t *testing.T, dir, name string) (*ast.File, string) {
	t.Helper()

	path := filepath.Join(dir, name)
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("%s missing in %s", name, dir)
		}

		t.Fatalf("parse %s: %v", path, err)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return file, string(src)
}

// funcSource returns the raw source text of the named function or method.
func funcSource(t *testing.T, f *ast.File, src, name string) string {
	t.Helper()

	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != name {
			continue
		}

		if funcDecl.Body == nil {
			return ""
		}

		return src[funcDecl.Body.Pos()-1 : funcDecl.Body.End()-1]
	}

	t.Fatalf("function %s not found", name)

	return ""
}

func fileHasComment(f *ast.File, needle string) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, needle) {
				return true
			}
		}
	}

	return false
}

func TestDriftV210(t *testing.T) {
	t.Parallel()

	assertMechanism(t, doSourceDir(t, "v2.1.0"))
}

func TestDriftV200(t *testing.T) {
	t.Parallel()

	assertMechanism(t, doSourceDir(t, "v2.0.0"))
}

// assertMechanism encodes the README §2 pins as executable assertions. Each
// one documents which rule would silently break if the pin regressed.
func assertMechanism(t *testing.T, dir string) {
	t.Helper()

	// §2.5 lifecycle interfaces: all six exist, with the exact method names.
	_, lcSrc := parseSrc(t, dir, "di_lifecycle.go")
	ifaces := collectIfaces(t, "di_lifecycle.go", lcSrc)

	for _, name := range []string{
		"Healthchecker", "HealthcheckerWithContext",
		"Shutdowner", "ShutdownerWithError",
		"ShutdownerWithContext", "ShutdownerWithContextAndError",
	} {
		if !ifaces[name] {
			t.Errorf("pinned interface %s missing from di_lifecycle.go", name)
		}
	}

	// The original §2.5 table error this repo exists to correct:
	// ShutdownerWithError is `Shutdown() error`, not `ShutdownWithError()`.
	withErr := sourceOfInterface(t, lcSrc, "ShutdownerWithError")
	if !strings.Contains(withErr, "Shutdown() error") ||
		strings.Contains(withErr, "ShutdownWithError") {
		t.Errorf("ShutdownerWithError signature drifted: %s", withErr)
	}

	// §2.4 transient healthcheck: upstream TODO, unconditional nil; and
	// isHealthchecker() unconditional false.
	fTr, trSrc := parseSrc(t, dir, "service_transient.go")

	hc := funcSource(t, fTr, trSrc, "healthcheck")
	if !strings.Contains(hc, "return nil") {
		t.Errorf("transient healthcheck no longer unconditionally returns nil — HW-3's rationale is void: %s", hc)
	}

	if !fileHasComment(fTr, "@TODO: implement healthcheck") {
		t.Errorf("transient healthcheck TODO comment gone — upstream may have implemented checks; revisit HW-3")
	}

	isHc := funcSource(t, fTr, trSrc, "isHealthchecker")
	if !strings.Contains(isHc, "return false") {
		t.Errorf("transient isHealthchecker no longer unconditional false — §7 runtime metrics spec is void: %s", isHc)
	}

	// §2.2 eager wrapper: dispatches on HealthcheckerWithContext then
	// Healthchecker over the stored instance, with a fallthrough nil.
	fEager, eagerSrc := parseSrc(t, dir, "service_eager.go")

	eager := funcSource(t, fEager, eagerSrc, "healthcheck")
	if !strings.Contains(eager, "HealthcheckerWithContext") ||
		!strings.Contains(eager, "Healthchecker)") {
		t.Errorf(
			"eager healthcheck no longer dispatches on HealthcheckerWithContext "+
				"then Healthchecker — HW-1/HW-5 method-set logic is void: %s",
			eager,
		)
	}

	if !strings.Contains(eager, "return nil") {
		t.Errorf("eager healthcheck fallthrough nil gone: %s", eager)
	}

	// §2.3 lazy wrapper: !s.built → nil before any dispatch.
	fLazy, lazySrc := parseSrc(t, dir, "service_lazy.go")

	lazy := funcSource(t, fLazy, lazySrc, "healthcheck")
	if !strings.Contains(lazy, "!s.built") || !strings.Contains(lazy, "return nil") {
		t.Errorf("lazy unbuilt fast-path changed — HW-4's rationale is void: %s", lazy)
	}

	// §2.6 registration surface: six Provide* + six Override* + As/AsNamed.
	_, diSrc := parseSrc(t, dir, "di.go")
	fns := collectFuncNames(t, "di.go", diSrc)

	for _, fn := range []string{
		"Provide", "ProvideNamed", "ProvideValue", "ProvideNamedValue",
		"ProvideTransient", "ProvideNamedTransient",
		"Override", "OverrideNamed", "OverrideValue", "OverrideNamedValue",
		"OverrideTransient", "OverrideNamedTransient",
		"As", "AsNamed",
	} {
		if !fns[fn] {
			t.Errorf("registration function %s missing from di.go — the matcher surface must be revisited", fn)
		}
	}

	// Alias rows delegate their healthcheck to the target wrapper.
	fAlias, aliasSrc := parseSrc(t, dir, "service_alias.go")

	aliasHc := funcSource(t, fAlias, aliasSrc, "healthcheck")
	if !strings.Contains(aliasHc, "targetName") && !strings.Contains(aliasHc, "serviceGetRec") {
		t.Errorf(
			"alias healthcheck no longer delegates to its target — HW-6 alias dedupe must be revisited: %s",
			aliasHc,
		)
	}
}

// collectIfaces returns the names of the interface types declared in src.
func collectIfaces(t *testing.T, name, src string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}

	ifaces := map[string]bool{}

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
					ifaces[ts.Name.Name] = true
				}
			}
		}
	}

	return ifaces
}

// collectFuncNames returns the names of the functions declared in src.
func collectFuncNames(t *testing.T, name, src string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}

	fns := map[string]bool{}

	for _, decl := range file.Decls {
		if fDecl, ok := decl.(*ast.FuncDecl); ok {
			fns[fDecl.Name.Name] = true
		}
	}

	return fns
}

func sourceOfInterface(t *testing.T, src, name string) string {
	t.Helper()

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, name+".go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == name {
				return src[ts.Pos()-1 : ts.End()-1]
			}
		}
	}

	t.Fatalf("interface %s not found", name)

	return ""
}
