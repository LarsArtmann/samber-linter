# samber-linter

A static analyzer that kills **health-washing** in projects using
[`samber/do` v2](https://github.com/samber/do): services registered in the DI
container that render as green `pass` on health dashboards but **cannot
actually fail**, because they implement no health check at all, are never
instantiated, or are registered through a wrapper that never runs checks.

**One-line thesis:** a health dashboard's value equals the fraction of its
checks that can actually fail. samber/do's `HealthCheck*` sweep silently
inflates that denominator with always-green rows. This linter finds them at
CI time, before they become production theater.

---

## 1. The problem (real-world evidence)

Discovered 2026-09-09 on the CV production health dashboard
(`go-health` + `go-health-dashboard` over a samber/do root scope):

- **60 services** listed under "Healthy Services", every single one green
  `pass`, `Check Latency: 0ms`, `Details: —`.
- **~6 of the 60 rows can actually fail** (analytics DB connection,
  worker-pool lifecycle, dashboard watchdog, probe, Groq chat service,
  pipeline-store ping riding a side channel).
- The other **~54 rows verify only that a non-nil pointer exists**. Example:
  `graphrag.Store` holds an open SQLite file handle, implements
  `do.Shutdowner`, and reports unconditional `pass`. A dead store stays green
  until the process dies.
- Even the one real funnel check (SQLite pipeline store ping) is wired via a
  handler option (`WithPipelineStoreHealth`), not via the store's own
  `Healthchecker` implementation, so the dashboard row for the store service
  itself remains green-by-default.

This is not a bug in go-health or go-health-dashboard. It is the documented
behavior of samber/do's scope-wide health sweep interacting with ordinary
lazy, minimal service implementations. It is invisible in production
(everything green) and invisible in review (nothing looks wrong).

The companion skill rule this linter mechanizes
(`samber-do-best-practices` §6.3): _"Add lifecycle interfaces
(`do.Shutdowner*`, `do.Healthchecker*`) before first release for any
resource-holding service."_ Teams nod at that rule and skip it. A linter
does not nod.

## 2. Verified mechanism anatomy

Every claim below was read from `samber/do v2.1.0` source (module cache,
2026-09-09). File:line pins are for that version.

### 2.1 The sweep includes every service, checked or not

- `scope.go:307` `Scope.HealthCheckWithContext` walks the results of
  `asyncHealthCheckWithContext`.
- `scope.go:330-334`: `for name := range s.services` — **all** registered
  services are queued, plus every ancestor scope's services. Non-implementers
  are not filtered out; they come back as `nil` errors (below) and render as
  green rows.

### 2.2 "Healthy" and "doesn't implement" are the same output

- `scope.go:733-735` (`serviceHealthCheck`) doc, verbatim:
  _"Returns an error if the health check fails, or nil if the service is
  healthy **or doesn't implement the Healthchecker interface**."_
- `service_eager.go:87-101`: the wrapper type-asserts
  `HealthcheckerWithContext`, then `Healthchecker`; if neither matches, it
  falls through and returns `nil`. No marker distinguishes "checked and
  passed" from "nothing to check".

### 2.3 Lazy + never resolved = green without construction

- `service_lazy.go:128-134`: if `!s.built`, `healthcheck` returns `nil`
  immediately. A lazy singleton that no one has invoked yet reports
  **healthy without ever having been constructed**. On a freshly booted
  server, the first dashboard sweep paints every not-yet-used service green.

### 2.4 Transient services can never fail, period

- `service_transient.go:62-66`:

  ```go
  func (s *serviceTransient[T]) healthcheck(ctx context.Context) error {
      // @TODO: implement healthcheck ?
      return nil
  }
  ```

  An upstream TODO. A transient service that **does** implement
  `Healthchecker` still reports `pass` forever. Implementing the interface on
  a transient registration is dead code plus false confidence.

### 2.5 The lifecycle interfaces (di_lifecycle.go)

| Interface                       | Line | Signature                              |
| ------------------------------- | ---- | -------------------------------------- |
| `Healthchecker`                 | :21  | `HealthCheck() error`                  |
| `HealthcheckerWithContext`      | :42  | `HealthCheck(context.Context) error`   |
| `Shutdowner`                    | :62  | `Shutdown()`                           |
| `ShutdownerWithError`           | :82  | `Shutdown() error`                     |
| `ShutdownerWithContext`         | :102 | `Shutdown(context.Context)`            |
| `ShutdownerWithContextAndError` | :122 | `Shutdown(context.Context) error`      |

All four `Shutdowner*` variants share the method name `Shutdown`, differing
only in parameters/returns. Method-set matching must compare exact
signatures — a naive name-only match conflates `Shutdowner` with
`ShutdownerWithError` and misses real implementers.

### 2.6 Registration-to-wrapper mapping (di.go)

| Public API                     | Wrapper            | Health check behavior                             |
| ------------------------------ | ------------------ | ------------------------------------------------- |
| `Provide` (di.go:57)           | `serviceLazy`      | nil when unbuilt; nil when type doesn't implement |
| `ProvideNamed` (di.go:78)      | `serviceLazy`      | same                                              |
| `ProvideValue` (di.go:92)      | `serviceEager`     | nil when type doesn't implement                   |
| `ProvideNamedValue` (:107)     | `serviceEager`     | same                                              |
| `ProvideTransient` (:131)      | `serviceTransient` | **always nil** (upstream TODO)                    |
| `ProvideNamedTransient` (:155) | `serviceTransient` | **always nil**                                    |
| `Override` (di.go:187)         | `serviceLazy`      | same as `Provide`                                 |
| `OverrideNamed` (:199)         | `serviceLazy`      | same                                              |
| `OverrideValue` (:212)         | `serviceEager`     | same as `ProvideValue`                            |
| `OverrideNamedValue` (:224)    | `serviceEager`     | same                                              |
| `OverrideTransient` (:238)     | `serviceTransient` | **always nil**                                    |
| `OverrideNamedTransient` (:251)| `serviceTransient` | **always nil**                                    |

The six `Override*` functions (di.go:187-251) create the identical wrappers
(`newServiceLazy`:201, `newServiceEager`:226, `newServiceTransient`:253)
with identical health-check semantics. Override\* belongs in tests per DO-3 —
but this linter analyzes test files too, and production DO-3 violations are
exactly where blindness hurts.

`As`/`AsNamed` (di.go:456/507) register `serviceAlias` rows
(`service_alias.go`): the alias's `healthcheck` delegates to the target
wrapper (:114-126), so aliases introduce no new washing class — but they DO
add sweep rows. HW-6's coverage math dedupes aliases by target type;
HW-1..5 attribute only to `Provide*`/`Override*` sites (an alias adds no new
method set).

### 2.7 The pointer-receiver trap

The sweep type-asserts the **stored instance** (`any(s.instance)`). If a type
declares `func (s *T) HealthCheck(...)` (pointer receiver) but is registered
as the **value** type `T` (`ProvideValue(i, T{})` or a provider closure
returning `T`), the stored value does not satisfy the interface. The check
silently never runs. This compiles clean and renders green forever.

## 3. Rules

Analyzer package name: `healthwash`. Rule IDs `HW-*` — stable forever once
shipped; suppressions and configs key on them (`HW-0` is the suppression
meta-rule, `HW-unresolved` the strict-mode placeholder). (Backport IDs into
`branching-flow/pkg/doanalyzerv2` as `DO-9` family; that analyzer currently
ends at DO-8 — verified locally 2026-09-09.)

### HW-1 `unchecked-resource-holder` (severity: warn) — the headline rule

A service type registered via any `Provide*` call implements at least one
`Shutdowner*` variant but **neither** `Healthchecker` variant.

Rationale: if a service holds resources worth releasing at shutdown, it holds
state worth checking while running. Shutdown-without-healthcheck is exactly
the "we care on the way out but not during" asymmetry that produces
always-green dashboards.

```
internal/di/handlers.go:42:9: HW-1: *graphrag.Store implements do.Shutdowner
    but no Healthchecker; it renders unconditional "pass" on health dashboards.
    Implement HealthCheck(context.Context) error or suppress with a reason:
    //samber-linter:allow hw-1 <reason>
```

Precision note: handlers, config values, and stateless adapters do not
implement `Shutdowner` and are **not** flagged. The rule is deliberately
narrow: Shutdowner-without-Healthchecker is a high-precision proxy for
"resource-holding".

### HW-2 `contextless-check` (severity: info)

Implements `Healthchecker` but not `HealthcheckerWithContext`.

Rationale: the context variant lets the sweep's per-service timeout
(`RootScope` opts) propagate; the bare variant cannot be cancelled and a
hung check degrades the whole sweep. Prefer the context form.

### HW-3 `transient-health-washing` (severity: warn)

Registered via `ProvideTransient`/`ProvideNamedTransient` AND implements a
`Healthchecker` variant.

Rationale: §2.4. The check can never execute. Either the registration kind is
wrong (should be a singleton) or the implementation is dead code advertising
false confidence.

### HW-4 `lazy-never-built-pass` (severity: info)

Registered via `Provide`/`ProvideNamed` (lazy) AND implements a
`Healthchecker` variant.

Rationale: §2.3. Until first resolution, the service reports green without
existing. Two accepted fixes: register eagerly when the service is
boot-critical, or suppress. Informational because this is documented
lazy-singleton semantics, not a defect per se; the finding exists so the
team makes the choice consciously.

### HW-5 `pointer-receiver-value-registration` (severity: warn)

The registered type expression denotes value type `T`, and `T`'s method set
contains `HealthCheck`/`HealthCheckWithContext` only on receiver `*T`.

Rationale: §2.7. The implementation exists, the sweep never sees it. Fix:
register `*T` (or move the receiver to `T`). This is the nastiest variant
because reading the source shows a plausible implementation.

### HW-6 `health-coverage-ratchet` (severity: none; CI gate mode)

Not a per-line finding. Reports the module-tree ratio:

```
health-coverage: 6/60 = 10% of registered services implement a check (threshold: 60%)
```

Modeled after CV's `any-count` ratchet: the count lives in a committed
baseline file; coverage below baseline fails CI; improving coverage requires
`--set-baseline` to lock the gain immediately. Directional, never gratuitous.

## 4. Detection algorithm

Built on `golang.org/x/tools/go/analysis` (type-checking based; AST alone
cannot decide interface satisfaction).

1. **Find registration sites.** Selector calls matching the six `Provide*`
   functions AND the six `Override*` functions (resilient to dot-imports and
   renames via package-path match, not identifier text). `As`/`AsNamed`
   alias rows are tracked but never attributed (§2.6).
2. **Resolve the service type.**
   - `Provide*`: the provider closure's **return type** (first return; drop
     the error return). Chase named types and type aliases to the underlying
     struct/named type.
   - `ProvideValue*`: the type of the value argument expression.
   - Interface-typed registrations (`Provide[SomeInterface]` with a closure
     returning a concrete type): analyze the **concrete returned type**, since
     the sweep asserts the stored instance (§2.2), not the registration
     interface.
   - Unresolvable (returns an interface with multiple implementations,
     generic parameter): emit `HW-unresolved` only in `--strict` mode;
     default silent.
