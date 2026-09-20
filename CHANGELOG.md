# Changelog

All notable changes to this project are documented here. Format based on
Keep a Changelog; versioning: SemVer.

## [Unreleased]

### Added

- **Max-recall profile (opt-in) + `--strict` unresolved summary.** The
  documented `--min-confidence 0.5 --strict` profile surfaces HW-4 and
  unresolved registrations without flipping any default; the threshold
  decision is recorded as profile-first pending one release of measured FP
  data. With `--strict`, human output carries a summary line counting
  statically unresolvable registrations (the findings themselves stay
  Medium/triage-only).
- **HW-7 `unconditional-nil-check`**: a new detection rule for the founding
  incident in syntactic form — a health check whose body is exactly
  `return nil` can never fail and always renders green. Fires on lazy/eager
  registrations whose stored type satisfies a `Healthchecker` variant;
  warn severity, Full confidence (gates by default, like HW-1/3/5).
  Deliberately narrow in v1: delegation, multi-statement bodies, and naked
  returns can fail and stay negative; transients stay HW-3's, unreachable
  `*T` bodies stay HW-5's. Bodies are read where they are declared and
  shipped across package boundaries as a `NilBodyFact` object fact, so the
  standard declare-here/register-there architecture is covered. Budget and
  boundary cases in `docs/FP-BUDGETS.md`.

### Fixed

- **Machine output no longer carries human gate lines.** The coverage line
  (`health-coverage: …`), baseline acknowledgement, and the `--strict`
  summary were appended to stdout in every mode, corrupting `--json`/
  `--sarif`/structured `--output` whenever a gate ran (same defect class as
  the `--check` advisory line). Gate verdicts now travel via the exit code
  and stderr; human lines print only on human-facing presentations.
- **`--coverage-min` no longer bypasses the baseline gate.** The absolute
  coverage floor used to short-circuit the baseline ratchet entirely: a
  baseline file that existed was never read, so a stale or corrupt one (e.g.
  a v1-schema file written by an old binary) passed a green gate unnoticed.
  The gates now compose — with `--coverage-min` set, the baseline file is
  still validated (schema, counters, per-rule floors) and enforced as a
  ratchet. Found via the 2026-09-20 CV healthwash-gate incident, where this
  driver defect was the root cause.
- **The reported tool version can no longer go stale.** `-version` and the
  version field in findings/SARIF were a hand-pinned constant that had
  silently drifted two releases behind the tags (`0.1.1` while v0.2.1 was
  latest), making analyzer version-gating impossible for consumers. The
  version now resolves from the build: the module proxy version for
  `go install …@vX` builds, the short VCS revision for source builds
  (`devel+<rev>[.dirty]`), `devel` when the build has no identity. Release
  builds can still pin it with `-ldflags "-X main.version=vX.Y.Z"`.
- **The CI `lint` job never had a working golangci-lint.**
  `golangci-lint-action` `version: latest` resolves to a **v1** binary
  (v1.64.8) that cannot load the v2 config (exit 3, run 35089293309) — the
  job was red since day one, behind two layers (this on top of the ~141
  findings burned down in 3572927). The version is now pinned to **v2.13.2**
  everywhere: CI action, `.custom-gcl.yml`, and nixpkgs (whose 2.13.2 binary
  is built with go1.27 ≥ the 1.26.7 go.mod floor; the hermetic
  `nix run .#lint` gate already ran that version).
- **`--check` corrupted machine output.** The advisory line ("--check:
  advisory run; exit code forced to 0") was appended after `--json`/`--sarif`
  and the structured `--output` formats (csv/tsv/html/xml/asciidoc),
  breaking parsers. It is now printed only on human-facing presentations
  (plain text, table, markdown).
- **A failed quality gate no longer escalates only from exit 0.** Baseline
  regressions and coverage-gate failures now force exit 1 even when findings
  alone would map to 2 (triage-only): a regressed ratchet is never advisory.
- **The CLI ignored inherited `GOFLAGS` poisoning.** A global
  `GOFLAGS=-mod=vendor` (or `-mod=mod`) broke package loading for every CLI
  invocation — only the new programmatic SDK path sanitized it. Both paths
  now share one loader (`load`): the inherited GOFLAGS is stripped of `-mod`
  tokens (explicit caller env still wins, last occurrence), and the SDK's
  duplicate `packages.Config` — previously kept in sync only by a comment —
  is gone.
- **The ecology survey over-scanned go.work projects.** The workspace
  pattern `all` expands to the full dependency closure, so dependency
  packages with their own samber/do registrations were counted against the
  consumer (a phantom HW-2 surfaced from a dependency's own self-
  registration, unfixable and unownable by the analyzed repo). The survey
  now scans each workspace module listed in `go.work` with `./...` and sums
  coverage across modules; ownership lands where the code lives.

### Added

- **Baseline v2 (HW-6):** the committed ratchet floor now also records
  per-rule finding counts; any rule exceeding its committed count fails the
  gate even when aggregate coverage stays flat (the regression mode v1 could
  not see). Baseline files are validated loudly — older schema (v1 names the
  `--set-baseline` migration), newer schema, counters inconsistent with the
  stored coverage, negative counts, or unparseable JSON all fail the run
  instead of silently degrading to "no baseline".
- `scripts/ecology-scan.sh`: repeatable 43-project ecology survey with
  pseudonymous output (stable `p-` pseudonyms matched against the
  out-of-repo keyfile), ranked by unprotected services, load errors surfaced
  with stderr excerpts. Replaces the hand-run one-off scans; used as the
  post-change regression proof.
- CI `dogfood` job also runs `--output markdown`, so the presentation path
  can no longer rot silently (plain-text dogfood remains).
- The upstream snippet gate now also covers `docs/status/**`: the same
  contract applies to point-in-time reports (a ```go block is either a
  runnable repro or marked `snippet-skip`; unclassified blocks fail), so a
  pasted-in fragment can never silently rot.
- `nix flake check` now gates hand-edited markdown/json/yaml via a hermetic
  dprint check (`checks.format-dprint`): the repo's URL-pinned dprint
  plugins are prefetched by hash and injected as store paths, with a drift
  guard that fails when dprint.json references a version the flake does not
  pin.

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
