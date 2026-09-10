package healthwash

import (
	"go/types"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// runOnFixture loads one fixture package from testdata/src in GOPATH mode and
// runs the analyzer over it, returning the raw diagnostics. Used by the
// discrimination proofs, which need to compare healthy vs mutant analyzer
// output without touching any shared fixture file.
func runOnFixture(t interface {
	Helper()
	Fatalf(format string, args ...any)
}, a *analysis.Analyzer, pkgPattern string,
) []analysis.Diagnostic {
	t.Helper()

	abs, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatalf("testdata abs: %v", err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedDeps | packages.NeedImports,
		Dir: filepath.Join(abs, "src", pkgPattern),
		Env: append(os.Environ(), "GOPATH="+abs, "GO111MODULE=off", "GOFLAGS=-mod=mod"),
	}

	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil {
		t.Fatalf("load %s: %v", pkgPattern, err)
	}

	if len(pkgs) == 0 {
		t.Fatalf("no packages loaded for %s", pkgPattern)
	}

	pkg := pkgs[0]
	for _, e := range pkg.Errors {
		t.Fatalf("fixture %s has errors (compile gate): %v", pkgPattern, e)
	}

	pass := &analysis.Pass{
		Analyzer:          a,
		Fset:              pkg.Fset,
		Files:             pkg.Syntax,
		OtherFiles:        pkg.OtherFiles,
		IgnoredFiles:      pkg.IgnoredFiles,
		Pkg:               pkg.Types,
		TypesInfo:         pkg.TypesInfo,
		TypesSizes:        types.SizesFor("gc", runtime.GOARCH),
		Report:            func(analysis.Diagnostic) {},
		ImportObjectFact:  func(obj types.Object, fact analysis.Fact) bool { return false },
		ExportObjectFact:  func(obj types.Object, fact analysis.Fact) {},
		ImportPackageFact: func(p *types.Package, fact analysis.Fact) bool { return false },
		ExportPackageFact: func(fact analysis.Fact) {},
		AllObjectFacts:    func() []analysis.ObjectFact { return nil },
		AllPackageFacts:   func() []analysis.PackageFact { return nil },
		ResultOf:          map[*analysis.Analyzer]any{},
	}

	var diags []analysis.Diagnostic

	pass.Report = func(d analysis.Diagnostic) { diags = append(diags, d) }

	if _, err := a.Run(pass); err != nil {
		t.Fatalf("analyzer run on %s: %v", pkgPattern, err)
	}

	return diags
}
