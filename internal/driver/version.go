package driver

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/larsartmann/samber-linter/pkg/healthwash"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// warnDover reads the analyzed module's go.mod and warns when its samber/do
// version falls outside the verified set. All behavioral claims in the rules
// are pinned to specific samber/do releases; an unverified version means the
// mechanism may have drifted (README §11).
func warnDover(out io.Writer, pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		if pkg.Module == nil || pkg.Module.GoMod == "" {
			continue
		}
		data, err := os.ReadFile(pkg.Module.GoMod)
		if err != nil {
			continue
		}
		mf, err := modfile.Parse(pkg.Module.GoMod, data, nil)
		if err != nil {
			continue
		}
		for _, req := range mf.Require {
			if req.Mod.Path == healthwash.DoPath {
				if VerifiedDover[req.Mod.Version] {
					return
				}
				verified := make([]string, 0, len(VerifiedDover))
				for v := range VerifiedDover {
					verified = append(verified, v)
				}
				sort.Strings(verified)
				fmt.Fprintf(out, "[info] target module uses github.com/samber/do/v2 %s; mechanism assertions are verified against %v — rules may not hold\n",
					req.Mod.Version, verified)
				return
			}
		}
		return
	}
}
