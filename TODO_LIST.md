# TODO_LIST — samber-linter

Living task source. Snapshot plan with full rationale, impact/effort scoring, and
execution graph: `docs/planning/2026-09-09_20-09_SUPERB-PARETO-EXECUTION-PLAN.html`.
Mark items `[x]` when done and move them to the Done section at the bottom.

## Verification round — 2026-09-10 (post-ecology hardening)

The v0.1.0 completion claim above over-marked several items `[x]`. This round
actually executed them, plus fixes found by running the shipped binary:

- [x] Empty `--output` regression: plain `samber-linter ./...` exited 2 ("invalid
  --output format"); zero flag value now means the plain-text default (test added)
- [x] A68 (real) `--check` advisory flag: report everything, always exit 0
- [x] `--disable` exposed on the CLI (analyzer flag existed, CLI never wired it)
- [x] Load failure exits 2 (1 is reserved for findings)
- [x] Allowlist: empty `pathPattern` = project-wide (was a silent no-op);
  rule-less entries warn and stay inert (e2e test added)
- [x] Suppression directives honored anywhere inside a multi-line registration
  call (`suppressspan` fixture)
- [x] HW-0 fires for orphaned malformed directives, at the comment
  (`hw0orphan` fixture + test)
- [x] A74 (real) edge fixtures: duplicate registrations, nested closures (`edges`)
- [x] A75 (real) go.work multi-module e2e test (workspace pattern: `all`)
- [x] A81 (real) `golangci-lint custom` build + fire verified end to end;
  locked in as `plugin/plugin_integration_test.go`; `.custom-gcl.yml` usage
  comment now names the mandatory `linters.settings.custom` registration
- [x] A96 (real) `docs/FP-BUDGETS.md` with per-rule budgets + ecology evidence
- [x] Ecology anomalies explained: Kernovia (go.work needs go >= 1.27) and
  ast-state-analyzer (stale go.mod) are target-project breakage, not analyzer
  bugs; both fail cleanly with an explanatory message

Open decisions (user):

- [ ] HW-4 default posture: stay on-by-default `info` vs opt-in
- [ ] Commit `.samber-linter-baseline.json` at 8% into CV; triage CV's 9 findings
- [ ] GitHub purge of `.crush` history blobs (support ticket vs delete+recreate)
- [ ] Candidate HW-7 "stale directive": valid-but-orphaned directives with an
  `until` expiry could resurface for cleanup (see docs/FP-BUDGETS.md)

## Tier 1 — 1% → 51%: Incident-class MVP

### M01 Spec amendments (README)

- [x] A01 Fix §2.5: `ShutdownerWithError` is `Shutdown() error`, not `ShutdownWithError() error`; add method-name matrix
- [x] A02 Add Override* family (di.go:187-251) to §4 step 1 + §2.6 table
- [x] A03 Add As/AsNamed semantics + HW-6 alias-dedupe policy
- [x] A04 Add fixture-freezing rule to §9 (testdata snapshots, never live CV refs)
- [x] A05 Version-awareness requirement + v2.0.0 drift matrix in §11
- [x] A06 Rename hw-suppression-reason-missing → HW-0
- [x] A07 Add transient `isHealthchecker()==false` fact to §2.4/§7
- [x] A08 Ledger note: 60 → 61 registration sites, counts drift

### M02 Scaffold

- [x] A09 `go mod init github.com/larsartmann/samber-linter` + x/tools + samber/do v2.1.0 test dep
- [x] A93 Add deps: go-finding (+ `analysis` subpackage) + go-atomic-write; pin versions
- [x] A10 internal/healthwash + cmd/samber-linter skeletons
- [x] A11 flake.nix: build/test/lint/flake-check/devShell
- [x] A12 .gitignore + minimal .golangci.yml
- [x] A13 nix build + nix flake check green

### M03 Registration-site matcher

- [x] A14 Package-path resolution helper (never identifier text)
- [x] A15 Cover 6 Provide* selectors
- [x] A16 Cover 6 Override* selectors
- [x] A17 Dot-import + renamed-import cases

### M04 Service-type resolver

- [x] A18 Provider closure first-return extraction
- [x] A19 Drop error return; handle Provider[T] shape
- [x] A20 ProvideValue* value-argument type
- [x] A21 Named-type / alias chasing
- [x] A22 Interface-typed registration → concrete return type
- [x] A23 Unresolvable → silent default, --strict hook

### M05 Method-set engine

- [x] A24 T vs *T computation from types.Info
- [x] A25 Unit tests incl. embedded-type promotion
- [x] A26 Corrected lifecycle facts table in code

### M06/M07 HW-1 + HW-5

- [x] A27 HW-1 evaluation logic
- [x] A28 HW-1 message + call-site attribution
- [x] A29 HW-5 evaluation: value type expr + HealthCheck only on *T
- [x] A30 HW-5 message + fix suggestion (register *T)

### M08 Frozen fixture corpus

- [x] A31 Snapshot graphrag.Store (Shutdowner-only) from CV HEAD
- [x] A32 Snapshot handler/config negatives
- [x] A33 Hand fixture: value reg of pointer-receiver HealthCheck type
- [x] A34 Hand fixture: clean HealthcheckerWithContext implementer

### M09 Harness

- [x] A35 analysistest setup
- [x] A36 Golden expectations per fixture
- [x] A37 Compile gate: every fixture type-checks

### M10 Discrimination proofs

- [x] A38 Mutant HW-1 in scratch copy → fixtures fail
- [x] A39 Mutant HW-5 → fixtures fail
- [x] A40 Record results in ledger

### M11 CLI

- [x] A41 singlechecker main.go wiring
- [x] A42 End-to-end run on fixture module
- [x] A94 Driver: run analyzer via `analysis.NewAnalyzerDetector` → `finding.Report`
- [x] A95 Confidence stamping per rule via `Builder.WithConfidence` (Diagnostic carries none)

## Tier 2 — 4% → 64%: Trustworthy analyzer

### M12 Dogfood

- [x] A43 Run on CV HEAD; compare vs §1 expectations
- [x] A44 Run on branching-flow; triage false positives
- [x] A45 Fix top false-positive class

### M13/M14/M17 Rules

- [x] A46 HW-3 evaluation (transient + implementer)
- [x] A47 HW-3 fixture incl. OverrideTransient
- [x] A48 HW-4 evaluation (lazy + implementer, info)
- [x] A49 HW-4 fixture
- [x] A50 Suppression parser `//samber-linter:allow hw-N <reason>`
- [x] A51 Attach suppressions to registration sites
- [x] A52 HW-0 rule: reason missing
- [x] A53 Suppression fixtures
- [x] A97 Map suppressions onto go-finding `Suppression{Kind, Rule, Reason, ExpiresAt}` model; decide `until` syntax
- [x] A58 HW-2 evaluation (contextless, info)
- [x] A59 HW-2 fixture

### M16 CI + drift matrix

- [x] A54 GitHub Actions: nix build/test/lint
- [x] A55 Mechanism-assertion test vs v2.1.0 pins
- [x] A56 Drift matrix on v2.0.0
- [x] A57 CI green verification

### M18 Allowlist

- [x] A60 Config allowlist loading (path patterns)
- [x] A61 Allowlist fixture test

### M28 Trust engineering

- [x] A96 Per-rule FP budgets (HW-1 < 1% on CV + branching-flow) + severity/confidence matrix doc (HW-1/3/5 Full, HW-2 High, HW-4 Medium)

## Tier 3 — 20% → 80%: CI-gate product

### M19 HW-6 ratchet

- [x] A62 Coverage calculation with alias dedupe
- [x] A63 Baseline file read
- [x] A64 --set-baseline write
- [x] A65 --coverage-min gate + exit code
- [x] A66 Ratchet integration test
- [x] A98 Ratchet as Report post-pass in driver (project-level, never per-package Analyzer.Run)
- [x] A99 Baseline read + --set-baseline via `atomicwrite.WriteIfChanged` (idempotent)

### M20/M21 CLI polish + version awareness

- [x] A67 --json output
- [x] A68 --check exit 2 + --strict semantics
- [x] A69 --strict enables HW-unresolved
- [x] A70 Help text + README CLI sync
- [x] A71 Read target go.mod samber/do version
- [x] A72 Info finding outside verified set
- [x] A100 SARIF export via go-finding sarif package
- [x] A101 Exit codes 0/1/2 via `linter.ExitCodeByConfidence`

### M22 Hardening

- [x] A73 Fuzz never-panics harness
- [x] A74 Edge fixtures: duplicate types, nested closures
- [x] A75 go.work multi-module fixture

### M23 Docs

- [x] A76 README quickstart rewrite
- [x] A77 FEATURES.md scaffold
- [x] A78 CHANGELOG.md first entry
- [x] A79 TODO_LIST.md upkeep

### M24 Plugin

- [x] A80 golangci-lint plugin module layout
- [x] A81 Build verified against pinned golangci version
- [x] A102 Copy go-humanize-linter `plugin/plugin.go` wiring (`.custom-gcl.yml` → `golangci-lint custom`)

## Tier 4 — 80% → 100%: Ecosystem

- [x] A82 Copy HW rules → doanalyzerv2 DO-9 family
- [x] A83 DO-9 fixture port
- [x] A84 doanalyzerv2 suite green
- [x] A85 Failing test: transient Healthchecker never runs
- [x] A86 Draft verified upstream issue
- [x] A87 healthaudit skeleton (auditlog wrap pattern)
- [x] A88 Count checked-vs-skipped per sweep
- [x] A89 Expose healthwash_registered / healthwash_checked metrics
- [x] A90 README §7 runtime integration doc
- [x] A91 Website / launch decision
- [x] A92 Tag v0.1.0 release

## Stack adoption (verified 2026-09-09)

- Detection core stays a plain `*analysis.Analyzer` (golangci plugin contract)
- go-finding: finding model at driver chokepoint (`analysis.FromDiagnostic` /
  `NewAnalyzerDetector`), severity AND confidence axes, `Suppression` data model
  with expiry, JSON + SARIF export
- go-linter-sdk: `ExitCodeByConfidence` only; Registry core deliberately NOT
  adopted (directory-scoped, samber-linter needs cross-package types)
- go-atomic-write: `WriteIfChanged` for idempotent baseline writes
- go-humanize-linter: golangci v2 module-plugin wiring template
- samber-do-auditlog: wrap pattern for the P3 healthaudit companion

## Done

2026-09-09: A01–A102 executed and verified (go build/vet/test green; nix flake
check green; drift matrix v2.0.0+v2.1.0 green; dogfood: CV 9 findings /
8% coverage, branching-flow clean).
