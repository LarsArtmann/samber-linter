# Changelog

All notable changes to this project are documented here. Format based on
Keep a Changelog; versioning: SemVer.

## [0.1.1] - 2026-09-10

Hardening round after the first ecology scan (43 samber/do v2 consumers, 66
findings, 0 confirmed false positives). First release where the
golangci-lint CI job, `nix flake check` (build + tests + hermetic lint +
treefmt) and the upstream snippet gate all pass green.

### Fixed

- **Every CI run on 2026-09-10 was red:** go-finding v1.9.2 imports
  `encoding/json/v2`, and the workflow ran without `GOEXPERIMENT=jsonv2`, so
  the toolchain excluded those files and `test`/`dogfood` failed at build
  (run 34425222926). The workflow now sets `GOEXPERIMENT: jsonv2` — matching
  the flake and the verified-local suite.
- **Empty `--output` broke every plain invocation.** `samber-linter ./...`
  failed at flag validation with "invalid --output format" and exit 2. The
  zero value now means the documented plain-text default; regression test
  added.
- **Allowlist entries without `pathPattern` were silent no-ops.** An empty
  pattern now covers the whole project (the natural reading); entries without
  a rule warn on stderr and stay inert, matching the existing
  reason-mandatory warning.
- **Load failures exited 1**, the code reserved for findings. A run that
  cannot load any package has none and now exits 2.
- **Reasonless suppressions vanished once their finding was fixed.** HW-0 now
  also fires for orphaned malformed directives, at the comment itself —
  "unexplained suppressions rot" no longer depends on a live violation.

### Added

- `--check` advisory mode: report everything, always exit 0 (for CI
  annotation pipelines that parse output and set statuses themselves).
- `--disable HW-1,HW-4` on the CLI: the analyzer's rule mute, previously
  reachable only through the golangci plugin settings.
- Suppression directives are honored anywhere inside a multi-line
  registration call, not only on the line above (long provider closures
  commonly carry the directive in the argument list).
- Edge fixtures: duplicate registrations (both sites reported, one coverage
  row), registrations nested inside closures, orphaned malformed directives.
- `go.work` multi-module end-to-end test (workspace pattern is `all`).
- Plugin integration tests: in-process registration + settings pass-through
  proofs, and a `golangci-lint custom` build-and-fire proof gated on network.
- `docs/FP-BUDGETS.md`: per-rule false-positive budgets with the ecology
  evidence.
- Upstream snippet gate: `scripts/check-upstream-snippets.sh` compiles and
  runs every Go block in `docs/upstream/*.md` verbatim (CI job
  `upstream-snippets`; blocks quoting upstream source are marked
  ` ```go snippet-skip `).
- README links to the filed upstream issues samber/do#317/#318.

### Changed

- `flake.nix` rewritten on the `go-standard` module (go-nix-helpers) and the
  committed `vendor/` (659 files) removed — builds fetch via proxy.golang.org
  with a pinned `vendorHash`; the full test suite gates every `nix build`.

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
