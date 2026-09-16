# TODO_LIST — samber-linter

Living task source: **open work only**. Completed items are deleted on sight
and live in `CHANGELOG.md`; directional ideas live in `ROADMAP.md`;
point-in-time session detail lives in `docs/status/`. The full v0.1.0
execution plan is historical:
`docs/planning/2026-09-09_20-09_SUPERB-PARETO-EXECUTION-PLAN.html`.

## Quality gate

- [ ] **First all-green CI run still unobserved.** The `GOEXPERIMENT=jsonv2`
      fix made `test`/`dogfood`/`drift-matrix`/`upstream-snippets` green
      (run 35089293309, 2026-09-16), but `lint` stayed red because
      `golangci-lint-action` `version: latest` resolves to a **v1** binary
      (v1.64.8) that cannot load the v2 config. The v2.13.2 pin (matching
      `.custom-gcl.yml` and nixpkgs) lands with the next push — watch the run
      before claiming the gate green anywhere.

## Short-term work queue (bounded, actionable)

- [ ] Re-baseline CV: its committed baseline is schema v1; baseline v2 now
      fails the gate loudly with the migration hint. Run
      `samber-linter --set-baseline ./...` in CV and commit the v2 file.
- [ ] Baselines/gates for samber-do-auditlog (coverage 60%) and
      standard-bug-tracking-schema (3%): commit v2 baselines and wire
      samber-linter into their CI so the fixed state cannot regress silently.
- [ ] Refresh the pseudonymous triage doc with the post-fix ecology numbers
      (`scripts/ecology-scan.sh` output supersedes the 2026-09-10 table).
- [ ] Snippet gate over `docs/status/**` Go blocks too (currently only
      `docs/upstream`).

## Open decisions (user)

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
