# Upstream issue draft: transient services never dispatch health checks

> Status: FILED 2026-09-10 as [samber/do#317](https://github.com/samber/do/issues/317)
> (concise, human-edited version of this draft; repro re-run and failing as
> expected immediately before filing). The broader status-model proposal
> (explicit passed/not-built/unsupported/skipped states) was filed separately
> as [samber/do#318](https://github.com/samber/do/issues/318), referencing #317.
> Source verification: service_transient.go:58-66, scope.go:733-735,
> service_lazy.go:128-134 (v2.1.0); drift matrix covers v2.0.0 + v2.1.0.

## Summary

`ProvideTransient` services that implement `Healthchecker` /
`HealthcheckerWithContext` are never dispatched by
`Scope.HealthCheckWithContext`: the wrapper's `healthcheck` is an upstream
TODO that unconditionally returns nil, and `isHealthchecker()` reports false
regardless of the concrete type.

## Evidence (samber/do v2.1.0)

- `service_transient.go:62-66`:

  ```go
  func (s *serviceTransient[T]) healthcheck(ctx context.Context) error {
      // @TODO: implement healthcheck ?
      // It requires to store each instance of service, which is not good because of memory leaks.
      return nil
  }
  ```

- `service_transient.go:58-60`: `isHealthchecker()` returns `false`
  unconditionally, so even introspecting consumers cannot distinguish
  "checked and passed" from "not checkable".

- Consequence (same collapse as non-implementers): a transient service whose
  type implements a health check renders an unconditional green `pass` on
  health dashboards — false confidence, statically indistinguishable from an
  honest check.

## Reproduction

```go
type perRequestCheck struct{}

func (perRequestCheck) HealthCheck(context.Context) error {
    return errors.New("transient service down")
}

injector := do.New(func(i do.Injector) {
    do.ProvideTransient(i, func(i do.Injector) (perRequestCheck, error) {
        return perRequestCheck{}, nil
    })
})
results := injector.HealthCheckWithContext(ctx)
// results["pkg.perRequestCheck"] == nil, expected errTransientDown
```

## Proposed directions (for discussion)

1. **Document + guard (cheapest):** document on `ProvideTransient` that
   health checks are not dispatched for transients, and have the sweep skip
   them with a distinguishable marker.
2. **Distinguishable result:** add a sentinel or result metadata so sweep
   consumers can tell "checked: pass" from "not checkable" — this also
   benefits eager/lazy non-implementers (a "checked vs skipped" marker on
   sweep results, independent of the transient TODO).
3. **Implement transient checks (the TODO):** the memory-leak concern in the
   TODO comment is real; a bounded approach could dispatch checks on
   recently-created instances with a per-scope LRU, or explicitly reject
   `ProvideTransient` registrations whose type implements a `Healthchecker`
   variant at registration time (fail fast, zero memory cost).

Option 3's reject-at-registration variant turns a silent false-green into a
startup error, which matches the library's fail-fast posture elsewhere
(e.g. duplicate registration).

## Relationship to this repository

`samber-linter` (HW-3) flags transient registrations that implement health
checks at CI time; the upstream fix would make HW-3's "register as a
singleton or drop the dead implementation" guidance enforceable at runtime.
