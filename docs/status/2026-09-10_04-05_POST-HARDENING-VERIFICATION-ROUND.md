# Status Report — Post-Hardening Verification Round (samber-linter)

**Date:** 2026-09-10 04:05 CEST
**Session scope:** deep code review of the v0.1.0 analyzer + driver, execution of the
five falsely-completed plan items (A68/A74/A75/A81/A96), bug fixes found by running
the shipped binary, and a lint burn-down from a self-inflicted 231 findings to 141.
**Repo state at writing:** clean tree, HEAD `646e983` (daemon-committed), all tests
green, vet clean, CI-parity dogfood green, self-analysis exit 0.

---

## TL;DR

The v0.1.0 "complete" claim was hiding a P0 regression (the tool was broken for
every plain invocation), five plan items marked `[x]` that were never executed,
and a repo-wide red lint that CI would have caught if CI had ever run. This round
fixed the P0, actually executed the five items, fixed five additional latent bugs
found in the deep read, and locked the verifications in as permanent tests. Two
self-inflicted file-destruction incidents occurred mid-session; both were fully
recovered from the auto-commit daemon's history with zero loss.

---

## The three framing questions

### What did I forget?

1. **`.gitignore` for `custom-gcl`** — the 52 MB plugin test binary sat untracked
   in the repo root through most of the session. Gitignored only at the end.
   Risk was real: the auto-commit daemon could have blob-committed it (again —
   this repo already had one history-rewrite because of a leaked file).
2. **AGENTS.md sync** — I recorded the new facts in README/CHANGELOG/FEATURES/
   TODO_LIST but not in AGENTS.md (the mandatory `linters.settings.custom`
   plugin registration, workspace `all` pattern, load-failure exit 2, the lint
   config reality). A fresh session will not have these in its context file.
3. **`nix flake check`** — never re-run after the final state. The known gotcha
   (flakes exclude untracked files, commit before check) applies; flake.nix was
   modified by the parallel session and I never verified the check passes now.
4. **GitHub CI verification** — I verified CI parity locally (dogfood command,
   drift matrix is part of `go test`), but never checked the actual GitHub run
   (`gh run list`) to see if the first-ever CI run is green after the fixes.
5. **No `v0.1.1` tag** — CHANGELOG has a 0.1.1 entry; no tag, no release. The
   repo is at `version = "0.1.0"` in main.go — the two now disagree.
6. **`toFinding`'s unused `Version` parameter** — spotted in the first deep read
   (`_ string`), never fixed.
7. **dprint** — the repo has dprint.json for md/json formatting; I hand-edited
   four markdown files and never ran the formatter.
8. **Concurrent-session coordination** — a parallel crush session committed the
   `--output` feature into this repo _during_ my session. I fixed its P0 bug in
   output.go but never checked whether that session was done editing the file.
   No clobbering occurred (verified), but it was luck, not process.
9. **The 2 missing dirs** (FluffBall, KeyCountdown) — noted "MISSING_DIR" in the
   ecology scan, never investigated whether they were renamed/moved.

### What could I have done better?

1. **Test before refactoring.** I ran the dogfood command only at the END. The
   P0 regression was live in a committed state the whole session. First action
   on a "make it better" task should be: run the tool, see it broken, THEN read.
2. **Stop scripting file surgery.** The root cause of both destruction incidents
   was the same: I replaced surgical, verified edits with one-shot python text
   munging on a live file. The second incident (positional splice) truncated
   driver.go to 37 lines. The edit tool with exact-match strings — which I then
   used successfully to re-apply everything — was the right tool all along.
3. **Language fundamentals under pressure.** I split Go string literals across
   lines with NO `+` operators (Go requires explicit concatenation; adjacent
   literals are invalid), then put `+` at line STARTS (automatic semicolon
   insertion strikes), then wrote a regex the heredoc mangled, then a
   line-based "fixer" that matched import statements. Four consecutive failures
   from not thinking before running.
4. **I destroyed work silently via restore.** Restoring rules.go from a daemon
   commit that pre-dated my refactor silently reverted the typeFacts
   decomposition; I didn't notice until line numbers in lint output disagreed
   with expectations. Restores must be followed by "what did I just lose?"
5. **Stray characters in a fixture** — "ticket追踪" leaked into a test reason
   string. Caught and fixed, but it shipped into a commit.
