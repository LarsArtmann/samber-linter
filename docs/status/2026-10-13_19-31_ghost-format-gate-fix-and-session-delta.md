# Status Report — session delta: ghost format gate, untracked-lag, markdown hygiene

**Date:** 2026-10-13 19:31 CEST
**Scope:** continuation snapshot, one hour after `2026-10-13_18-23_samber-linter-buildflow-integration-hw7-wiring-session.md`. No new feature work this hour; everything below is verification, a broken-gate fix, and honest bookkeeping. Carried items from 18:23 are referenced, not re-argued — that report remains the base for sections it covered.
**Repos:** `~/projects/samber-linter` (all changes), `~/projects/BuildFlow` (read-only this hour).

---

## Direct answers first

### What did I forget?

1. **The verifier itself.** Last turn ended with the repo "green" — but the green `checks.format-dprint` was a **ghost gate**: it scanned only its own sandbox (`$TMPDIR`, where the rewritten dprint config lived), so it validated **dprint's own plugin-cache metadata (4 files) and never a single repo file**. Months of green meant nothing. I ran it this hour to verify my markdown and it failed on 4 cache files — the failure was the most useful output it ever produced.
2. **File-type ownership of formatters.** I edited markdown all session (AGENTS.md ×3, the 18:23 report) and ran only `nix fmt` — which is treefmt, owning **go/nix only**. dprint owns md/json/yaml/dockerfile and is a separate check. My report's tables and `SECURITY.md` were non-canonical; `dprint fmt` fixed both (the `SECURITY.md` drift predates this session and had survived the ghost gate for months).
3. **Unverified daemon claims.** Last turn I told the user "the auto-commit daemon will pick it up" — repeating the status-report skill's stale assumption against the buildflow skill's explicit warning ("known blind spot; verify with git status"). Reality: the 18:23 report sat untracked for ~25 minutes before commit `6a3d296` picked it up. Slow, not dead — but my claim was habit, not evidence.
4. **The HW-8 lesson generalized one level up:** last turn was "check the facts before writing the claim"; this hour added "check the verifier before trusting the check". Same disease, one level meta.

### What could I have done better?

- My **first gate fix was wrong**: I theorized "cache lands in `$HOME` scope" and pinned `DPRINT_CACHE_DIR` via `mktemp -d` — which defaulted into `/build` again, still inside the scan scope. The rebuild falsified it in one minute; the ground-truth run (`dprint check` directly on the repo with the real config) then nailed the actual model: **dprint scans the config's directory**, so the rewritten config must live inside a copy of the source tree, with the cache outside it. I almost stopped at theory #2 without the ground-truth run — that would have shipped another plausible-sounding non-fix.
- I could have run `dprint fmt` immediately after writing the 18:23 report instead of discovering the drift an hour later through a failing gate.

### What could still improve?

See (e). Headline: every quality gate deserves a one-time "does it scan the thing it claims to gate?" audit — this repo had at least one theater check, and the fleet likely has siblings.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                           | Evidence                                                                                        |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------- |
| A1 | 18:23 status report written, and committed (daemon lag ~25 min, `6a3d296`)                                                                                                                                                     | `git log docs/status/`                                                                          |
| A2 | **Ghost gate fixed**: `checks.format-dprint` now copies the flake source, writes the rewritten config _inside_ the copy, pins `DPRINT_CACHE_DIR` outside it, and runs `dprint check` from there — it now scans the actual repo | `flake.nix` (format-dprint script); `nix build .#checks.x86_64-linux.format-dprint` → **GREEN** |
| A3 | The fixed gate immediately proved real value: flagged my non-canonical report tables **and** pre-existing `SECURITY.md` drift (table padding only, verified benign by diff)                                                    | `git diff SECURITY.md`, report diff                                                             |
| A4 | Session markdown canonicalized via `dprint fmt` (2 files)                                                                                                                                                                      | `Formatted 2 files`                                                                             |
| A5 | The 18:23 report's A1–A7 all stand unchanged (no regressions; full suite and provider tests were green at 18:23 and nothing touched code since)                                                                                | 18:23 report §a                                                                                 |
| A6 | Trap recorded in memory: `nix fmt` = treefmt = go/nix ONLY; markdown/json/yaml need dprint; gate silently broken until today                                                                                                   | `AGENTS.md` build section, formatting bullet                                                    |

