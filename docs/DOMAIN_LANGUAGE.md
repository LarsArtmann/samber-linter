# Domain Language — samber-linter

The vocabulary this repo uses. Terms are defined once; code, README, and
reports use these meanings.

## Core concepts

- **Health-washing** — a service whose health-dashboard row renders green
  `pass` while no real check can fail it. The bug class this tool exists for.
- **Coverage** — `checked / registered`: the fraction of registered services
  in a module tree that implement at least one `Healthchecker*` variant the
  sweep can dispatch. HW-6's metric.
- **Unprotected service** — `registered − checked`; a sweep row that is an
  unconditional green. The ecology triage queue's ranking unit.

## Registration shapes (wrapper kinds)

Matching samber/do's wrapper per registration function (README §2.6):

- **Lazy** (`Provide`, `ProvideNamed`, `Override`, `OverrideNamed`) — built
  on first resolution; health check returns nil until built.
- **Eager** (`ProvideValue`, `ProvideNamedValue`, `OverrideValue`,
  `OverrideNamedValue`) — value stored immediately.
- **Transient** (`ProvideTransient`, …, `OverrideNamedTransient`) — new
  instance per resolution; the sweep never dispatches its check (upstream
  TODO, filed as samber/do#317).
- **Alias** (`As`, `AsNamed`) — delegates its check to the target wrapper;
  adds a sweep row but never a new method set. Tracked, never attributed,
  deduped in HW-6 coverage.

## Rules

- **HW-0** — suppression directive without a reason (`hw-suppression` meta
  rule); also fires for orphaned malformed directives. Not disable-able.
- **HW-1 `unchecked-resource-holder`** — registered type implements a
  `Shutdowner*` variant but no `Healthchecker*` variant; the headline rule.
- **HW-2 `contextless-check`** — implements bare `HealthCheck()` without the
  context variant; a hung check cannot be cancelled.
- **HW-3 `transient-health-washing`** — transient registration implementing a
  check that can never execute.
- **HW-4 `lazy-never-built-pass`** — lazy registration implementing a check;
  reports green until first resolution (informational; the noise dial).
- **HW-5 `pointer-receiver-value-registration`** — check reachable only on
  `*T` while the registration stores value `T`; the sweep's type assertion
  never sees it.
- **HW-6 `health-coverage-ratchet`** — not a finding; the CI gate over
  coverage against a committed baseline.
- **HW-unresolved** — strict-mode placeholder for statically unresolvable
  service types (e.g. closure returning an interface). Silent by default.

## Mechanism vocabulary

- **Registration site** — the `Provide*`/`Override*` call, where findings
  attribute (that is where the fix lands). Matched by **package path**
  (`github.com/samber/do/v2`), never identifier text.
- **Method set** — for `T` and `*T` computed separately; the sweep
  type-asserts the stored instance, so value registrations only see `T`'s
  set (the HW-5 trap).
- **Suppression directive** — `//samber-linter:allow hw-N <reason>`, honored
  above, trailing, or anywhere inside a multi-line registration call;
  optional `until YYYY-MM-DD` expiry with automatic resurfacing.
- **Allowlist** — `--config` file for recurring suppression categories;
  reason mandatory; empty `pathPattern` = project-wide.
- **Baseline** — the committed HW-6 floor (`.samber-linter-baseline.json`,
  written atomically by `--set-baseline`); coverage below it fails the run.
- **Drift matrix** — executable assertions of README §2 mechanism claims
  against real samber/do v2.0.0 + v2.1.0 sources; upstream behavior change
  fails CI, not the user's findings.
- **Discrimination proof** — every golden fixture must FAIL on a mutant
  analyzer (rule disabled); a rule whose tests test nothing cannot ship.

## Process vocabulary

- **Ecology scan** — running the analyzer over all local samber/do v2
  consumers (43 analyzed 2026-09-10), pseudonymized for commit-safe reports.
- **FP budget** — per-rule statement (docs/FP-BUDGETS.md) of tolerated
  residual false-positive risk; a rule exceeding its budget is a bug.
- **Golden fixture** — frozen CV-incident snapshot in `testdata/`; frozen on
  purpose, never a live reference.