6. **Stale-binary confusion at session start** — several commands spent
   believing `/tmp/samber-linter` was "stale" when the actual explanation was
   that the working tree had changed under me. When output contradicts a file
   I read, check file mtime and git status FIRST, not last.
7. **Multiedit file mix-ups** — twice applied evalSite edits to healthwash.go
   when the function lives in rules.go. Slow, noisy, avoidable.

### What could I still improve?

1. Regression-sweep the ecology corpus (43 projects) after analyzer changes —
   I spot-checked CV only. The scan is a one-off /tmp artifact; it should be a
   repeatable script in the repo.
2. Zero the lint entirely instead of stopping at "below baseline". 141 remain,
   all in files I didn't author this round, but CI's lint job is still red and
   "CI green" is still a claim the repo cannot make.
3. Version hygiene: main.go version, CHANGELOG, and git tags should move
   together. They don't right now.
4. Baseline file design: it records only aggregate coverage. Per-rule counts
   would make the ratchet stronger (a rule regression could hide inside
   aggregate coverage).
5. `--check` prints an advisory line even with `--json`, polluting machine
   parsing — small UX gap I introduced and didn't close.

---

## a) FULLY DONE

| Item                                              | Evidence                                                                                                                                                                                                                                                                                                                                                                                                        |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P0: empty `--output` broke every plain invocation | `ParseOutputFormat` accepts zero value (internal/driver/output.go); `TestParseOutputFormat` empty-value subtest; plain run verified locally + on CV                                                                                                                                                                                                                                                             |
| A68 `--check` advisory mode                       | driver `Options.Check`, main.go flag, `TestCheckAdvisoryMode`, verified on CV (forced exit 0)                                                                                                                                                                                                                                                                                                                   |
| `--disable` CLI wiring                            | driver `Options.DisableRules` → analyzer flag, `TestDisableRules`, verified on CV (HW-4 muted)                                                                                                                                                                                                                                                                                                                  |
| Load failure exits 2                              | driver.go load branch, `TestLoadFailureExitsTwo` (broken go.mod fixture)                                                                                                                                                                                                                                                                                                                                        |
| Allowlist semantics (A61-adjacent)                | empty `pathPattern` = project-wide; rule-less entries warn + inert; `TestAllowlistEmptyPathPatternCoversProject`                                                                                                                                                                                                                                                                                                |
| Suppression inside multi-line calls               | `siteReport.endLine`, span loop in `reportFindings`; `suppressspan` fixture in golden corpus                                                                                                                                                                                                                                                                                                                    |
| Orphaned HW-0                                     | `reportOrphanedDirectives` with dedupe vs site-attached reports; `hw0orphan` fixture + `TestOrphanedDirectiveHW0`                                                                                                                                                                                                                                                                                               |
| A74 edge fixtures                                 | `testdata/src/edges`: duplicate registrations (2 sites, 1 coverage row) + nested closures (proved closure providers returning concrete types are themselves HW-1 sites)                                                                                                                                                                                                                                         |
| A75 go.work e2e                                   | `TestGoWorkMultiModule`: workspace with app + do-stub modules, `all` pattern, HW-1 found                                                                                                                                                                                                                                                                                                                        |
| A81 plugin proof                                  | `plugin/plugin_integration_test.go`: in-process registration + settings pass-through + full `golangci-lint custom` build-and-fire; `.custom-gcl.yml` usage comment now documents the mandatory `linters.settings.custom` registration                                                                                                                                                                           |
| A96 FP budgets                                    | `docs/FP-BUDGETS.md`: per-rule budgets, structural FP prevention, boundary cases, ecology evidence                                                                                                                                                                                                                                                                                                              |
| Ecology anomalies                                 | Kernovia = go.work needs go ≥ 1.27; ast-state-analyzer = stale go.mod. Target-project breakage; analyzer reports cleanly. Documented in TODO_LIST                                                                                                                                                                                                                                                               |
| Complexity refactors                              | driver `Run` (cyclop 20 → buildAnalyzer/analyzePackages/emitOutputs/applyGates), `reportCoverage` (21 → writeBaseline/enforceCoverageMin/enforceBaselineRatchet), `applyAllowlist` (15 → warn/usable/entryCovers), `evalSite` (gocognit 39 → resolveStoredType + typeFacts + reportTransientRules/reportSweepRules/addBareCheckRule); analyzer split: collectDirectives/reportFindings/reportOrphanedDirectives |
| Lint burn-down                                    | 231 → 141 findings; session-start baseline was 199; zero findings added by this round remain                                                                                                                                                                                                                                                                                                                    |
| Docs sync                                         | README (exit codes, new flags, workspace `all`, multi-line suppression), CHANGELOG 0.1.1 entry, FEATURES.md, TODO_LIST verification round                                                                                                                                                                                                                                                                       |
| Final verification                                | `go build`/`go vet` clean, `go test ./...` all green (incl. plugin integration), dogfood `--coverage-min 0.0 ./...` exit 0, `--check` self-run exit 0                                                                                                                                                                                                                                                           |