## b) PARTIALLY DONE

| #  | Item                                                                                                        | What's missing                                                                                                                                                                                                                                                                                          |
| -- | ----------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 | Ghost-gate fix                                                                                              | The scope model ("dprint scans the config's directory") is inferred from black-box behavior, not documented upstream. If dprint changes resolution, the gate could regress to ghost again. A **negative control** (deliberately broken file → expect red) would prove the gate end-to-end — not started |
| B2 | 18:23 §b items (B1–B4: HW-7 "fix" framing, BuildFlow doc sweep, HW-8 SDK/provider coverage, mutation rigor) | Unchanged — nothing touched them this hour                                                                                                                                                                                                                                                              |
| B3 | Policy questions g1–g3 (18:23)                                                                              | Still unanswered after ~1 h; they block the baseline cut, the release decision, and the portability fix                                                                                                                                                                                                 |
| B4 | `nix build` of BuildFlow after the master-pin flake edit (18:23 C4)                                         | Still open                                                                                                                                                                                                                                                                                              |

## c) NOT STARTED

| #  | Item                                                                                                                                                                         | Note                                                          |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| C1 | 18:23 §c C1–C7 (baseline ratchet, toolsdk migration, samber-linter release, BuildFlow doc sweep, BuildFlow `nix build`, RuleTable export, replace-portability)               | All unchanged; several blocked on g-answers                   |
| C2 | Negative-control proof for the repaired gate (see B1)                                                                                                                        | New this hour                                                 |
| C3 | Fleet-wide ghost-gate audit: which OTHER checks in this repo (`golangci-version-drift` — script inspected, looks real — and elsewhere) or in sibling repos validate nothing? | New this hour; one confirmed ghost is a strong prior for more |
| C4 | dprint ownership note is in samber-linter AGENTS.md but the same split exists in every fleet repo                                                                            | New this hour                                                 |

## d) TOTALLY FUCKED UP

1. **A quality gate that checked nothing, for months.** `checks.format-dprint` scanned its own sandbox cache and called it verification. This is the exact "green step that scanned zero files" anti-pattern — found not in BuildFlow but in the nix gate layer this session's own repo trusts. First real failure exposed: my session's report tables + months-old `SECURITY.md` drift.
2. **My fix #1 was a non-fix** (cache still inside scope); caught by rebuild, corrected by ground truth. If I had moved on after the edit without re-running the check, the repo would carry a confident-looking wrong comment describing a wrong model.
3. **Pre-existing `SECURITY.md` non-canonical formatting** survived every "green" `nix flake check` — harmless content-wise, but it is standing proof of the gate's blindness (now fixed).
4. **Stale assumption stated as fact to the user** ("daemon picks it up within seconds") — mild this time (25-min lag, file did land), but it is the same claim-shape as last turn's HW-7 misdiagnosis: assertion where a check was owed.
5. Carried: 18:23 §d1–d5 (false HW-7 defect report; shared-tree mutants; the HW-0..HW-7 bound miss; AGENTS.md split brain; weak-mutant near-miss) — all still stand as session history; none superseded.

## e) WHAT WE SHOULD IMPROVE