3. **Compute method sets** for `T` and `*T` separately from the type info,
   then evaluate the rule table above.
4. **Attribute the finding** to the registration call site (that is where
   the fix lands), with the service type name in the message.

Cross-module scope: one analyzer pass per module tree is sufficient because
registrations live in composition roots; running under `go.work` covers the
whole workspace. Services registered by **third-party libraries** (e.g.
go-health-dashboard registering itself) are out of static reach and are the
runtime companion's job (§7).

## 5. Suppression model

- Inline, reason **required**:

  ```go
  //samber-linter:allow hw-1 config struct is inert data; nothing to check
  do.ProvideValue(injector, cfg)
  ```

- A suppression without a reason text is itself a finding (`HW-0`), mirroring
  the `//nolint // rationale` discipline: unexplained suppressions rot into
  permanent darkness.
- Suppressions map onto `go-finding`'s `Suppression{Kind, Rule, Reason,
  ExpiresAt}` data model (verified 2026-09-09): inline directives are
  `SuppressionInSource`, the config allowlist is `SuppressionInConfig`. The
  optional `until <date>` expiry triggers `IsExpired()` re-review; adopted
  in P2 alongside the config layer.
- Config-file allowlist for recurring categories (e.g. `*.handlers.*`), kept
  separate from inline suppressions so inline remains the default.

## 6. CLI surface

```
samber-linter ./...                     # analyze, text output
samber-linter --json ./...              # go-finding JSON (rule, pos, type, service, confidence)
samber-linter --sarif ./...             # SARIF 2.1 for code scanning
samber-linter --coverage-min 0.6 ./...  # HW-6 as a gate
samber-linter --set-baseline ./...      # lock current coverage as the new floor (atomic write)
```

Exit codes follow the confidence ternary (`go-linter-sdk`'s
`ExitCodeByConfidence`): `0` clean, `1` high-confidence findings, `2` needs
triage. Severity ≠ confidence: HW-1/3/5 are type facts (confidence Full),
HW-2 High, HW-4 Medium — a judgment call by design.

Stack: detection is a plain `*analysis.Analyzer` (golangci plugin contract);
findings, confidence, suppression, JSON/SARIF, and the HW-6 ratchet (a
project-level Report post-pass, never per-package `Analyzer.Run`) live in
the driver via `go-finding`. Baselines persist through `go-atomic-write`.
Ship as a `go vet`-style driver and as a `golangci-lint` custom plugin
(wiring copied from `go-humanize-linter/plugin`).

## 7. Runtime companion (optional, phase 3)

Static analysis cannot see third-party registrations, and cannot catch the
"implements but always returns nil" class of fake checks. A tiny
`healthaudit` package (modeled on `samber-do-auditlog`'s wrapping pattern)
can count, per sweep, services that were **actually dispatched** to a
`Healthchecker` implementation vs. skipped as non-implementers, exposing:

```
healthwash_registered{scope="root"} 60
healthwash_checked{scope="root"} 6
```

If `checked / registered` dips below the static baseline, the runtime caught
what the compiler could not. Transients count as *skipped*, never *checked*
(`serviceTransient.isHealthchecker()` returns `false` unconditionally,
`service_transient.go:58-60`). Optional; the static analyzer is the MVP and
delivers most of the value.

## 8. Relationship to existing tooling

| Tool                                | Owns                                                                                                                                                           | Relationship                                                                                                                               |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `branching-flow/pkg/doanalyzerv2`   | DO-1..DO-8: Must-in-runtime, missing shutdown, override-in-prod, global injector, invoke-in-loops, shutdown-accesses-dep, service locator, injector-in-service | Complementary. All eight are **usage-shape** rules; `healthwash` is a **capability-completeness** rule. Backport as DO-9 once stable.      |
| `samber-do-auditlog`                | Runtime audit of registrations, invocations, health checks, shutdowns                                                                                          | Feeds the runtime companion (§7); does not judge implementations, only records them.                                                       |
| `go-health` / `go-health-dashboard` | Serving and rendering health state                                                                                                                             | The victim, not the culprit: they faithfully render what the sweep returns. Fix the registrations, and their dashboards become meaningful. |
| golangci-lint                       | Runner                                                                                                                                                         | Distribution channel via custom plugin (§6).                                                                                               |

## 9. Implementation phases

- **P0 (MVP):** analyzer skeleton + HW-1 + HW-5. These two are pure
  type-facts over registration sites and catch the production incident class
  described in §1.
- **P1:** HW-2, HW-3, HW-4 (trivial once type resolution exists).
- **P2:** HW-6 coverage ratchet + `--json` + baseline file.
- **P3:** runtime companion (`healthaudit` metrics); doanalyzerv2 DO-9
  backport; upstream conversation (the transient TODO in §2.4 and a
  "checked vs skipped" marker on sweep results are worth proposing upstream
  the verified way, with a failing test).

### Test corpus (golden cases from the CV incident, 2026-09-09)

| Fixture                                                          | Expected                        |
| ---------------------------------------------------------------- | ------------------------------- |
| `graphrag.Store` (Shutdowner, no Healthchecker, Provide)         | HW-1 fires                      |
| Value registration of struct with pointer-receiver `HealthCheck` | HW-5 fires                      |
| Transient registration implementing `HealthcheckerWithContext`   | HW-3 fires                      |
| Lazy registration implementing `HealthcheckerWithContext`        | HW-4 fires (info)               |
| Contextless `HealthCheck() error` implementer                    | HW-2 fires (info)               |
| `internal/database/connection.go` (real checker)                 | clean                           |
| `chat/groq` ChatService (real checker)                           | clean                           |
| Handler struct, no Shutdowner, no Healthchecker                  | clean (rule precision)          |
| `//samber-linter:allow hw-1 <reason>` on a flagged site          | suppressed                      |
| `//samber-linter:allow hw-1` without reason                      | `HW-0` fires                    |

