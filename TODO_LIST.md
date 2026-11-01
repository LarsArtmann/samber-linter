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

- [ ] **CHANGELOG entry for the instantiated-call fix (`50df0ad` + follow-ups)**
      — shipped 2026-10-31 with code and tests but no changelog entry, violating
      this file's own "completed items live in CHANGELOG.md" rule (harvested from
      the 2026-11-01 storbi closeout report §d-5).
- [ ] **Push + release decision**: ~8 local commits (instantiated-call detection,
      GOWORK test isolation, dprint fix, debt records) sit unpushed on master;
      consumers get nothing until push + tag. Owner call — harvested 2026-11-01.
- [ ] **Instantiated cross-package fixture** (foreignprov-inst variant): the
      five-shape repro proved instantiated cross-package providers land as
      HW-unresolved, but no testdata pins that shape (harvested 2026-11-01).
- [ ] **Sandbox lint debt: gocognit 30 on `TestBuildDriverOptions`**
      (`cmd/samber-linter/options_test.go:15`, limit 25; file untouched since
      2026-10-02). Pre-existing — surfaced by the BuildFlow nix-build gate on
      2026-10-31 while landing the instantiated-call fix (`50df0ad`); it is
      the one finding keeping that gate's `lint` check red (historically
      83% failed runs). Fix by extracting subtests/table cases; do NOT bump
      the gocognit threshold.
- [x] **Release the post-v0.2.2 batch: DONE 2026-09-23.** Cut v0.3.0
      (HW-8 + wrapper-indirection detection + single-source rule table) with
      the README §12 release flow.
- [ ] **Post-v0.4.0 follow-ups from the 2026-10-02 cmdguard session**
      (detail: `docs/status/2026-10-02_11-42_cmdguard-cli-adoption-and-stale-fixture-completion.md`
      §f): re-run `scripts/ecology-scan.sh` as the post-resolver-change
      regression proof (needs a go ≥ 1.27.1 scanner toolchain); re-run the
      old-vs-new binary parity on a findings-bearing module for the FINAL
      binary; file the tagalign fixer/checker tag-order disagreement against
      golangci-lint v2.14.0 (verify-before-filing first); decide whether
      README §5/§11 should name cmdguard as the CLI framework. DONE from
      that list on 2026-10-02 later session: golangci pin alignment to
      v2.14.0 + drift guards, CLI exit-code harness, `-help` fix,
      `--check`-on-load-failure fix, resolver fixtures (diverging,
      foreignprov, hwwrap/hw7cross composition pins).
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
