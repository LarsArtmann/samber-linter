# Status Report — Green Lint, v0.1.1 Release & Ecology Fix Round

Date: 2026-09-10, 08:55. Source: `docs/status/2026-09-10_06-50_UPSTREAM-FILING-AND-VERIFICATION.md`
50-item list, executed top-down. Release: **v0.1.1 shipped** (tag pushed,
remote verified). First-ever green lint: 44 → 0 findings, `nix flake check`
all checks pass.

## TL;DR

- Lint burn-down: **44 → 0 findings** (was "141" per stale nix figure; the
  hermetic run's real count was 44). First fully green golangci-lint run,
  first green `nix flake check` (build + full suite + hermetic lint +
  treefmt) on this tree.
- **v0.1.1 released**: version bumped, CHANGELOG/README updated, annotated
  tag pushed (tag-only, matching the v0.1.0 convention; no GitHub Release
  page). `go install` consumers get `--output` + all hardening fixes.
- **#317 option-1 PR branch is ready and pushed** to `LarsArtmann/do`
  (`transient-healthcheck-sentinel`): `ErrHealthCheckSkipped` sentinel,
  scope-doc fix, sweep-level test, all sentinel tests pass; base = upstream
  master, remote diff verified. Opening the PR stays a 5-minute act.
- **Ecology: all three targets clean.** samber-do-auditlog 13 → 0 findings
  (incl. its live #317 case: transient email-notifier → named singleton);
  standard-bug-tracking-schema 4×HW-1 → 0 (two real checks + judged
  suppressions); CV baseline locked at 8% → 10% (`--set-baseline`).
- `.config/metadata.yaml` mystery resolved: tool-owned timestamp/tag churn,
  daemon-committed, not a foreign edit.

## a) FULLY DONE

| Item                                       | Result                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Lint burn-down (items 20/4)                | 44 findings (11 classes) → 0. paralleltest/tparallel 14, varnamelen 14, wrapcheck 3 (all wrapped with context), unparam 2, lll 4, gocognit 1 (decomposed `assertMechanism` into `collectIfaces`/`collectFuncNames`), globals 2 (`VerifiedDover` → `VerifiedDoVersions()` function, table nolinted with reason), forbidigo 1, golines/wsl auto-fixed. All tests green incl. `-race`; plugin custom-gcl test still passes                                                                                                                                                                             |
| Nix flake check green                      | `nix flake check` → "all checks passed" (first time complete)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Snippet gate (17)                          | `scripts/check-upstream-snippets.sh`: compiles AND runs every `docs/upstream` Go block verbatim; `` ```go snippet-skip `` for upstream quotes; unclassified blocks fail. CI job `upstream-snippets` added. Negatively tested (broken snippet exits 1). Repro in ISSUE_DRAFT.md is now a self-contained runnable program in lockstep with the filed issue                                                                                                                                                                                                                                            |
| v0.1.1 release (19)                        | main.go `0.1.1`, CHANGELOG `[Unreleased]` → `[0.1.1] - 2026-09-10`, README notes updated; annotated tag `v0.1.1` pushed (master synced 0/0); tag visible remotely                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| #317 PR branch (2)                         | Sentinel `ErrHealthCheckSkipped` in `errors.go`; transient `healthcheck` returns it; `serviceHealthCheck` doc comment fixed (the exact conflation the issue flags); `TestServiceTransient_healthcheck` + the "transient always nil" test updated to `ErrorIs`; new sweep-level `TestScope_HealthCheckWithContextTransient` (proves map presence + sentinel). Base = upstream master 9bae325c; remote branch verified (5 files, right parent). Only new local failures are the repo's pre-existing path-dependent stacktrace tests (identical on base)                                               |
| Prior-PR search (6)                        | No prior transient-healthcheck PRs upstream (only #195 "hooks for healthcheck", unrelated)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| README upstream links (8)                  | #317/#318 referenced in §2.4                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| AGENTS.md sync (22)                        | New "Driver contract (v0.1.1)" + "Upstream engagement" sections: exit-2, empty `--output`, allowlist semantics, workspace `all`, suppression span, custom-gcl registration, voice rules, snippet gate, verification lessons                                                                                                                                                                                                                                                                                                                                                                         |
| TODO_LIST fold (38)                        | "Upstream + hardening round — second pass" section with done list + carried items                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `.config/metadata.yaml` (34)               | Verified: `updated_at` timestamp + `linter` tag churn, already daemon-committed; closed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Hygiene (35/36)                            | `/tmp` artifacts trashed (upstream drafts, three stale binaries, lint reports, sentinel patch); `git worktree prune` clean                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Ecology: samber-do-auditlog (28)           | 13 findings → 0, coverage 25% → 60%. example/: Database + EmailNotifier → context checks; Vehicle/DriverService/PassengerService/MatchingEngine/HTTPServer got real `HealthCheck`s (shutdown-only → HW-1 gone); LeakyService suppressed (demo trait is the failing Shutdown, not health); lazy-with-check registrations suppressed with an honest "resolved before the sweep" reason. live/demo: Database/Cache/UserService/EmailNotifier → context checks; **transient EmailNotifier → named singleton** (the live #317 case; also fixed its latent invoke-name mismatch); delay demo suppressions |
| Ecology: standard-bug-tracking-schema (27) | 4×HW-1 → 0. `TelemetryProvider` + `OTELProvider` got real `HealthCheck(ctx)` implementations (nil provider = failure); boot-critical + doc-example + test-fixture lazy registrations suppressed with reasons; build + vet + targeted tests green; **used the released v0.1.1 binary** to scan                                                                                                                                                                                                                                                                                                       |
| Ecology: CV baseline (29)                  | `--set-baseline` → 5/51 = 10% committed (was 8%), 0 findings                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |

## b) PARTIALLY DONE

- **Upstream engagement:** #317/#318 still OPEN, 0 comments. Fork + branch
  stored **outside** `~/projects` (`~/upstream-work/do-fork`) — the
  scheduled-agents daemon that sweeps `~/projects` swallowed the working
  tree mid-edit and emitted a "chore: auto-commit" on the fork's master;
  the change was recovered from that commit, rebased onto upstream master,
  and the clone relocated to keep PR hygiene. The branch is the current
  upstream master + the sentinel commit only.
- **Filtering the sweep** (`--check` silent under `--json`, item 43) — not
  started; trivial follow-up.

## c) NOT STARTED

- Watching/answering #317/#318 (no activity to answer yet; committing to a
  <1-day response when maintainers move)
- go-health / healthaudit status-model work (items 9-16; design belongs in
  the #318 thread, which needs a maintainer first)
- FluffBall / KeyCountdown MISSING_DIR investigation (30)
- Kernovia go.work go≥1.27 courtesy note (31); ast-state-analyzer `go mod
  tidy` + re-scan (32)
- 43-project ecology re-sweep as a repo script (25)
- HW-4 posture decision (26/Open decisions)
- SARIF coverage property, suppression expiration report, config
  auto-discovery, benchmark on the 63-service corpus, allowlist fuzzing
  (items 44-49)

## d) TOTALLY FUCKED UP

- **The daemon ate the fork working tree.** While preparing #317, the
  scheduled-agents auto-commit daemon (sweeps `~/projects/*`) committed the
  in-progress fork edits as a stray git commit; jj's colocation then
  emptied the working-copy change. Diagnosed via `git cat-file -t
  4c894669` (content = exactly my diff), recovered with `git diff
  9bae325..4c894669 | git apply`, verified byte-identical, then relocated
  the clone out of the swept path. Net damage: zero — but the mechanism
  (background daemon + jj colocation + mid-edit snapshots) is still able to
  surprise.
- **One behavior slip of my own:** while adding sweep suppressions I
  briefly deleted two _unflagged_ config providers (OTELConfigProvider,
  TelemetryConfigProvider) — restored within the same minute after
  noticing the dependency break; build confirmed.

## e) WHAT WE SHOULD IMPROVE

1. **Fork work never lives in `~/projects`.** Scheduled agents sweep it.
   `~/upstream-work/` is the new home; record in the jj skill or memory.
2. **`go build` cache hygiene:** `/mnt/buildcache` hit 100% (139GB go build
   cache) mid-session, breaking loads until `go clean -cache` (33%).
   Add a low-frequency cache sweeping note where build automation is
   documented.
3. **Lint counts must cite the hermetic binary.** The carried "141" figure
   was stale; the true count was 44 and the burn-down is complete — the
   TODO_LIST line claiming "CI lint never green with this config" is now
   history and must not be re-quoted (it defeated the burn-down planning
   for two rounds).

## f) UP TO 50 THINGS TO DO NEXT

1. Watch #317/#318; respond within a day of any maintainer comment
2. Open the prepared PR the moment the maintainer invites (branch
   `transient-healthcheck-sentinel` in `LarsArtmann/do`; open via
   `gh pr create` from `/home/lars/upstream-work/do-fork`)
3. `--check` silent under `--json` (line-noise fix)
4. Compile-check script for `docs/status/*/*.md` snippets too (currently
   only `docs/upstream`)
5. Healthaudit typed `Status` enum + go-health `Check.Status` extension
   (needs #318 to gain a maintainer first)
6. Ecology re-sweep (43 repos) with the released v0.1.1; store as repo
   script; refresh the pseudonymous triage doc with post-fix numbers
7. FluffBall + KeyCountdown MISSING_DIR investigation
8. Kernovia go.work go≥1.27 courtesy note; ast-state-analyzer tidy
9. HW-4 posture (on-by-default info vs opt-in) — your call
10. SARIF coverage as automation metric property; suppression expiry
    report; `samber-linter.yml` auto-discovery; 63-service benchmark;
    allowlist parser fuzz
11. Push the auditor's own `--coverage-min` gates into CI for auditlog +
    standard-bug-tracking-schema (baseline files now exist for CV only)

## g) QUESTIONS (cannot self-answer)

1. **HW-4 posture:** after this round, auditlog + standard-bug-tracking
   both carry hw-4 suppressions for intentional lazy boot providers. Does
   the "lazy = skip until built" reading (which #318 argues should become
   an explicit state) change where you want HW-4 (on-by-default info vs
   opt-in flag)?
2. **Upstream identity:** #317/#318 are filed under your GitHub account,
   and the ready PR is in your fork. Maintainer dialogue ownership — draft
   replies for your review, or you handle directly? (Default if silent: I
   draft, you review within a day.)
3. **Trust the branch:** the pushed `transient-healthcheck-sentinel` branch
   implements option 1 (#317) exactly. If the maintainer picks option 2
   (exclude transients from results), the same test harness flips in
   minutes — worth also preparing the option-2 variant while we wait?
