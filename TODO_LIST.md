# TODO_LIST — samber-linter

Living task source: **open work only**. Completed items are deleted on sight
and live in `CHANGELOG.md`; directional ideas live in `ROADMAP.md`;
point-in-time session detail lives in `docs/status/`. The full v0.1.0
execution plan is historical:
`docs/planning/2026-09-09_20-09_SUPERB-PARETO-EXECUTION-PLAN.html`.

## Quality gate (red today)

- [ ] **CI `lint` job is red, two layers deep.** (a) `golangci-lint-action`
      `version: latest` ships a binary built with go1.24, which refuses go.mod
      1.26.7 ("can't load config", run 34425222926) — pin a golangci version built
      with go ≥ 1.26 and adopt one version across CI / nixpkgs /
      `.custom-gcl.yml` (v2.12.2) / local. (b) Behind that: ~141 pre-existing
      findings (paralleltest on analysistest-based tests, varnamelen in legacy
      code, 2× gochecknoglobals tables, wrapcheck on `packages.Load`, drift_test
      gocognit/lll). Burn down in a dedicated pass; never mix into feature work.
      Evidence: `gh run view 34425222926`, `nix run .#lint`.
- [ ] **First green CI run unconfirmed** — the `GOEXPERIMENT=jsonv2` fix for
      the `test`/`dogfood` jobs and the new `upstream-snippets` job (2026-09-10)
      have not been observed passing on GitHub yet; `drift-matrix` was the only
      job ever green.
- [ ] **Ecology drift:** CV fixed its 9 findings after the scan; the committed
      8% baseline now reads 10% and correctly suggests `--set-baseline`.

## Short-term work queue (bounded, actionable)

- [ ] Tag **v0.1.1** — blocked on user go-ahead (decision below). Contains the
      P0 plain-invocation fix; until tagged, README's `@latest` serves v0.1.0
      (no `--output`/`--check`/`--disable`).
- [ ] Make the 43-project ecology scan a repo script
      (`scripts/ecology-scan.sh`, pseudonymous output) and re-run it as the
      post-change regression proof (source: docs/status/2026-09-10_04-05 §e.1).
- [ ] Baseline v2: per-rule finding counts + schema version + validation
      errors; aggregate-only coverage can hide a per-rule regression
      (docs/status/2026-09-10_04-05 §e.5).
- [ ] `--check` advisory line pollutes `--json`/`--sarif` output — suppress it
      in machine formats (docs/status/2026-09-10_04-05 §e.6).
- [x] Prepare the samber/do#317 option-1 PR branch in advance (sentinel
      `ErrHealthCheckSkipped`) so filing a PR is a five-minute act if invited
      — DONE 2026-09-10: branch `transient-healthcheck-sentinel` pushed to
      LarsArtmann/do (remote diff verified vs upstream master).
- [x] Ecology triage execution, biggest offenders first:
      samber-do-auditlog (13 → 0 findings — its HW-3 was the live #317 case;
      transient notifier → named singleton), rank-1
      standard-bug-tracking-schema (4×HW-1 → 0, two real checks + judged
      suppressions) — DONE 2026-09-10.
- [ ] Dogfood `--output markdown` in the CI dogfood job (the flag can rot
      silently otherwise; source: docs/status/2026-09-10_03-00 §c.3).
- [ ] Integrate dprint (markdown/json/yaml) into a nix check — treefmt owns
      go/nix only, so hand-edited markdown is currently unverified by
      `nix flake check` (docs/status/2026-09-10_02-22 §b.6).

## Open decisions (user)

- [ ] **HW-4 default posture:** on-by-default `info`/Medium vs opt-in.
      24 of 66 ecology findings were HW-4; it is the main noise dial
      (docs/FP-BUDGETS.md soft ceiling ≈ 20%).
- [x] **Release policy:** v0.1.1 CUT 2026-09-10 (P0 fix + hardening round;
      tag-only, matching v0.1.0) — resolved.
- [x] **CV baseline:** committed at 10% via --set-baseline (5/51), 0
      findings on scan — resolved.
- [ ] **GitHub `.crush` history purge:** support ticket vs delete+recreate
      (untracked since f441f34; history blobs remain).
- [ ] **HW-7 "stale directive":** valid-but-orphaned directives with an
      `until` expiry resurface for cleanup — rule candidate
      (docs/FP-BUDGETS.md).
- [ ] **Upstream ownership:** who watches samber/do#317 and #318 (both OPEN,
      no maintainer response as of 2026-09-10), on what response SLA, and
      whether design notes beyond #318 wait for a maintainer ask.