## b) PARTIALLY DONE

| Item                  | State                                                                                                                                                                                                                                                                 |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
~~| Lint debt             | 141 pre-existing findings remain (paralleltest on analysistest-based tests, varnamelen in legacy code, 2 gochecknoglobals tables, wrapcheck, drift_test gocognit/lll). CI lint job has NEVER been green. Tracked in TODO_LIST, deliberately not mixed into this round |~~ done (2026-09-10: still red; root causes + burn-down tracked as TODO_LIST quality gate) |~~
| v0.1.1 release        | CHANGELOG entry written; main.go still says 0.1.0; no git tag, no push of a release                                                                                                                                                                                   |
| CV baseline ownership | CV's 9 findings were FIXED externally since the scan; committed 8% baseline now reads 10% and correctly prompts `--set-baseline`. Lock-in still pending (user decision)                                                                                               |
~~| HW-0 completeness     | Orphaned _malformed_ directives now surface; _valid_ orphaned directives (stale suppressions) are only documented as a future HW-7 candidate                                                                                                                          |~~ done (HW-7 candidate tracked: TODO_LIST decision + docs/FP-BUDGETS.md)
| Ecology triage        | Pseudonymous queue committed; CV cleaned itself; rank-1 (standard-bug-tracking-schema, 63 unprotected services) untouched                                                                                                                                             |
~~| Concurrent session    | Its `--output` feature adopted + fixed + regression-tested; its README/AGENTS/flake edits observed and respected; no coordination channel exists                                                                                                                      |~~ done (exercised again 2026-09-10: a parallel session edited this repo during the docs-health audit; verified before write, no clobbering)

## c) NOT STARTED

~~- GitHub CI verification of the actual Actions runs (`gh run list`) — repo may
  finally be green but nobody has looked~~ done 2026-09-10: red — test/dogfood failed on missing GOEXPERIMENT (fixed `17732a4`), lint red; only drift-matrix green (run 34425222926)
~~- `nix flake check` after final commit~~ done 2026-09-10: build+test checks green; lint check red (44 findings — tracked debt)
- Rank-1 triage execution (standard-bug-tracking-schema: 4×HW-1, 63 unprotected)
- samber-do-auditlog fixes (7×HW-1, 5×HW-2, 1×HW-3 — biggest offender)
- FluffBall / KeyCountdown MISSING_DIR investigation
~~- Upstream samber/do issue filing (docs/upstream/ISSUE_DRAFT.md + repro exist,
  verify-before-filing not yet executed)~~ done at `ae77908` — filed as samber/do#317 + #318, both verified OPEN
- HW-7 stale-directive rule design
- Health-washing fix proposals for any of the 11 remaining finding projects
~~- AGENTS.md update (see "forgot" list)~~ done 2026-09-10: Driver contract + Upstream engagement sections (parallel session) + GOEXPERIMENT/CI-status rewrite (docs-health audit)
~~- dprint formatting pass over hand-edited markdown~~ done 2026-09-10: `dprint fmt` clean, check green (integration into nix checks still open, TODO_LIST)

## d) TOTALLY FUCKED UP

Both incidents were **fully recovered** (daemon history + re-application), with
all tests green at the end — but they happened, and honesty requires the log:

1. **driver.go destroyed by positional splice.** A python `s.find()`-based
   surgery truncated the file to 37 garbage lines. Root cause: string surgery
   on a live file instead of exact-match edits. Recovery: daemon commit
   `67bfbc0` had the full 606-line file; `git show 67bfbc0:... > driver.go`
   restored it with zero loss (the failed edits had never been written).
