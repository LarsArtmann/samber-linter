# Status Report — Cross-Repo Ratchet Round: Analyze-API Split Brain, CV Re-Baseline, Survey Ownership Fix

**When:** 2026-09-16 18:44 CEST · **Repo:** `/home/lars/projects/samber-linter` (HEAD `caf09c4` + daemon flow) · **Session scope:** execute the post-quality-gate TODO queue (CV re-baseline, sibling baselines + CI wiring, triage refresh, snippet gate) plus the queued Analyze-API contract review.

**Format note:** this report is Markdown per explicit user instruction; the status-report skill's canonical HTML format is overridden for this report only.

**Context at start:** previous round ended with all gates green (`nix flake check` "all checks passed"), a delivered status report, and three open questions (push, CV, survey scope). This round executed everything reachable without push authorization.

---

## a) FULLY DONE (executed + verified this session)

| # | Item | Evidence / notes |
|---|------|------------------|
| a.1 | Repo-state rebuild before touching anything | HEAD had moved to `4f4e762` (2 daemon commits beyond the last-seen blob, incl. `d032163` — 86 lines across the 3 driver files, late formatter rewrites). One uncommitted line in the prior status report (a.11: "lint green" → "flake check all passed") was judged accurate against the verified final state and kept, not reverted |
| a.2 | **Analyze-API split brain eliminated.** The parallel session's `driver.Analyze`/`pkg/sdk` shipped its own `packages.Config` copy ("must stay in sync" by comment) and a private GOFLAGS sanitizer — the CLI path had neither. Consolidated into one `load(ctx, opts)` in `internal/driver/driver.go:301`: cancellable ctx (x/tools nil-ctx tolerance verified in source, `packages.go:751`), inherited `GOFLAGS` stripped of `-mod` tokens for **both** entry points (a global `-mod=vendor` previously poisoned every CLI run; explicit `opts.Env` still wins, last occurrence). `loadWithCtx`, `sanitizedGoFlagsEnv`, `stripModTokens` (sdk) deleted; table test moved to `internal/driver/stripmod_internal_test.go` | New tests: `TestLoadSanitizesInheritedGoFlags`, `TestLoadExplicitEnvOverridesHostileInherited`, `TestLoadEnvSanitizedEntryAlwaysPresent` (`internal/driver/load_test.go`) — all PASS; `TestStripModTokens` moved; `go vet` clean; driver+sdk suites green; committed at HEAD (`loadEnv` ×4 verified via `git show HEAD:`) |
| a.3 | **Snippet gate extended to `docs/status/**`** (`scripts/check-upstream-snippets.sh`): globstar over both trees, collision-proof work names (`docNN` — same basenames exist across dated status subdirs). Same contract: runnable repro or `snippet-skip`; unclassified fails | Positive path PASS (upstream ISSUE_DRAFT snippet); negative proven both ways: unclassified → exit 1 with file:line, `snippet-skip` → SKIP exit 0; probe trashed. Zero existing Go blocks in docs/status → zero migration cost, pure future-proofing |
| a.4 | **CV re-baselined to schema v2** at the honest scope: nine workspace modules enumerated (`./...` + 8 siblings), 10/30 = 33% coverage, empty findings map | Gate re-run exit 0; advisory line shows `health-coverage: 10/30 = 33% (baseline: 33%)`; committed at CV HEAD (verified via `git show HEAD:.samber-linter-baseline.json` → `"version": 2`). Numbers match the old v1 exactly → apples-to-apples confirmed |
| a.5 | **Phantom CV HW-2 root-caused.** The finding lives in `cmdguard` v4 (the user's library), not CV: `Package()` self-registers `*CLI[T]` via `do.ProvideValue` (`pkg/cmdguard/v4/scope.go:427`) and `*CLI[T]` declares a bare `HealthCheck()` (`cli_accessors.go:42`) while a ctx variant (`HealthCheckWithContext`) exists next to it. CV never calls `Package`; all nine CV modules scan clean individually | Ownership decision recorded in TODO_LIST as a cmdguard-repo item; CV baseline stays clean of it |
| a.6 | **Sibling baselines written + committed:** samber-do-auditlog v2 **12/20 = 60%** (gate exit 0), standard-bug-tracking-schema v2 **2/61 = 3%** (gate exit 0) | Both files verified at their HEADs; both repos' AGENTS.md gained a "Health-wash ratchet" section (baseline numbers, regeneration command, and the exact pending CI step) |
| a.7 | **Ecology survey de-fucked (go.work over-scan fixed).** Pattern `all` expands to the full dependency closure → dependency packages' own registrations were attributed to the consumer (the CV phantom). The survey now parses `go.work` via `go work edit -json` and scans each member with `./...`, sums coverage across modules, preserves LOAD_ERROR/TIMEOUT semantics, and hardens a broken go.work to LOAD_ERROR (never silent-CLEAN) | Fixture-proofed before the real run: clean / HW-1 / HW-2 / broken / two-module go.work aggregation — all five behave exactly as specified. Script fix + disk evidence below |
| a.8 | **Corrected survey run complete** (`/tmp/ecology-2026-09-16-v2.txt` → `docs/ecology/2026-09-16-scan-per-module-fixed.txt`): **66 analyzed · 50 clean · 16 with findings · 56 findings · 4 load errors** (pre-fix: 65/47/18/59/5) | CV (`p-a32b`) moved from rank 40 (0/0/0 + phantom HW-2=1) to **rank 3, 30 registered / 10 checked / 0 findings** — the row is now CV's own code only. samber-do-auditlog (`p-d9a9`) regression proof holds: rank 15, 20/12, 0 findings. Both full ranked tables (ranks 1–65+, nothing piped through `tail` this time) persisted in-repo: pre-fix as `2026-09-16-scan.txt` (kept as evidence of the over-scan), post-fix as `2026-09-16-scan-per-module-fixed.txt` |
| a.9 | Docs truth pass: CHANGELOG (+2 Fixed: CLI GOFLAGS poisoning, go.work over-scan; +1 Added: snippet gate over status docs), AGENTS.md (per-module go.work doctrine + the `DiskPath`-not-`DiskDir` trap + scan fixture traps + persistent-output lesson), TODO_LIST rewritten (release-gated wiring, cmdguard decision, push authorization as explicit open decisions) | treefmt across the tree: 0 changed — everything already format-clean |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
|---|------|------|---------|
| b.1 | Triage-doc refresh (`docs/status/2026-09-10_02-40_ECOLOGY-TRIAGE-PSEUDONYMOUS.md`) | Corrected data collected and persisted in-repo; regression trio verified under the fixed binary | The doc itself is not yet rewritten; the 2026-09-10 table is now doubly stale (numbers AND methodology) |
| b.2 | Sibling CI wiring | Baselines committed, AGENTS notes in place with the exact job recipe | The CI job YAML cannot be added yet: the only released samber-linter (v0.2.1) predates baseline schema v2 and **fails closed** on the new files — a v0.2.1 pin would guarantee red CI. Gated on the next release, which is gated on push |
| b.3 | Final gates for this round's code | `go build ./...`, driver+sdk tests, `go vet`, treefmt: all green; the survey (which builds the binary) exit 0 | Full `nix flake check` (hermetic lint + dprint + nix checks) not yet run on this round's tree — queued as immediate next step |
| b.4 | CV enforcement of its own baseline | Baseline exists and is committed | Nothing in CV's CI/flake runs the linter — the ratchet is currently survey-evidence only. Wiring = cross-repo change awaiting scope confirmation (question 3) |

## c) NOT STARTED (unchanged, by design — user decisions or explicitly queued)

1. Push of `master` (hard rule: never push without explicit go-ahead) — now ~7 unreleased commits incl. schema v2.
2. Tag/release v0.2.2 (blocked by 1; blocks b.2).
3. cmdguard HW-2 fix in the cmdguard repo (needs the owner's API decision, question 2).
4. HW-4 default posture (on-by-default info/Medium vs opt-in) — 24-of-66 noise dial decision.
5. `.crush` GitHub history purge (support ticket vs delete+recreate).
6. HW-7 "stale directive" rule candidate.
7. Upstream ownership/sla for samber/do#317 + #318 (both still OPEN, no response).
8. P3 items: doanalyzerv2 DO-9 backport deepening, runtime companion (`healthaudit`) beyond current metrics.
9. README documentation of the new programmatic API (`pkg/sdk`) — not verified whether the parallel session documented it; flagged as a suspected gap, not researched this session.

## d) TOTALLY FUCKED UP (honest failure log — all caught before shipping, none reached the real survey undetected)

1. **Wrong JSON field from `go work edit -json`.** I piped `.Use[].DiskDir` without checking the raw schema; the real field is `DiskPath` (and it is *relative*, needing resolution against the project dir). Result: silent `null` module paths → the workspace fixture row came out 0/0 CLEAN instead of 2/2+HW-2. This is exactly the "independently verify tool output before mutating" lesson — applied to *code* but skipped for *tool output schema*. Caught by the fixture smoke test, fixed, then verified.
2. **Five fixture-design bugs in a row** before the smoke test went green: (i) a `go.work` placed above sibling fixtures poisoned them via workspace discovery ("directory prefix . does not contain modules listed in go.work") — 5 false LOAD_ERRORs; (ii) the go.work project lacked a root `go.mod`, so discovery never saw it as one candidate; (iii) relative `replace ../do-stub` broke the moment I moved fixture dirs into a subtree; (iv) fixture `main.go` used `context.Context` without importing `context`; (v) my "clean" fixture was actually a bare-`HealthCheck` module (a genuine HW-2) — a mislabeled fixture that could have convinced me the *rule* was wrong instead of my fixture. Lesson: fixture scaffolding needs its own checklist (isolated tree, root go.mod, path audit after any move, compile the fixtures, name fixtures by what they prove).
3. **Known-bad artifact persisted deliberately:** `docs/ecology/2026-09-16-scan.txt` is the pre-fix run whose go.work rows are wrong by construction (CV rank 40 with a phantom). It is kept as paired evidence with the fixed run, but anyone citing the wrong file later gets the wrong numbers. Mitigation exists (AGENTS + TODO point at the fixed file) but the risk is real until the triage doc (b.1) names the canonical table.
4. **One external-state flip left unexplained:** Standup-Killer was LOAD_ERROR (dep mismatch) in the previous round and analyzed clean this round without anyone touching it from here. Probably a parallel session tidied its deps — but "probably" is not verification, and a survey whose load-error set silently changes between runs undermines its use as a regression gate.
5. Minor: shellcheck is not installed, so the edited survey script got `bash -n` (syntax only) — weaker than the check the repo deserves; my module-isolation loop had a stray duplicate scan command before the `cd` (harmless, but sloppy); one cycle spent suspecting the parallel session's status-doc edit of damage before verifying it was an accuracy fix.

## e) WHAT WE SHOULD IMPROVE

1. **Verify tool-output schemas before wiring them** — print the raw JSON/-help output first; `null`s from jq are silent failures. Generalize the existing "verify tool output" memory rule to cover *schema assumptions*, not just staleness.
2. **Fixture design checklist** (isolated tree, no go.work above, root go.mod, relative-path audit, compile fixtures, prove-named fixtures) — encoded into AGENTS.md already; apply it to every future script fixture.
3. **Add shellcheck to the flake checks** — the repo hermetically gates go/nix/json/yaml/md formatting but nothing lints `scripts/*.sh`, and this session shipped bash twice.
4. **Survey diff mode:** consecutive surveys should print the delta (new/vanished rows, load-error flips, coverage changes per pseudonym) instead of making the reader diff two 70-line tables by eye; also log go.mod/go.sum hashes so "the project changed underneath us" becomes provable.
5. **Parallelize the survey** (~10 min serial today, bounded `xargs -P`) and write per-project durations to spot timeout-budget pressure.
6. **Baseline should pin its scan scope** (schema v3 candidate): store the patterns (or a workspace-module list) used at `--set-baseline` time and fail loudly when a check runs with different ones — the apples-to-apples guarantee currently lives in human memory and AGENTS prose.
7. **Driver UX for the `all` trap:** when the target has a go.work and the pattern is `all`, print a warning pointing at the dependency-closure semantics (the trap already caught two sessions).
8. **Cross-repo TODO discipline:** the cmdguard finding belongs to cmdguard's queue; keep the pointer in this repo's TODO (done) but avoid growing this repo's TODO into a multi-repo dumping ground — each repo's AGENTS should carry its own next step (done for auditlog/schema; pending for cmdguard).
9. **Release/version coupling is manual:** consumer CI pins ↔ linter releases move together; a release checklist entry (or renovate-style PR) should update auditlog/schema/branching-flow pins in the same round as tagging.
10. **Load-error triage debt:** Kernovia (go.work 1.27 floor), reports/app (redeclaration), 2× archived go.mod rot — project-side, pre-existing, and now the *only* errors left in the survey; small fixes each, big hygiene win.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT (sorted by impact; tier 1 = this week, tier 5 = backlog/roadmap fuel)

**Tier 1 — unblock everything (needs your answer to Q1)**
1. Push `master` and watch the first all-green CI run (v2.13.2 lint pin gets its first real test on GitHub).
2. Run full `nix flake check` on the current tree (immediate, unblocked).
3. Tag v0.2.2 (schema-v2 baseline, per-module go.work scan, shared loader, snippet-gate scope).
4. Wire the health-wash CI job into samber-do-auditlog pinning v0.2.2 (recipe in their AGENTS).
5. Wire the health-wash CI job into standard-bug-tracking-schema pinning v0.2.2.
6. Finish the triage-doc refresh from the corrected table (b.1; data ready).
7. Wire CV's baseline into CV's own gates (flake check or CI) so the ratchet actually bites (see Q3).
8. Delete the "first all-green CI run unobserved" TODO item once run 1 proves green; correct AGENTS/CHANGELOG "unobserved" language.

**Tier 2 — correctness/hygiene this round created or exposed**
9. cmdguard: decide + implement the HW-2 resolution (Q2), then re-survey to see cmdguard's own row go clean.
10. Add shellcheck to `nix flake check` for `scripts/`.
11. Survey diff mode (delta vs previous run; load-error flips; per-pseudonym coverage trend).
12. Persist survey metadata (timestamps, scanner version, per-project duration) into the output file.
13. Explain the Standup-Killer flip (check its git log for an external dep fix; note in survey docs if external).
14. Annotate the 2026-09-10 triage doc as superseded, pointing at `docs/ecology/2026-09-16-scan-per-module-fixed.txt`.
15. Add a README to `docs/ecology/` explaining the paired pre-fix/post-fix tables and which is canonical.
16. End-to-end test that the CLI (not just `load`) survives a hostile global `GOFLAGS=-mod=vendor`.
17. Fixture for dependency-package self-registrations in `testdata/` so the per-module doctrine is encoded in Go tests, not only in the bash script.
18. Verify branching-flow's `doanalyzerv2` still compiles against the driver changes (Analyze addition is additive, but it consumes this repo).
19. Check whether README documents `pkg/sdk` (suspected gap from the parallel session); add a short section if not.
20. Confirm standard-bug's AGENTS.md note got committed (it was still uncommitted at report time; daemon should have it by now).
21. `golangci-lint` version-policy sweep outside this repo: auditlog's CI still caches/pins **v2.12.2** (`lint` job cache key + install) — violates the "one version everywhere = v2.13.2" policy; same audit for standard-bug and branching-flow.
22. FP-BUDGETS.md: add the HW-unresolved budget row (cmdguard showed 4 under `all`; document that per-module scanning keeps it at 0 for honest scopes).
23. Prune `archived/` from the default survey root (flag like `--include-archived`); 2 of the 4 remaining load errors are archived repos.

**Tier 3 — linter product work (queued/roadmap)**
24. HW-4 posture implementation once you decide (default-on info vs opt-in flag).
25. HW-7 stale-directive rule prototype (valid-but-orphaned directives with `until` expiry).
26. Baseline schema v3: store scan patterns; fail loudly on scope mismatch (improvement 6).
27. Driver warning on `all` + go.work (improvement 7).
28. `--output json` mode for the survey (machine-readable diffs for CI/PR comments).
29. Move survey scoring/aggregation from bash into a small Go command (testability, reuse of the finding model).
30. Consider a rule for "library self-registrations" (cmdguard-class: a library registering itself into a consumer's injector) — document as out-of-scope for consumers if rejected.
31. samber/do drift: when v2.2.x lands, re-run the drift matrix + mechanism assertions (version-pinned facts policy).
32. Plugin integration: verify the golangci custom build (`plugin/`) against v2.13.2 in CI remains green after the loader consolidation.
33. Keyfile bootstrap doc: how to regenerate/migrate `~/backups/ecology/keyfile.json` on a new machine (it is the only place real names exist).
34. ROADMAP entry: healthaudit runtime companion phase 3 scope review.
35. ROADMAP entry: doanalyzerv2 DO-9 family backport completion.
36. Docs: README section for `scripts/ecology-scan.sh` usage + interpretation of the ranked table.

**Tier 4 — project-side debt surfaced by the survey (other repos, small fixes)**
37. reports/app: fix the `sessions` redeclaration (internal/auth/session.go:15/17) — only real Go bug the survey can see.
38. Kernovia: go.work demands go 1.27 floor — either bump toolchain availability or pin workspace lower; re-enters survey when fixed.
39. archived/experiment-100m-arr + website-holger-hahn: `go mod tidy` rot — tidy or delete the archived repos (your call).
40. Standup-Killer: record what fixed its dep mismatch (if a parallel session did it, AGENTS there should say so).

**Tier 5 — standing user decisions (unchanged, awaiting direction)**
41. HW-4 posture (decision item).
42. `.crush` history purge (decision item).
43. Upstream ownership + SLA for samber/do#317/#318 (decision item).
44. Whether design notes beyond #318 wait for a maintainer ask (decision item).
45. Survey scope canonicalization: keep `~/projects` (66 analyzed, incl. archived) as the default root, or exclude `archived/` permanently (overlaps 23; needs your preference).
46. Whether ecology scan should also cover `~/backups` or non-project trees (almost certainly no — recorded so nobody "discovers" it later).
47. Release cadence: batch v0.2.2 with the sibling CI wiring in one round (recommended) or release first, wire later.
48. CV: drive the 20-unprotected-services number down (rank 3 exposure) — candidates for honest `HealthCheck(ctx)` implementations or documented suppressions.
49. auditlog: same exercise at 8 unprotected (rank 15) — it is the biggest *healthy* exposee after CV.
50. Celebrate criterion: define what "ecology green" means (e.g. all non-archived projects clean or baselined) so the survey has a finish line instead of a moving one.

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Push + release authorization (blocks tier 1):** May I push `master` (~7 unreleased commits: lint pin, baseline v2, loader consolidation, survey fix, snippet gate) and then tag **v0.2.2**? Everything in tier 1 except the flake check hangs on this — the sibling CI wiring cannot reference a version that does not exist, and the "first all-green CI run" claim stays unprovable.
2. **cmdguard API direction (blocks its HW-2 resolution):** For `*CLI[T]` (bare `HealthCheck()` at `cli_accessors.go:42`, self-registered via `Package()`): the ctx variant `HealthCheckWithContext` already exists, but the do.Healthchecker interface needs the literal `HealthCheck(context.Context) error` name, which collides with the existing bare method. Do you want (a) a breaking signature change of the public `HealthCheck()` across cmdguard consumers, (b) an adapter type registered instead of the CLI, or (c) a documented `//samber-linter:allow hw-2 <reason>` suppression in cmdguard?
3. **Cross-repo enforcement scope:** The baselines I committed (CV 10/30, auditlog 12/20, standard-bug 2/61) currently prove nothing automatically — CV has no wiring at all, and the siblings get CI jobs only after the release. Do you want me to wire enforcement everywhere it fits (CV flake check + sibling CI + branching-flow) as a standing pattern in the same round as the release, or keep the linter survey-only in CV for now?

---

*Point-in-time snapshot; goes stale on contact with the next push. Section (f) tiers 1–2 are the primary HARVEST input for TODO_LIST/ROADMAP; the standing decisions are already mirrored in TODO_LIST "Open decisions (user)".*