Discrimination proof required before shipping P0: each golden case must be
shown to **fail** on a mutant analyzer (rule inverted or removed) in a
scratch copy, never by mutating the shared tree.

Fixture-freezing rule: all golden fixtures are **frozen snapshots copied
into `testdata/`** at authoring time, never live references into the CV (or
any) repository. CV's `graphrag.Store` gained a `HealthCheck` in its working
tree within hours of the incident analysis — live references rot.

## 10. Non-goals

- Not a general DI style linter. DO-1..DO-8 already exist; do not duplicate.
- Not runtime monitoring. Dashboards and auditlog own that.
- Not a judge of check **quality** (a check that always returns nil is
  statically indistinguishable from an honest one; §7 is the answer there).
- No support for samber/do v1 (different registration API; assess demand
  first).

## 11. Verification ledger

Claims in this document and their sources:

| Claim                                                                        | Source                                                                                                                                                          |
| ---------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Sweep queues every service in scope + ancestors                              | samber/do v2.1.0 `scope.go:307,328-348`                                                                                                                         |
| Non-implementers return nil ("healthy or doesn't implement")                 | `scope.go:733-735`, `service_eager.go:87-101`                                                                                                                   |
| Lazy unbuilt services return nil unchecked                                   | `service_lazy.go:128-134`                                                                                                                                       |
| Transient healthcheck is an upstream TODO, always nil                        | `service_transient.go:62-66`                                                                                                                                    |
| Interface names/signatures                                                   | `di_lifecycle.go:21,42,62,82,102,122`                                                                                                                           |
| Registration-to-wrapper mapping                                              | `di.go:57,78,92,107,131,155`                                                                                                                                    |
| Wrapper asserts the stored instance (pointer-receiver trap)                  | `service_eager.go:88,94`, `service_lazy.go:136`                                                                                                                 |
| CV dashboard: 60 rows, all pass, ~6 fail-capable                             | `https://cv.home.lan/admin/health` (fetched 2026-09-09) + implementer grep (`chat/groq/chat.go`, `internal/database/connection.go`, `internal/di/lifecycle.go`) |
| CV pipeline-store ping rides a handler option, not the store's Healthchecker | `internal/di/handlers_pipeline.go:38-40`                                                                                                                        |
| doanalyzerv2 currently defines DO-1..DO-8                                    | `~/projects/branching-flow/pkg/doanalyzerv2/doc.go` (read 2026-09-09)                                                                                           |
| samber-do-best-practices §6.3 lifecycle-interface rule                       | `~/.config/crush/skills/samber-do-best-practices/SKILL.md`                                                                                                      |
| `ShutdownerWithError` is `Shutdown() error`, not `ShutdownWithError()`       | `di_lifecycle.go:82-83` (re-read 2026-09-09; the original table was wrong)                                                                                      |
| Override\* family registers identical wrappers                               | `di.go:187-251` (`newServiceLazy`:201, `newServiceEager`:226, `newServiceTransient`:253)                                                                        |
| As/AsNamed register delegating alias rows                                    | `di.go:456,507`, `service_alias.go:100-126`                                                                                                                     |
| transient `isHealthchecker()` returns `false` unconditionally                | `service_transient.go:58-60`                                                                                                                                    |
| CV registration sites: 61 today (53 `Provide`, 8 `ProvideValue`) — counts drift | `~/projects/CV` grep 2026-09-09                                                                                                                              |
| go-finding / go-linter-sdk / go-atomic-write APIs                            | local source reads 2026-09-09 (`analysis/analysis.go`, `registry.go:358`, `atomicwrite.go:66-140`)                                                              |

Upstream drift guard: pin the analyzed samber/do version in CI and re-run the
mechanism assertions (§2) against new releases; a behavior change upstream
should fail the build loudly rather than silently invalidate the rules. The
matrix covers **v2.0.0 and v2.1.0** (both in the local module cache); the
driver reads the target module's samber/do version and emits an info finding
when it falls outside the verified set.
