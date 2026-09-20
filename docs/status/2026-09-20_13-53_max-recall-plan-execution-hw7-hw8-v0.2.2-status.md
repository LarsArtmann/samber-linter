# Status: Max-Recall Plan Execution (T1–T12) + HW-7/HW-8 + v0.2.2

**Created:** 2026-09-20 13:53 CEST
**Session window:** ~12:26–13:53 CEST (same day as the plan it executed)
**Repo:** `samber-linter`, branch `master` at `b3fc0f7`, pushed, clean tree
**Headline:** executed the entire max-recall Pareto plan (T1–T12): HW-7 + HW-8
rules with cross-package fact plumbing, v0.2.2 released and proxy-verified,
machine-pure gates, self-ratchet, drift tests, honest nix versions. All gates
green (`nix flake check`, full test suite, `nix run .#lint` 0 issues, CI green
on master and on the v0.2.2 tag).

---

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                                                                                                                                                      | Evidence                                                                                                                                                                                |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| a1  | **HW-7 `unconditional-nil-check`** implemented: reachable check body is exactly `return nil`; lazy/eager only; promoted methods count; warn/Full (gates by default)                                                                                                                                                                                       | `pkg/healthwash/rules.go` (`hasSoleNilReturnCheck`, `isSoleNilReturn`), driver `ruleMetaByRule`                                                                                         |
| a2  | **Cross-package body facts** — the session's biggest find. Same-package-only body reading missed every declare-here/register-there site (the standard architecture; CV registers in `internal/di`, declares in `internal/*`). Fixed with `NilBodyFact` object facts + a driver-owned **two-sweep** (`factStore`, sweep 1 collects, sweep 2 authoritative) | `internal/driver/driver.go:216-275`, `pkg/healthwash/facts.go`; fixtures `testdata/src/hw7cross/{lib,main}`, driver `TestHW7CrossPackage`                                               |
| a3  | **HW-8 `empty-check-body`** (plan T7): lone naked `return` on a named result; `NakedReturnFact`; disjoint from HW-7 (one return value vs none) — no double report                                                                                                                                                                                         | `pkg/healthwash/rules.go` (`isSoleNakedReturn`), fixture `testdata/src/hw8empty`, discrimination proof row                                                                              |
| a4  | **Coverage-variant audit** (T2): bare/ctx/value-receiver/promoted-embedded/cross-package nil bodies all fire; duck-typed non-do `Check`, naked-return-with-statements, delegation, naked `return` after statements, and HW-5's unreachable body stay clean                                                                                                | scratch modules (this session) + `hw7nil` + `hw7cross` fixtures; README §11 ledger row                                                                                                  |
| a5  | **Release v0.2.2**: CHANGELOG `[0.2.0]`/`[0.2.1]` reconstructed from tag diffs; annotated tag cut and pushed; **`go run …@v0.2.2 -version` prints `v0.2.2` via the module proxy** (the consumer version-gating that was impossible before this morning's provenance fix); CI green on the tag (run 35506911178)                                           | `git tag v0.2.2`; `CHANGELOG.md` `[0.2.2]`                                                                                                                                              |
| a6  | **Gate-matrix tests** (T3): `--check` + impossible `--coverage-min` + corrupt baseline → exit 0 with both gate diagnoses on stderr; per-rule-ratchet-under-coverage-min → exit 1 (existed from the morning fix, kept)                                                                                                                                     | `internal/driver/driver_test.go` `TestCheckForcesZeroThroughFailedGates`, `TestCoverageMinComposesWithBaseline`                                                                         |
| a7  | **Machine-purity fix + strict summary** (T4): coverage/baseline/summary lines are human-only now (`humanPresentation()` guard on every stdout gate print); `--strict` prints an unresolved-count line on human output only; max-recall profile documented; threshold decision recorded (profile-first)                                                    | `internal/driver/driver.go` (`reportStrictUnresolved`, `enforce*`), README quickstart + "Threshold policy", tests `TestStrictUnresolvedSummary`, `TestMachineOutputStaysPureUnderGates` |
| a8  | **Ecology scan rerun** (T6): 72 discovered, 40 analyzed, 32 LOAD_ERROR, 8 with findings, 19 findings, **0 HW-7 in the wild**; stderr persisted; triage recorded in README §11                                                                                                                                                                             | `docs/ecology/2026-09-20-scan.txt` + `-stderr.txt`                                                                                                                                      |
| a9  | **Self-dogfood ratchet** (T9): committed `.samber-linter-baseline.json` (schema v2, 0 registered) enforced by a third CI dogfood leg                                                                                                                                                                                                                      | `.github/workflows/ci.yml`, `.samber-linter-baseline.json`                                                                                                                              |
| a10 | **Drift tests** (T10): README §3 rule sections ↔ analyzer registry (both directions; HW-6 whitelisted as gate mode); README §12 latest-release line ↔ max `git tag` (skips where git is unavailable, e.g. nix sandbox); `ruleMetaByRule` ↔ every exported rule constant                                                                                   | `pkg/healthwash/readme_rules_test.go`, `internal/driver/readme_drift_test.go`                                                                                                           |
| a11 | **nix ldflags version injection** (T11): built binary reports `devel+a269a68` (verified by running `./result/bin/samber-linter -version`); `dirtyShortRev or shortRev` convention matches the source-build form                                                                                                                                           | `flake.nix` `extraBuildAttrs.ldflags`                                                                                                                                                   |
| a12 | **Scope adjudications** (T8): alias-over-eager double-shutdown **rejected** as a rule (shutdown-side, idempotence not statically decidable); DO-1..8 boundary recorded (usage-shape stays in branching-flow's `doanalyzerv2`, DO-9a–e delegate here); value-receiver check = non-candidate (already found via method sets)                                | README §10 "Scope adjudications (2026-09-20)"                                                                                                                                           |
| a13 | **Docs hygiene** (T12): lessons report item 13 annotated done inline (annotate-prose tool, dry-run first); TODO_LIST/ROADMAP harvested (done items deleted, survivors re-cited); `.md` report-format standing recorded; FP-BUDGETS HW-7/HW-8 rows written **before** implementation (budget-before-ship held)                                             | `docs/status/2026-09-16_13-27_*:130`, `TODO_LIST.md`, `ROADMAP.md`, `docs/FP-BUDGETS.md`                                                                                                |
| a14 | **Fixture corpus updated coherently**: incidental nil bodies in single-purpose fixtures converted to delegation (golden `GroqChat`, `hw4lazy`, `hw0orphan`, driver `e2e/hw4/clean` inline modules, sdk `Honest`) so each fixture keeps pinning exactly one rule; `hw2bare`/`hw5value`/`overr` gained HW-7 wants where the body fact is genuinely in scope | `testdata/src/*`, `internal/driver/driver_test.go`, `pkg/sdk/sdk_test.go`                                                                                                               |
| a15 | **Every gate green at the end**: `nix flake check` all checks passed; `go test ./...` all 6 packages ok; `nix run .#lint` 0 issues; CI success on master (`35507875031`) and on tag v0.2.2 (`35506911178`)                                                                                                                                                | CI runs; local gates                                                                                                                                                                    |

## b) PARTIALLY DONE

| #  | Work                                | Done                                                                                       | Missing                                                                                                                                                                   |
| -- | ----------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| b1 | **HW-8 delivery**                   | Implemented, tested, documented, on master                                                 | **Not in any release** — v0.2.2 predates it; consumers don't have HW-8 until the next tag                                                                                 |
| b2 | **T5.7 verification**               | `go run …@v0.2.2` proxy check green; CI green on the tag                                   | The exact `go install module/cmd@v0.2.2` invocation was **blocked by the tool sandbox** ("go install" disallowed) — never executed verbatim                               |
| b3 | **T5.8 CV runbook note**            | Routed to TODO_LIST with the verified facts (`@v0.2.2 -version` → `v0.2.2`)                | The runbook paragraph was never drafted, and nothing was delivered to CV's repo                                                                                           |
| b4 | **README §11 audit** (12.1)         | Two new evidence rows added; composition change does not touch the upstream mechanism pins | No row-by-row re-verification of all 18 existing ledger claims was performed this session — asserted, not audited                                                         |
| b5 | **FP-BUDGETS freshness**            | HW-7/HW-8 rows + boundary cases written pre-implementation                                 | HW-7's ecology cell still says "pending next scan" while README §11 already records the 2026-09-20 scan (0 findings) — internal doc inconsistency; HW-8 genuinely pending |
| b6 | **Ecology scan coverage of HW-8**   | Scan ran with the HW-7 binary (13:03 build)                                                | Scan predates HW-8 — no wild data for HW-8 at all yet                                                                                                                     |
| b7 | **Strict-summary semantics**        | Counts `records` with `Unresolved`                                                         | Counts raw records, not alias-deduped/uniq sites — duplicate registrations of unresolved types inflate the count; no test pins that edge                                  |
| b8 | **`go install` sandbox workaround** | —                                                                                          | Not attempted via a script file or alternate path; accepted the block too quickly                                                                                         |

## c) NOT STARTED

| #   | Work                                                                                             | Where it lives                 |
| --- | ------------------------------------------------------------------------------------------------ | ------------------------------ |
| c1  | HW-8 in a release (tag v0.2.3)                                                                   | `CHANGELOG.md` `[Unreleased]`  |
| c2  | Wire `samber-do-auditlog` + `standard-bug-tracking-schema` ratchet jobs onto v0.2.2              | TODO_LIST                      |
| c3  | CV runbook note draft + delivery                                                                 | TODO_LIST                      |
| c4  | Stale-directive rule candidate renumber (HW-9+) and design                                       | TODO_LIST                      |
| c5  | Default-rules single source (driver + plugin + README drift test)                                | TODO_LIST (lessons item 14)    |
| c6  | Pseudonymous triage doc refresh with 2026-09-20 numbers                                          | TODO_LIST                      |
| c7  | Go toolchain floor bump (go ≥ 1.27.1 via goTarballVersion) to un-LOAD_ERROR 32 ecology consumers | ROADMAP Theme 5                |
| c8  | Upstream engagement: #317/#318 watch (untouched this session)                                    | AGENTS.md upstream section     |
| c9  | Runtime companion `healthaudit` growth (Phase 3)                                                 | ROADMAP Theme 1                |
| c10 | `.crush` history purge decision                                                                  | TODO_LIST open decisions       |
| c11 | Performance benchmark of the two-sweep analyzer on the largest consumer                          | ROADMAP Theme 3 (pre-existing) |
| c12 | Plugin-mode cross-package fact verification under a real custom golangci build                   | new (see e4)                   |

## d) TOTALLY FUCKED UP

Nothing irreversible. Two real mistakes, both caught and corrected in-session, both instructive:

| #  | What happened                                                                                                                                                                                                                                                                                               | Damage                                                                                              | Repair                                                                                                                                  | Root cause                                                                                                                                                            |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| d1 | **CHANGELOG claimed HW-8 inside the already-tagged `[0.2.2]` section** — I edited a frozen release section casually while adding HW-8. A false release-history claim was pushed to origin (commit 7895235) and lived there until the next push                                                              | ~15 minutes of wrong public history; none of it in a tag (the tag's tree never contained the claim) | Caught by my own tag-vs-changelog integrity check; `[Unreleased]` section re-inserted, `[0.2.2]` restored to its true content (b3fc0f7) | Treated a tagged release section like a living doc. Rule now obvious: after tagging, `[vX.Y.Z]` sections are append-only-frozen; new work goes to `[Unreleased]` only |
| d2 | **HW-8's original predicate was physically impossible** — I designed "empty or comment-only body", wrote the FP budget row, and only the fixture compile gate (`missing return`) revealed that Go forbids statement-less error-returning bodies. Budget-before-ship did NOT catch a shape that cannot exist | ~10 minutes of wasted design + one discarded fixture                                                | Predicate re-derived to the compilable no-op: sole naked return on a named result                                                       | The FP budget reasoned about FP risk but never asked "can this shape compile?" — fixture-first would have caught it before the budget prose                           |
| d3 | (minor) **Commit-message quality collapse**: the auto-commit daemon captured most of the session's substantive work under "auto-commit N changed file(s)" messages (e.g. 410fc8c carries HW-8's core). History reads poorly between my real commits                                                         | History archaeology cost                                                                            | Not repaired (daemon commits are house-normal); noted as process debt                                                                   | I batched too much work between my own commits                                                                                                                        |

## e) WHAT WE SHOULD IMPROVE

1. **Tagged-section discipline**: after cutting a release, treat `[vX.Y.Z]` CHANGELOG sections as frozen; a drift test could assert "every section after `[Unreleased]` matches its tag tree" (grep a marker rule from `git show vX:...`).
2. **Compile-feasibility check before budgets**: budget-before-ship should include a 2-minute "does a Go method of this shape compile?" probe with the actual signature before any prose is written.
3. **Commit hygiene under the daemon**: commit my own work with real messages at every task boundary instead of letting the daemon bundle it — the session produced ~8 "auto-commit" messages hiding real changes.
4. **Plugin-mode fact verification**: the golangci custom build runs its own checker; cross-package facts are asserted to work there but never verified. One integration fixture would close it.
5. **Two-sweep cost**: analyzer execution doubled per run with zero measurement. Even a rough before/after timing on a big consumer (CV, 61 registrations) would convert "cheap" from a claim to a number.
6. **Unresolved-count dedup**: decide and pin whether the strict summary counts raw records or uniq sites, and test it.
7. **Ecology scan comparability**: the 2026-09-16 → 2026-09-20 scans are not comparable (discovery grew 45→72, toolchain floor moved). The scan script could record its own go version + discovery count in the header for honest diffing.
8. **Sandbox workarounds**: when a verification command is tool-blocked (`go install`), attempt the scripted-file route once before accepting — the proxy path deserved the exact consumer command.
9. **FP-BUDGETS should be updated in the same breath as evidence lands** (the HW-7 "pending" cell vs README's recorded scan) — one sync pass per scan.

## f) UP TO 50 THINGS TO DO NEXT (Pareto-ordered within tiers)

**Release & consumers (highest value, all bounded):**

1. Tag **v0.2.3** (HW-8 + drift tests + self-ratchet + nix version) — the only way any of today's post-v0.2.2 work reaches consumers.
2. Wire `samber-do-auditlog` healthwash CI job onto `@v0.2.3` (baseline 12/20 = 60% committed).
3. Wire `standard-bug-tracking-schema` ratchet job onto `@v0.2.3` (baseline 2/61 = 3%).
4. Draft the CV runbook paragraph; bump CV's `scripts/healthwash.sh` pin from the blind `@v0.2.1`.
5. Verify `go install …@v0.2.3 -version` with the exact consumer command (outside the tool sandbox if needed).
6. Re-run the ecology scan with the v0.2.3 binary for HW-8 wild data; update FP-BUDGETS rows from "pending" to measured.
7. Refresh the pseudonymous triage doc (2026-09-20-scan supersedes both 2026-09-16 files) with the load-error wave noted.
8. Add a "scan header" to `ecology-scan.sh`: go version, discovery count, linter version — makes future diffs honest (see e7).

**Correctness hardening:**
9. Plugin-mode cross-package fact test under the real custom golangci build (closes e4/b-gap).
10. Pin strict-summary dedup semantics with a duplicate-unresolved-registration test.
11. CHANGELOG-freeze drift test: `[vX.Y.Z]` sections must match their tag trees (closes e1/d1 class).
12. Perf-measure the two-sweep on CV (61 registrations, 3 packages): load+analyze ms before/after; record in ROADMAP Theme 3 item or resolve it.
13. `--json` + `--output <format>` simultaneous use: currently untested — decide allow/reject/last-wins and pin it.
14. `--set-baseline` + `--check` combination test (baseline write under advisory mode — should it write? untested edge).
15. HW-8 on promoted (embedded) methods — covered by shared lookup, but no explicit fixture; one want-line would pin it.
16. HW-7/HW-8 on `Override`-family fixtures — `overr` covers eager; add a lazy-override want for the composition matrix.
17. Suppression interaction: `//samber-linter:allow hw-7` must NOT suppress HW-8 (disjoint rules) — one suppress-fixture row pins it.
18. Assert `hw-unresolved` findings + strict summary agree in count on one fixture (summary vs finding-stream consistency).
19. Baseline schema: consider recording the analyzer version that cut the floor (audit context for "who locked this").

**Rules backlog (each: budget → fixture → proof → docs):**
20. Stale-directive rule (renumber HW-9+): orphaned valid directives with `until` resurface — design already noted in FP-BUDGETS.
21. HW-7/HW-8 v2 widening candidates (only with measured FP data): constant-condition bodies (`if true { return nil }` needs flow), log-then-nil.
22. Alias-coverage honesty: aliases delegate checks to targets — should the HW-6 coverage ratio credit the target's check? (design question, currently "never attributed").
23. Shutdown-quality family (out of scope today by adjudication) — revisit only if a consumer incident demands it.
24. docs/DOMAIN_LANGUAGE.md: add "nil-body check", "naked-return check", "reachable check", "body fact" terms.

**Ecosystem/toolchain:**
25. Go toolchain floor bump to 1.27.x in go-standard (goTarballVersion) — un-LOAD_ERRORs 32 ecology consumers (ROADMAP Theme 5, now with evidence).
26. Re-scan after the toolchain bump; expect the finding counts to jump back toward the 2026-09-16 baseline.
27. Dependabot/golangci pin alignment check: CI action v7 + v2.13.2 still latest-compatible (quarterly posture check).
28. `nix flake check --all-systems` (darwin/aarch64 still unexercised).
29. Binary cache (cachix/attic) for CI speed.
30. Renovate for go.mod + flake.lock.

**Docs debt:**
31. README §11 full claim-by-claim audit (18 rows, one focused pass; close b4).
32. Sync FP-BUDGETS HW-7 cell with the 2026-09-20 scan evidence (close b5).
33. `samber-linter explain HW-N` (rule docs single-sourced from code) — pairs with item 5 (default-rules single source).
34. Config auto-discovery (`samber-linter.yml`).
35. Document the two-sweep architecture in README §4 (currently only AGENTS + code comments).
36. Add HW-8 to `docs/upstream/ISSUE_DRAFT.md` context if/when upstream #318 discussion resumes.
37. Annotate the 2026-09-20_12-09 status report's (f) items now that most are resolved (ANNOTATE mode, inline).
38. Annotate the max-recall plan (61-step table) with done-at markers — it is now a historical snapshot whose items are ~95% executed.

**Upstream & adoption:**
39. Check samber/do#317/#318 for maintainer responses (last checked 2026-09-10).
40. Decide upstream-watch ownership + SLA (user decision, TODO_LIST).
41. GitHub Release object for v0.2.2/v0.2.3 (releases are tag-only today; lessons item 16's runbook covers the rest).
42. Public website: still deferred until adoption exists — revisit criteria remain undefined; define them or delete the item.

**Quality-of-life:**
43. `--output` markdown dogfood already exists; add one assertion that the HW-7/HW-8 rows render (format coverage for the new rules).
44. SARIF property round-trip test for HW-7/HW-8 (rule + confidence in the property bag).
45. Consider `--include-suppressed` (pre-existing FEATURES "worth considering").
46. Corpus growth: generic instantiated service types (`Svc[T]` registered as `Svc[int]`) — does LookupFieldOrMethod origin-object match hold? One fixture would answer.
47. Duplicate-registration × HW-7: two sites, same nil-body type — both fire? (edges-style fixture; expected yes per house rule).
48. `check-rows.py` completeness gate over annotated historical docs (docs-health tooling, one-shot).
49. ExploreCI: cache `~/go/pkg/mod` between jobs to speed the 5 CI jobs (~45s each today; nice-to-have only).
50. ROADMAP Theme 1 (runtime outcome triangle) needs a bounded first step or it stays forever-raw — draft the healthaudit `Status` enum sketch.

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Tag v0.2.3 now?** HW-8, the drift tests, the self-ratchet, and the nix version injection are all committed on master but invisible to consumers (v0.2.2 predates them). I recommend tagging v0.2.3 after items 9–11 (plugin fact test + dedup pin + changelog-freeze test) so the release carries the hardening too. Approve, or tag immediately?

2. **Toolchain floor:** 32 of 72 ecology consumers now require go ≥ 1.27.1; our floor is 1.26.7 (nixpkgs `go_1_26`, go.mod `go 1.26.7`). Bumping the floor means touching go-nix-helpers' `goTarballVersion` (per AGENTS). Do you want the repo floor to chase the consumers (better ecology coverage, more bump churn), or stay pinned and accept the scan blind spot?

3. **How should max-recall FP data be gathered?** The recorded threshold policy flips the `--min-confidence` default "after one release of measured FP data" — but nothing collects that data today (no telemetry by design). Is manual FP reporting from your own runs enough, should the max-recall profile get a `--report-fp` note-file convention, or should the default flip be judged from spot-checks on the next ecology scan instead?

---

**Self-check before filing:** no ```go blocks in this file (snippet gate applies to `docs/status/**`); every claim above traces to a file, commit, CI run id, or is explicitly marked as an unverified gap; the two mistakes in (d) are reported as shipped-and-fixed with hashes.