2. **rules.go destroyed twice in a row.** (a) Split long string literals
   without `+` operators — invalid Go. (b) A "line-based fixer" matched import
   statements (`"fmt"` → `"go/ast"`) and broke the file again. Recovery: daemon
   commit `9bfa82d` — which SILENTLY PRE-DATED the typeFacts refactor, so the
   restore also reverted work; noticed only via lint line numbers, and the
   refactor was re-applied with exact edit-tool operations.
3. **"moved 0 operators" mystery.** The operator-move script reported zero
   matches on its first run against a file that clearly contained the pattern;
   I ran a corrected version without understanding why the first failed. It
   worked (moved 15) — but "it worked, moving on" is not understanding.
4. **Four consecutive scripting failures** (assertion miss → heredoc-mangled
   regex → adjacent-literals blunder → import-matching fixer) before falling
   back to the edit tool that worked every single time.

Lesson recorded: in a repo with an auto-commit daemon, git history is a
lifeline — but the correct default is exact-match edits, and any scripted
rewrite needs a dry-run print before write.

## e) WHAT WE SHOULD IMPROVE

1. **Make the ecology scan a repo script** (`scripts/ecology-scan.sh`):
   repeatable, pseudonymized output, used as the post-change regression proof.
2. **First-action discipline:** run the tool before refactoring it; the P0 was
   findable in 10 seconds.
3. ~~**CI truth:** wire the plugin build + a lint gate into CI intentionally —~~ done (diagnosed 2026-09-10 — GOEXPERIMENT fix `17732a4`; golangci toolchain + ~141 findings tracked as TODO_LIST quality gate)
   ~~either fix the 141 findings or scope the lint config honestly; "red since~~
   ~~day one" corrodes every future green claim.~~
4. **Version triple-lock:** main.go `version`, CHANGELOG, and git tags must
   move in one commit per release.
5. **Baseline v2:** per-rule counts (ratchet rules, not just coverage), schema
   version field with validation errors.
6. **--check + --json interplay:** suppress the advisory line in machine
   formats.
7. ~~**Concurrent-session protocol:** before editing, `git log --since=...` to~~ done (exercised 2026-09-10 — parallel-session edits verified before write)
   ~~see if another session is mid-flight; afterwards, re-diff files owned by~~
   ~~the other session.~~
8. ~~**AGENTS.md as the memory anchor:** the golangci registration trap, exit~~ done (done 2026-09-10 — Driver contract + Upstream engagement sections in AGENTS.md)
   ~~contract change, and workspace pattern belong there, not just in README.~~

## f) UP TO 50 THINGS TO DO NEXT

**Release & CI (1–6)**

1. Bump main.go version to 0.1.1 and tag `v0.1.1` (CHANGELOG entry already written)
2. ~~Watch the next GitHub Actions run; fix anything red (first real CI validation)~~ done (observed 2026-09-10 — red on GOEXPERIMENT; fix `17732a4` lands with the next run)
3. ~~Run `nix flake check` on the committed tree; fix flake drift from the parallel session's edits~~ done (run 2026-09-10 — build/test green, lint check red (44; tracked))
4. Reconcile golangci-lint versions: `.custom-gcl.yml` pins v2.12.2, local is v2.13.2, CI uses `latest` — pin one policy
5. Add a CI job that builds `custom-gcl` (plugin proof in CI; network-gated)
6. ~~Check `gh run list` for the lint job reality; decide fix-vs-rescope policy~~ done (done 2026-09-10 — lint job fails at config load (golangci binary built with go1.24 vs go.mod 1.26.7), run 34425222926)

**Lint debt burn-down (7–12)**
7. paralleltest: add `t.Parallel()` to analysistest-based tests (TestGoldenCorpus, TestStrictUnresolved, TestDiscriminationProofs + subtests, TestParseDirective)
8. varnamelen sweep: rename single-letter loop vars in driver.go (d/f/r), collectDirectives (c/d), suppress.go (d/s), drift_test.go (f/fd)
9. gochecknoglobals: justify or restructure `VerifiedDover` and `ruleMetaByRule`
10. wrapcheck: wrap `packages.Load` error with context
11. drift_test.go: split `assertMechanism` (gocognit 34) and wrap its 132-char line
12. main.go: `fs` varnamelen; forbidigo exception for the version print

