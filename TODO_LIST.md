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

- [x] **Release the post-v0.2.2 batch: DONE 2026-09-23.** Cut v0.3.0
      (HW-8 + wrapper-indirection detection + single-source rule table) with
      the README §12 release flow.
- [ ] **Evaluate wrapper-channel findings in the ecology** (follow-up to the
      v0.3.0 release): re-scan needs a go ≥ 1.27.1 scanner toolchain —
      32 consumers are in the load-error wave until then.
- [ ] **Mutant-proof the wrapper channel further** (optional hardening):
      the `hwwrap` fixture proves the channel via HW-1; consider a dedicated
      negative fixture for wrapper chains (wrapper-calling-wrapper) once a
      real consumer shape is observed.

## Open decisions (user)

- [ ] **`--min-confidence` default flip:** profile-first was recorded
      2026-09-20 (README "Threshold policy"): the max-recall profile
      (`--min-confidence 0.5 --strict`) is opt-in until one release of
      measured FP data exists; then decide the default flip.
- [ ] **GitHub `.crush` history purge:** support ticket vs delete+recreate
      (untracked since f441f34; history blobs remain).
- [ ] **Upstream ownership:** who watches samber/do#317 and #318 (both OPEN,
      no maintainer response as of 2026-09-10), on what response SLA, and
      whether design notes beyond #318 wait for a maintainer ask.
- [ ] **HW-9 `stale-directive` go-ahead:** design + FP budget recorded in
      `docs/FP-BUDGETS.md` (2026-09-20); ships only on explicit user
      approval.
