# TODO_LIST — samber-linter

Living task source. Snapshot plan with full rationale, impact/effort scoring, and
execution graph: `docs/planning/2026-09-09_20-09_SUPERB-PARETO-EXECUTION-PLAN.html`.
Mark items `[x]` when done and move them to the Done section at the bottom.

## Tier 1 — 1% → 51%: Incident-class MVP

### M01 Spec amendments (README)

- [ ] A01 Fix §2.5: `ShutdownerWithError` is `Shutdown() error`, not `ShutdownWithError() error`; add method-name matrix
- [ ] A02 Add Override* family (di.go:187-251) to §4 step 1 + §2.6 table
- [ ] A03 Add As/AsNamed semantics + HW-6 alias-dedupe policy
- [ ] A04 Add fixture-freezing rule to §9 (testdata snapshots, never live CV refs)
- [ ] A05 Version-awareness requirement + v2.0.0 drift matrix in §11
- [ ] A06 Rename hw-suppression-reason-missing → HW-0
- [ ] A07 Add transient `isHealthchecker()==false` fact to §2.4/§7
- [ ] A08 Ledger note: 60 → 61 registration sites, counts drift

### M02 Scaffold

- [ ] A09 `go mod init github.com/larsartmann/samber-linter` + x/tools + samber/do v2.1.0 test dep
- [ ] A10 internal/healthwash + cmd/samber-linter skeletons
- [ ] A11 flake.nix: build/test/lint/flake-check/devShell
- [ ] A12 .gitignore + minimal .golangci.yml
- [ ] A13 nix build + nix flake check green

### M03 Registration-site matcher

- [ ] A14 Package-path resolution helper (never identifier text)
- [ ] A15 Cover 6 Provide* selectors
- [ ] A16 Cover 6 Override* selectors
- [ ] A17 Dot-import + renamed-import cases

### M04 Service-type resolver

- [ ] A18 Provider closure first-return extraction
- [ ] A19 Drop error return; handle Provider[T] shape
- [ ] A20 ProvideValue* value-argument type
- [ ] A21 Named-type / alias chasing
- [ ] A22 Interface-typed registration → concrete return type
- [ ] A23 Unresolvable → silent default, --strict hook

### M05 Method-set engine

- [ ] A24 T vs *T computation from types.Info
- [ ] A25 Unit tests incl. embedded-type promotion
- [ ] A26 Corrected lifecycle facts table in code

### M06/M07 HW-1 + HW-5

- [ ] A27 HW-1 evaluation logic
- [ ] A28 HW-1 message + call-site attribution
- [ ] A29 HW-5 evaluation: value type expr + HealthCheck only on *T
- [ ] A30 HW-5 message + fix suggestion (register *T)

### M08 Frozen fixture corpus

- [ ] A31 Snapshot graphrag.Store (Shutdowner-only) from CV HEAD
- [ ] A32 Snapshot handler/config negatives
- [ ] A33 Hand fixture: value reg of pointer-receiver HealthCheck type
- [ ] A34 Hand fixture: clean HealthcheckerWithContext implementer

### M09 Harness

- [ ] A35 analysistest setup
- [ ] A36 Golden expectations per fixture
- [ ] A37 Compile gate: every fixture type-checks

### M10 Discrimination proofs

- [ ] A38 Mutant HW-1 in scratch copy → fixtures fail
- [ ] A39 Mutant HW-5 → fixtures fail
- [ ] A40 Record results in ledger

### M11 CLI

- [ ] A41 singlechecker main.go wiring
- [ ] A42 End-to-end run on fixture module

## Tier 2 — 4% → 64%: Trustworthy analyzer

### M12 Dogfood

- [ ] A43 Run on CV HEAD; compare vs §1 expectations
- [ ] A44 Run on branching-flow; triage false positives
- [ ] A45 Fix top false-positive class

### M13/M14/M17 Rules

- [ ] A46 HW-3 evaluation (transient + implementer)
- [ ] A47 HW-3 fixture incl. OverrideTransient
- [ ] A48 HW-4 evaluation (lazy + implementer, info)
- [ ] A49 HW-4 fixture
- [ ] A50 Suppression parser `//samber-linter:allow hw-N <reason>`
- [ ] A51 Attach suppressions to registration sites
- [ ] A52 HW-0 rule: reason missing
- [ ] A53 Suppression fixtures
- [ ] A58 HW-2 evaluation (contextless, info)
- [ ] A59 HW-2 fixture

### M16 CI + drift matrix

- [ ] A54 GitHub Actions: nix build/test/lint
- [ ] A55 Mechanism-assertion test vs v2.1.0 pins
- [ ] A56 Drift matrix on v2.0.0
- [ ] A57 CI green verification

### M18 Allowlist

- [ ] A60 Config allowlist loading (path patterns)
- [ ] A61 Allowlist fixture test

## Tier 3 — 20% → 80%: CI-gate product

### M19 HW-6 ratchet

- [ ] A62 Coverage calculation with alias dedupe
- [ ] A63 Baseline file read
- [ ] A64 --set-baseline write
- [ ] A65 --coverage-min gate + exit code
- [ ] A66 Ratchet integration test

### M20/M21 CLI polish + version awareness

- [ ] A67 --json output
- [ ] A68 --check exit 2 + --strict semantics
- [ ] A69 --strict enables HW-unresolved
- [ ] A70 Help text + README CLI sync
- [ ] A71 Read target go.mod samber/do version
- [ ] A72 Info finding outside verified set

### M22 Hardening

- [ ] A73 Fuzz never-panics harness
- [ ] A74 Edge fixtures: duplicate types, nested closures
- [ ] A75 go.work multi-module fixture

### M23 Docs

- [ ] A76 README quickstart rewrite
- [ ] A77 FEATURES.md scaffold
- [ ] A78 CHANGELOG.md first entry
- [ ] A79 TODO_LIST.md upkeep

### M24 Plugin

- [ ] A80 golangci-lint plugin module layout
- [ ] A81 Build verified against pinned golangci version

## Tier 4 — 80% → 100%: Ecosystem

- [ ] A82 Copy HW rules → doanalyzerv2 DO-9 family
- [ ] A83 DO-9 fixture port
- [ ] A84 doanalyzerv2 suite green
- [ ] A85 Failing test: transient Healthchecker never runs
- [ ] A86 Draft verified upstream issue
- [ ] A87 healthaudit skeleton (auditlog wrap pattern)
- [ ] A88 Count checked-vs-skipped per sweep
- [ ] A89 Expose healthwash_registered / healthwash_checked metrics
- [ ] A90 README §7 runtime integration doc
- [ ] A91 Website / launch decision
- [ ] A92 Tag v0.1.0 release

## Done

(none yet)