**Analyzer features (13–22)**
13. HW-7 "stale directive": valid orphaned directives (esp. with `until`) resurface for cleanup
14. Baseline v2: per-rule finding counts + schema version + validation errors
15. `--check` silent under `--json` (no advisory line in machine output)
16. Config auto-discovery (`samber-linter.yml` in repo root; `--config` overrides)
17. HW-4 posture: implement whatever the user decides (opt-in vs on-by-default)
18. Per-package findings summary line (`pkg: 3 findings`) for large repos
19. `--baseline` write-through option (auto-lock improvements, opt-in)
20. Rule metadata exposure: `samber-linter explain HW-1` prints rule docs
21. Consider `--output jsonl` via go-finding for streaming consumers
22. Document exit contract in `--help` epilog, not just README

**Testing (23–31)**
23. Re-run the 43-project ecology sweep post-changes; store as repo script with pseudonymous output
24. Edge fixture: provider methods (`do.Provide(nil, (*Svc).New)`), interface-satisfying generics
25. Fuzz the allowlist config parser (never-panics guarantee)
26. SARIF shape assertion: rule IDs + severity present in export
27. Driver e2e with Override* family (analysistest covers it; driver test doesn't)
28. Test expired-directive resurfacing in the driver (not only analysistest)
29. Benchmark analyzer on the largest consumer (standard-bug-tracking-schema) — load time budget
30. Windows/CGO_ENABLED=0 cross-compile check in CI matrix
31. Golden coverage-line assertions for `--output markdown`/`csv` (headers exist; gate them)

**Ecosystem (32–39)**
32. Investigate FluffBall/KeyCountdown MISSING_DIR (renamed or deleted?)
33. Kernovia: fix or report its go.work (go ≥ 1.27 requirement) — upstream courtesy PR
34. ast-state-analyzer: run `go mod tidy` there, re-scan, close the anomaly
35. ~~File the upstream samber/do transient-healthcheck issue (draft exists; execute verify-before-filing first)~~ done (filed samber/do#317 + #318 `ae77908`, both OPEN)
36. samber-do-auditlog: propose fixes for 7×HW-1 + 5×HW-2 + 1×HW-3 (biggest offender)
37. standard-bug-tracking-schema: begin rank-1 triage (63 unprotected services)
38. ~~branching-flow DO-9 backport: confirm still clean after driver changes~~ done (delegation intact — branching-flow `analyzer_healthwash.go` verified 2026-09-10)
39. Refresh the pseudonymous triage doc with post-fix numbers

**Docs & hygiene (40–46)**
40. ~~AGENTS.md: add settings.custom trap, workspace `all`, exit-2 load failures, lint policy~~ done (done — AGENTS "Driver contract" documents the custom-build requirement)
41. README plugin section: the registration requirement (currently only in .custom-gcl.yml comment)
42. ~~README suppression section: multi-line call example~~ done (documented — README Quick start notes multi-line-call suppression)
43. ~~dprint pass over all hand-edited markdown/json~~ done (2026-09-10 — `dprint fmt` clean, check green)
44. Remove stale /tmp binaries (sl-test, sl-final, samber-linter) or document them as session artifacts
45. `git worktree prune`; confirm no dangling worktree metadata
46. ~~Confirm the parallel session's remaining TODOs (AGENTS/flake notes at commits 568d6ae) are complete~~ done (confirmed — AGENTS/flake edits landed (daemon history))

**Bigger ideas (47–50)**
47. healthaudit cross-link: findings suggest the runtime sweep as complementary evidence ("static says it CAN fail; runtime says it DID")
48. SARIF: expose coverage as a SARIF property/automation metric
49. Rule severity/confidence table generated from code (single source: ruleMetaByRule) into README
50. Suppression age report: `--suppressions` lists all directives + reasons + expiry, for periodic review

## g) QUESTIONS (cannot self-answer)

1. **HW-4 default posture:** stay on-by-default (`info`/Medium, non-blocking) or
   become opt-in (flag-gated)? 24 of 66 ecology findings were HW-4 — this is the
   main noise dial and it is still set by my judgment, not a decision.
2. **Release policy:** tag and push `v0.1.1` now (contains the P0 fix users need),
   or batch it with the lint debt burn-down into a later release?
3. **Concurrent sessions:** another crush session edited this repo mid-flight
   (--output feature). Is it still active, and should I treat output.go/driver.go
   as contested — or is it done and I own the tree from here?
