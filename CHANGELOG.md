# Changelog

All notable changes to this project are documented here. Format based on
Keep a Changelog; versioning: SemVer.

## [0.1.0] - 2026-09-09

Initial release: the healthwash analyzer with all five detection rules, the
HW-6 coverage ratchet, the go-finding based driver, and the healthaudit
runtime companion.

### Added

- Analyzer `pkg/healthwash`: HW-1..HW-5 type-based rules over samber/do v2
  registration sites (six `Provide*` + six `Override*` functions matched by
  package path), `HW-0` reason-missing meta-rule, `HW-unresolved` strict-mode
  placeholder. Facts exported per package for the ratchet.
- Driver `cmd/samber-linter`: text/JSON/SARIF output, 0/1/2 confidence exit
  codes, HW-6 coverage ratchet with committed baseline + `--set-baseline`
  (idempotent atomic writes), `--coverage-min` gate, `--config` allowlist,
  `--strict`, target-version awareness for samber/do {v2.0.0, v2.1.0}.
- Suppression model: `//samber-linter:allow hw-N <reason>` with optional
  `until YYYY-MM-DD` expiry; reason mandatory (HW-0).
- Drift matrix: executable README §2 mechanism assertions against real
  samber/do v2.0.0 and v2.1.0 module-cache sources.
- Discrimination proofs: P0 rules demonstrated to fail on mutant analyzers.
- `pkg/healthaudit`: runtime companion exposing registered/invoked/errored
  sweep counts (transients always counted as skipped).
- golangci-lint v2 module plugin (`plugin/`).
- Frozen CV-incident fixture corpus + compile-gated golden tests.
- Dogfood evidence: 9 findings on `~/projects/CV` (7× HW-1, 2× HW-4),
  coverage 5/61 = 8%; `~/projects/branching-flow` clean.

### Corrected (vs. the original design README)

- §2.5 table: `ShutdownerWithError` is `Shutdown() error` (was
  `ShutdownWithError() error` — a name-only match would have missed every
  implementer).
- §4 detection surface: added the six `Override*` functions and
  `As`/`AsNamed` alias-row policy (aliases tracked, never attributed,
  deduped in HW-6).
- Fixture rule: golden fixtures are frozen snapshots, never live repo
  references (CV fixed `graphrag.Store` within hours of the incident).
