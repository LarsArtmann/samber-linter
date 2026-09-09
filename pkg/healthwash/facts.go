// Package healthwash implements the samber-linter static analyzer: it detects
// health-washing in projects using samber/do v2 — services that render as
// green "pass" on health dashboards but cannot actually fail.
//
// Rules (stable IDs, never rename once shipped):
//
//	HW-0  suppression without a reason
//	HW-1  unchecked-resource-holder (Shutdowner without Healthchecker)
//	HW-2  contextless-check (bare Healthchecker without the ctx variant)
//	HW-3  transient-health-washing (transient registration implementing a check)
//	HW-4  lazy-never-built-pass (lazy registration implementing a check)
//	HW-5  pointer-receiver-value-registration (check on *T, registered as T)
//	HW-unresolved  strict-mode placeholder for unresolvable service types
//
// Detection is type-based: a plain *analysis.Analyzer over registration call
// sites (the six Provide* and six Override* functions of github.com/samber/do/v2,
// matched by package path, never identifier text). As/AsNamed alias rows are
// tracked for HW-6 coverage math but never attributed.
package healthwash

// ServiceKind classifies how a service enters the container and which wrapper
// the sweep dispatches to.
type ServiceKind string

const (
	// KindLazy is created by Provide*/Override* (serviceLazy): nil until built.
	KindLazy ServiceKind = "lazy"
	// KindEager is created by ProvideValue*/OverrideValue* (serviceEager).
	KindEager ServiceKind = "eager"
	// KindTransient is created by ProvideTransient*/OverrideTransient*
	// (serviceTransient): healthcheck is an upstream TODO, always nil.
	KindTransient ServiceKind = "transient"
	// KindAlias is created by As/AsNamed (serviceAlias): delegates to target.
	KindAlias ServiceKind = "alias"
)

// ServiceRecord is one registration site as seen by the sweep. It is exported
// as a package fact so the driver can compute the HW-6 coverage ratchet as a
// project-level post-pass (never per-package Analyzer.Run).
type ServiceRecord struct {
	// Name is the inferred service name (the full type string, mirroring
	// samber/do's NameOf[T]; alias rows carry the alias type).
	Name string `json:"name"`
	// Type is the resolved service type as written at the registration site.
	Type string `json:"type"`
	// Kind is the wrapper class this registration creates.
	Kind ServiceKind `json:"kind"`
	// ImplementsCheck reports whether the *stored instance* satisfies a
	// Healthchecker variant — exactly what the sweep type-asserts.
	ImplementsCheck bool `json:"implementsCheck"`
	// Unresolved marks registrations whose service type could not be resolved
	// (interface-typed closure results). Silent by default; HW-unresolved in
	// --strict mode.
	Unresolved bool `json:"unresolved,omitempty"`
}

// PackageFacts is the analysis.Fact exported per package.
type PackageFacts struct {
	Records []ServiceRecord `json:"records"`
}

// AFact marks PackageFacts as an analysis.Fact.
func (PackageFacts) AFact() {}

func (f PackageFacts) String() string { return "healthwash: registration records" }

// AffectsHW6 reports whether the record counts toward the coverage ratchet.
// Alias rows delegate their healthcheck to the target and are deduped away;
// they must not inflate the denominator.
func (r ServiceRecord) AffectsHW6() bool { return r.Kind != KindAlias }