1. **Verify the verifier**: any gate that has never been seen red against a real defect gets a one-time negative control. Cheap, mechanical, catches ghost gates.
2. **Formatter ownership is per file type**: `nix fmt` ≠ markdown. My per-change loop must include "run the tool that OWNS this file type" — the ownership split is now documented in AGENTS.md, but the habit is the fix.
3. **Daemon statements need git-status evidence**; the skills already warn this — treat the warning as operative.
4. **Keep the theory→rebuild→ground-truth loop**: it caught my wrong fix #1 within one command. The loop is the improvement; skipping it is the regression.
5. **Ghost-gate priors are now fleet-wide**: one confirmed theater check justifies a sweep (C3) — in this repo first, then BuildFlow's provider/check layer.

---

## f) Things to get done next

The 18:23 list (30 items) stands unchanged and remains the primary backlog input for a `docs-health` HARVEST pass — not repeated here. **Delta items 31–37, new this hour**, sorted by impact:

| #  | Tag  | Task                                                                                                                                                                     | Impact                               |
| -- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------ |
| 31 | SL   | Negative-control the repaired `format-dprint` gate (break a file in a scratch branch → expect red)                                                                       | Proves the fix end-to-end; kills B1  |
| 32 | SL   | Audit every remaining flake check for the ghost class (does it touch repo files?): `golangci-version-drift` inspected and real; check the rest                           | One ghost found → sweep              |
| 33 | X    | Fleet sweep: same audit for sibling repos' format/lint gates (`nix flake check` layers in BuildFlow, go-cqrs-lite, templ-components, …)                                  | The pattern is probably not unique   |
| 34 | SL   | Consider a `dprint fmt`-style app (or treefmt wiring) so markdown formatting is one `nix run` away pre-commit                                                            | Removes today's trap mechanically    |
| 35 | SL   | `docs-health` HARVEST: fold 18:23 §f (30 items) + 31–37 into `TODO_LIST.md`/`ROADMAP.md`                                                                                 | Un-entombs both lists                |
| 36 | USER | Answer 18:23 g1–g3 (restated below)                                                                                                                                      | Unblocks C1, C3, C6 and (f) #5/#8/#9 |
| 37 | SL   | `buildflow`-skill note: add "verify daemon pickup with git status before claiming" is already there — verify MY closing-message habit follows it (process note, no code) | Claim hygiene                        |

## g) Questions I cannot figure out myself (unchanged, restated)

1. **Portability policy**: keep the machine-local `replace => /home/lars/projects/samber-linter` in BuildFlow's go.mods (nix-only buildability), or make other-machine builds resolve a tagged version?
2. **Dogfood gate**: cut BuildFlow's `.samber-linter-baseline.json` ratchet now, and at which confidence gate — or keep fix-as-found?
3. **Release cadence**: cut samber-linter v0.4.1 (test/docs only) now, or batch with the next rule change?

---

## Appendix: brutal-self-review delta (11 questions, one-liners)

1. **Forgot?** The verifier, the formatter-ownership split, daemon-evidence discipline (detail: top of report).
2. **Stupid thing we do anyway?** Trusting green checks we've never seen red; asserting daemon behavior from habit.
3. **Done better?** Run dprint right after writing the report; ground-truth the gate before theorizing fixes.
4. **Still improve?** Negative controls for gates; fleet ghost-sweep; per-type format habit.
5. **Did I lie?** Not deliberately; one stale assumption ("daemon picks it up in seconds") stated without evidence — it was 25 minutes off and I had been warned.
6. **Less stupid?** Every green needs either a seen-red or a scope audit.
7. **Ghost systems?** Found a literal one: `checks.format-dprint` — existed, ran, "passed", validated nothing. Now fixed and proven (it caught real drift within one run of being repaired). This is the session's headline find.
8. **Scope creep?** The gate fix grew from verifying my own files — in-scope cause, bounded effect (one script, one check).
9. **Removed something useful?** No; the old script's drift-guard section (plugin-pin cross-check) was preserved intact.
10. **Split brains?** Confirmed and documented one: treefmt-vs-dprint ownership (now explicit in AGENTS.md). The gate-vs-reality split is fixed, not merely documented.
11. **Tests?** Gate is green and meaningful; new test debt: the negative control (31). Code suites unchanged and green.
