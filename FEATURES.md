# FEATURES — samber-linter

Honest inventory by status. Last updated: 2026-09-09.

## DONE

- **HW-1 `unchecked-resource-holder`** (warn, confidence full): service
  registered via Provide*/Override* (non-transient) implements a `Shutdowner*`
  variant but no `Healthchecker*` variant.
- **HW-2 `contextless-check`** (info, confidence high): implements bare
  `HealthCheck()` without the context variant.
- **HW-3 `transient-health-washing`** (warn, confidence full): transient
  registration implementing a `Healthchecker*` variant.
- **HW-4 `lazy-never-built-pass`** (info, confidence medium): lazy
  registration implementing a `Healthchecker*` variant.
- **HW-5 `pointer-receiver-value-registration`** (warn, confidence full):
  check reachable only on `*T` while the registration stores value `T`.
- **HW-0** (meta): suppression directive without a reason, attributed to the
  suppressible site. Not disable-able (it audits the auditor).
- **Suppression model**: `//samber-linter:allow hw-N <reason>` (line above or
  trailing), `all` token, `until YYYY-MM-DD` expiry with automatic resurfacing.
- **HW-6 coverage ratchet**: alias-deduped checked/registered ratio, committed
  baseline file, `--coverage-min`, `--set-baseline` (atomic, idempotent via
  go-atomic-write).
- **Driver CLI**: text output, `--json` (go-finding), `--sarif` (SARIF 2.1),
  0/1/2 confidence exit codes (go-linter-sdk `ExitCodeByConfidence`),
  `--strict` (HW-unresolved), `--config` allowlist (reason mandatory).
- **Drift matrix**: mechanism assertions executed against REAL samber/do
  v2.0.0 and v2.1.0 sources from the module cache.
- **Discrimination proofs**: every P0 rule demonstrated to fail on a mutant
  analyzer (rule disabled); HW-0 intentionally exempt.
- **Target version awareness**: warns when the analyzed module's samber/do
  version is outside the verified set {v2.0.0, v2.1.0}.
- **golangci-lint v2 module plugin** (`plugin/`, `.custom-gcl.yml`), wiring
  copied from go-humanize-linter.
- **Runtime companion** `pkg/healthaudit`: registration/invocation hooks +
  sweep audit; `errored` = only runtime proof a check can fail.
- **Dogfooded**: run on `~/projects/CV` (9 findings: 7× HW-1, 2× HW-4;
  coverage 5/61 = 8% — matching the incident) and `~/projects/branching-flow`
  (clean).

## PARTIALLY DONE

- **Fixture corpus**: golden cases from the CV incident are frozen in
  `testdata/`; the `graphrag.Store` snapshot reflects incident HEAD. CV's
  ongoing fixes are NOT tracked by design (frozen-fixture rule).

## PLANNED

- **DO-9 backport** into `branching-flow/pkg/doanalyzerv2` (HW rules as
  DO-9 family).
- **Upstream filing** of `docs/upstream/ISSUE_DRAFT.md` (transient health
  checks + "checked vs skipped" sweep marker), after re-verification against
  the then-current samber/do release.
- **Suppression staleness report**: periodic `until` deadline report.

## WORTH CONSIDERING

- `--fix` for HW-5 (rewrite `Provide(i, T{})` → `Provide(i, &T{})`) —
  mechanical; everything else must stay report-only (a generated
  `HealthCheck` stub would itself be health-washing).
- SARIF `--include-suppressed` passthrough.
- Per-scope coverage breakdown (root vs child scopes) in HW-6.
- Public website (decision: deferred until adoption exists).
