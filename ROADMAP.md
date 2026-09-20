# ROADMAP — samber-linter

Long-term direction and raw ideas. Bounded, actionable work lives in
`TODO_LIST.md`; shipped features live in `FEATURES.md`. Items here are
unrefined: they graduate to TODO_LIST when they gain a bounded first step.
Sources are the session reports in `docs/status/` (sections marked "50
things" are brainstorm-grade input, not commitments).

## Theme 1 — The runtime health-outcome triangle

The static analyzer says a check _cannot fail_; the runtime says what
_actually happened_. samber/do#317 (transient checks never dispatched) and
#318 (explicit outcome states) are the upstream anchor.

- healthaudit: typed `Status` enum (Registered/Invoked/Errored/Skipped) and
  sweep outcomes as data, not just counters (06-50 §f.9-10).
- go-health: extend `Check.Status` with `unknown`/`skipped`, fed by
  healthaudit outcomes; classifier lattice (Unknown < Pass < Warn < Fail);
  go-health-dashboard renders the new states distinctly (06-50 §f.11-16).
- HW-1/3/4 → runtime mapping doc: "what the dashboard should show once
  states exist" (06-50 §f.15).
- Watch both issues; a maintainer-picked #317 direction may spawn a PR.

Why it matters: closes the residual class static analysis cannot see
("implements but always returns nil") and makes HW findings runtime-visible.

## Theme 2 — Quality-gate unification

Three gates exist today (GitHub Actions, `nix flake check`, golangci plugin
build) with three golangci versions and two GOEXPERIMENT stances.

- Decide the CI↔nix relationship: thin `nix flake check` runner (hermetic,
  single gate; needs deploy-key auth for the `git+ssh` go-nix-helpers input)
  vs setup-go for non-nix OSS contributors (02-22 §g.1).
- One golangci-lint version policy across CI / nixpkgs / `.custom-gcl.yml`.
- `nix flake check --all-systems` (darwin/aarch64 unexercised); dprint under
  a nix check; binary cache (cachix/attic) for CI speed; Renovate for
  go.mod + flake.lock (02-22 §f.18-21).

Exit condition: `nix flake check` is a true merge gate on every tree.

## Theme 3 — Analyzer growth

- **Stale-directive rule (renumbered — the HW-7 ID shipped as
  `unconditional-nil-check` in v0.2.2):** valid-but-orphaned suppressions
  (esp. with `until`) resurface for cleanup — design noted in
  docs/FP-BUDGETS.md; bounded first step in TODO_LIST.
- Config auto-discovery (`samber-linter.yml` in repo root; `--config`
  overrides) (04-05 §f.16).
- `samber-linter explain HW-N`: rule docs from the code single-source
  (04-05 §f.20).
- Drift matrix extension: cover future samber/do v2.1.x/v2.2.x releases as
  they land; keep the verified-set info finding honest (03-00 §f.39).
- Corpus growth: provider methods (`(*Svc).New`), interface-satisfying
  generics, more mutant discrimination per new case (04-05 §f.24).
- Performance benchmark on the largest consumer (63 services) — load-time
  budget (04-05 §f.29).
- Worth considering (from FEATURES): `--fix` for HW-5, SARIF
  `--include-suppressed`, per-scope coverage breakdown, suppression age
  report (`--suppressions`), SARIF coverage property.

## Theme 4 — Ecology program

The 2026-09-10 scan (43 consumers, 66 findings, 0 confirmed FPs) proved the
tool on real code. The program: repeatable pseudonymous scan as a repo
script, then fix the worst offenders in priority order (see the pseudonymous
queue `docs/status/2026-09-10_02-40_ECOLOGY-TRIAGE-PSEUDONYMOUS.md`).
Findings fixed in target projects are the strongest possible FP evidence.
2026-09-20 state: 72 discovered consumers, 32 LOAD_ERROR because their
toolchains moved past the local go floor — ecology coverage now depends on
the toolchain-bump item in Theme 5.

## Theme 5 — Distribution & adoption

- GitHub Release automation (version provenance is solved as of v0.2.2:
  proxy tag for `go install …@vX`, `devel+<rev>` for source/nix builds —
  what remains is Release-chore automation, 02-22 §g.3).
- Public website: deferred until adoption exists (FEATURES decision);
  revisit criteria still undefined (01-17 §b.3).
- Go toolchain bump workflow readiness (goTarballVersion in go-standard) —
  now with live evidence: 32 of 72 ecology consumers require go ≥ 1.27.1
  while nixpkgs pins 1.26.7 (`docs/ecology/2026-09-20-scan.txt`).

## Open questions (user-owned)

Default output direction: plain text forever (CI-log compatibility) vs
styled table default with text opt-out — decides how much polish `--output
table` deserves (03-00 §g.2).

Recorded where the work is tracked: the `--min-confidence` default flip
(one release of max-recall-profile FP data first), CV runbook note, `.crush`
purge, upstream watch ownership — see `TODO_LIST.md` → "Open decisions".
CI↔nix policy and jsonv2-invariant ownership are partially answered (jsonv2
is now a hard build requirement via go-finding v1.9.2 and CI sets it).
Report format standing: **markdown is the default for status/planning
docs** (decided 2026-09-20; the single HTML plan was a one-off — HTML
remains available for diagram-heavy deliverables on explicit request).
