# TODO_LIST — samber-linter

Living task source: **open work only**. Completed items are deleted on sight
and live in `CHANGELOG.md`; directional ideas live in `ROADMAP.md`;
point-in-time session detail lives in `docs/status/`. The full v0.1.0
execution plan is historical:
`docs/planning/2026-09-09_20-09_SUPERB-PARETO-EXECUTION-PLAN.html`.

## Quality gate

- [x] **First all-green CI run: ACHIEVED 2026-09-16.** Run
      [35126418813](https://github.com/LarsArtmann/samber-linter/actions/runs/35126418813)
      — all five jobs green (`test`, `dogfood`, `drift-matrix`,
      `upstream-snippets`, and `lint` on golangci-lint v2.13.2 via
      golangci-lint-action v7). Getting there took two more real fixes the
      red runs had masked: action v6 rejects v2 version pins entirely, and
      the first actually-executing lint run flagged three findings. Keep
      claiming "CI green" only while the latest run says so.

## Short-term work queue (bounded, actionable)

- [ ] **After the next push + release:** tag the schema-v2 release
      (baseline v2, per-module go.work scanning, shared loader, snippet-gate
      extension are all unreleased as of v0.2.1), then wire the health-wash
      CI job into `samber-do-auditlog` and `standard-bug-tracking-schema`
      pinning that version. Both repos already carry committed v2 baselines
      (auditlog 12/20 = 60%, standard-bug 2/61 = 3%) and an AGENTS.md note
      describing the exact pending step; a v0.2.1 pin would fail closed on
      schema v2.
- [ ] **cmdguard (its repo, not here):** `*CLI[T]` implements a bare
      `HealthCheck()` while registered via `Package` (`do.ProvideValue`) —
      the ecology survey's only true HW-2 (ctx variant
      `HealthCheckWithContext` already exists next to it). Decide there:
      satisfy `do.Healthchecker` (naming collision with the existing bare
      method — needs an API decision) or suppress with a documented reason.
      CV is unaffected: its nine workspace modules scan clean, and its
      schema-v2 baseline (10/30 = 33%) is committed.
- [ ] Refresh the pseudonymous triage doc with the post-fix ecology numbers
      (`docs/ecology/2026-09-16-scan*.txt` + `scripts/ecology-scan.sh`
      output supersede the 2026-09-10 table).

## Open decisions (user)

- [ ] **Push authorization:** master carries the v2.13.2 lint pin, baseline
      v2, and this round's fixes; none of it is observable in CI until
      pushed (never push without an explicit go-ahead).
- [ ] **HW-4 default posture:** on-by-default `info`/Medium vs opt-in.
      24 of 66 ecology findings were HW-4; it is the main noise dial
      (docs/FP-BUDGETS.md soft ceiling ≈ 20%).
- [ ] **GitHub `.crush` history purge:** support ticket vs delete+recreate
      (untracked since f441f34; history blobs remain).
- [ ] **HW-7 "stale directive":** valid-but-orphaned directives with an
      `until` expiry resurface for cleanup — rule candidate
      (docs/FP-BUDGETS.md).
- [ ] **Upstream ownership:** who watches samber/do#317 and #318 (both OPEN,
      no maintainer response as of 2026-09-10), on what response SLA, and
      whether design notes beyond #318 wait for a maintainer ask.
