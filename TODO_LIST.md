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

- [ ] **Wire consumer ratchet jobs onto v0.2.2** (source: "tag the schema-v2
      release" — the tagging half is done, v0.2.2 shipped 2026-09-20):
      `samber-do-auditlog` and `standard-bug-tracking-schema` pin `@v0.2.2`
      in their healthwash CI jobs; both already carry committed v2 baselines
      (auditlog 12/20 = 60%, standard-bug 2/61 = 3%).
- [ ] **CV runbook note** (source: plan T5.8): CV's healthwash gate can now
      version-gate the analyzer (`go run …@v0.2.2 -version` → `v0.2.2`,
      verified via the module proxy 2026-09-20). Draft the runbook paragraph
      for CV's `scripts/healthwash.sh` pin bump — the v0.2.1 pin still
      carries both driver defects fixed in v0.2.2.
- [ ] **Renumber the stale-directive rule candidate to HW-9+** (source:
      "HW-7 stale directive" — the HW-7 ID is now taken by
      `unconditional-nil-check`, released v0.2.2). Valid-but-orphaned
      suppressions with `until` expiry resurface for cleanup; design noted in
      `docs/FP-BUDGETS.md`. Assign the next free ID before designing.
- [ ] **Default-rules single source** (source: lessons report item 14,
      `docs/status/2026-09-16_13-27_cross-project-lessons-go-auto-upgrade.md`):
      one `defaultRules` definition consumed by the driver, the plugin, and
      the README rule-table drift test, so posture changes are one-line.
- [ ] **Wrapper-indirection false negative** (source: 2026-09-20 consumer fix
      round): `inspectRegistration` matches only direct `do.*` calls
      (`pkg/healthwash/healthwash.go` ~L197), so registrations hidden behind
      repo-local helper functions/generics (e.g. a `provideNamed[T]` wrapper
      around `do.ProvideNamed`) are invisible to EVERY rule. Verified case: a
      service implementing `HealthcheckerWithContext`, lazily registered
      through such a wrapper, produced zero findings. Fix idea: resolve
      one-level local wrappers (param→arg mapping into the wrapped `do.*`
      call); start with a testdata fixture reproducing the shape.
- [ ] Refresh the pseudonymous triage doc with the post-fix ecology numbers
      (`docs/ecology/2026-09-20-scan.txt` supersedes both 2026-09-16 scans;
      note the load-error wave — 32 consumers require go ≥ 1.27.1).

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
