# Status Report — Upstream Filing & Verification Round (samber-linter)

**Date:** 2026-09-10 06:50 CEST
**Session scope since last report (04:05):** HW-4 explanation, go-health/go-health-dashboard
research, verify-before-filing execution, filing samber/do#317 + #318, humanizing both
issues in the owner's voice, discovering and fixing a broken filed repro, and a
verification-flaw incident (map-index gotcha) that turned into a stronger result.
**Repo state:** clean except `.config/metadata.yaml` (modified, NOT by this session —
uninvestigated). Both upstream issues OPEN, no maintainer response yet.

---

## TL;DR

Two upstream issues filed on samber/do after running the verify-before-filing gates:
#317 (transient services render nil/pass even when the type implements a check —
with the newly-found internal inconsistency: `isHealthchecker()==false` yet the
sweep lists the service as passed) and #318 (explicit health outcome states —
strengthened by the runtime-discovered `if !s.built { return nil }` line). Both were
rewritten into the owner's first-person voice with real project links and the
GLM-5.3-Flash-via-Crush disclaimer. Two quality incidents occurred in this segment:
the filed repro contained three compile-level errors (fixed, now literally executed),
and my first "verification" of that repro was vacuous (map-index gotcha) — exposed by
a control experiment that ended up *strengthening* both issues.

---

## The three framing questions

### What did I forget?

1. **Compile the repro before filing.** The verify-before-filing gates checked the
   diagnosis, duplicates, and platform — but not that the issue's own code snippet
   compiles. It had three errors (`NewWithOpts` takes `*InjectorOpts`, not option
   funcs; `do.WithHealthCheckTimeout` doesn't exist; package-level
   `HealthCheckWithContext` returns `error`, not the result map). Filed publicly
   broken in the user's name; fixed only when asked "anything to perfect?".
2. **A print that can't distinguish missing-from-present verifies nothing.** My
   first "ran and verified" used `res["repro.check"]` — wrong key (actual:
   `main.check`) and a missing `map[string]error` key also prints `<nil>`. The
   verification was vacuous and I declared it anyway.
3. **The oops research answer was left dangling** — I started verifying `Unwrap` on
   samber/oops and never closed it out when the conversation moved to issue polish.
4. **`.config/metadata.yaml`** is modified in the working tree, not by this session
   (parallel session or tooling). Noticed at report time, not investigated.
5. **Carried over, still forgotten:** AGENTS.md sync (golangci registration trap,
   workspace `all`, exit-2 load failures, map-index lesson), `nix flake check`,
   GitHub CI verification, `v0.1.1` tag (CHANGELOG says 0.1.1, main.go says 0.1.0),
   dprint pass, `toFinding`'s dead `Version` param.

### What could I have done better?

1. **Gate the snippet, not just the thesis.** "Verified" must mean every code block
   in a filed issue was compiled and run verbatim. That is now a personal rule:
   repro code gets the same treatment as test code.
2. **Distrust my own proofs.** The control experiment (lazy singleton) was the thing
   that exposed the vacuous print — controls should be standard, not improvisation
   under user pressure.
3. **Read the control result correctly the first time.** My first interpretation of
   "lazy after invoke still nil" was wrong (same map-index flaw); only the full-map
   dump (`len=1, map[main.check:boom]`) gave the truth. The lesson generalizes:
   when a result contradicts source reading, dump the raw structure, never a
   projection of it.
4. **Write in the user's voice from draft one.** The issues needed three polish
   rounds ("who talks like this??", "these 2 sound like shit", "do not suggest").
   The finished voice — first person, no offer-speak, concrete numbers, one clean
   closing — should have been the starting point, since the issues were always
   going out under the user's name.
5. **Finish research threads or park them explicitly.** The oops thread ended
   mid-verification; the useful findings (Join delegates to stdlib errors.Join,
   Unwrap is linear, public-vs-developer message split) should have been delivered
   or explicitly deferred.

### What could I still improve?

1. **Upstream engagement loop:** both issues are filed but nobody watches them;
   response handling, follow-up comments, and a possible PR are unowned work.
2. **The go-health × healthaudit × linter triangle** is now documented in public
   issues but not implemented: `Check.Status` gaining `unknown`/`skipped` fed by
   healthaudit's registered/invoked/errored/skipped vocabulary would make HW-1/3/4
   runtime-visible. Design exists in the issue thread; no code.
3. **Repro-as-test symmetry:** the filed repro and the repo's
   `upstream_transient_test.go` drifted (repro uses bare `do.New`, test uses the
   auditlog recorder). Keep one canonical version.
4. **Verification hygiene tooling:** a repo check that compiles every Go snippet in
   docs/upstream/*.md would have caught the broken repro mechanically.

---

## a) FULLY DONE

| Item | Evidence |
|------|----------|
| verify-before-filing executed | skill loaded; source gates against module cache (service_transient.go:58-66, scope.go:733-735, service_lazy.go:128-134, root_scope.go:208, injector.go:51-52); duplicate search (open+closed, none found); Discussions-disabled check (`has_discussions: false` → Issue, not Discussion) |
| samber/do#317 filed | https://github.com/samber/do/issues/317 — transient nil/pass + internal inconsistency + three fix directions + executable repro |
| samber/do#318 filed | https://github.com/samber/do/issues/318 — outcome states (Passed/Failed/NotBuilt/Unsupported/Skipped), grounded in `service_lazy.go:132-134`, additive API sketch, prior art, references #317 |
| Repro made literally executable | standalone module compiled and run: `map[main.check:<nil>]`; snippet in #317 updated to the full-map print with the missing-key warning |
| Lifecycle runtime-verified (3 experiments) | transient → nil (never dispatched); lazy before invoke → nil (never built); lazy after invoke → `boom` (check dispatches). Full-map dumps, not projections |
| Internal inconsistency documented | `isHealthchecker()==false` yet sweep lists the service as passed — added to #317 Evidence |
| Humanized both issues | first person, owner's voice, real links (go-health, go-health-dashboard — verified: dashboard imports go-health v0.1.3), linter numbers (43 repos, 66 findings, 30/24/1 breakdown), stilted phrasings removed, offer-speak removed from #318, single clean PR offer on #317 |
| GLM-5.3-Flash via Crush disclaimer | appended to both issues (small `<sub>` footer) |
| Multi-error design answered | `Err error` stays plain `error`: `errors.Join` covers multi-failure (verified oops.Join delegates to stdlib; Unwrap linear); `#Errs []error` rejected; noted in #318 as inline comment |
| Local records synced | `docs/upstream/ISSUE_DRAFT.md` marked FILED with both links + verification notes; TODO_LIST checked off; daemon-committed |

## b) PARTIALLY DONE

| Item | State |
|------|-------|
| Upstream engagement | #317 and #318 filed, OPEN, 0 comments, no response yet; follow-up/PR unowned |
| go-health integration idea | `unknown`/`skipped` statuses fed by healthaudit discussed in #318 context; no implementation in go-health or healthaudit |
| oops lessons | researched (Join→stdlib, Unwrap linear, hint/public/owner audience split, duration as field) but the delivered answer was incomplete; not folded into #318 beyond the Join comment |
| Lint debt | 141 findings unchanged this segment (untouched by design); CI lint still red |
| Still-open user decisions | HW-4 posture; CV baseline lock-in (CV now reads 10% vs 8% baseline); GitHub `.crush` purge; v0.1.1 tag |

## c) NOT STARTED

- Watching/responding to #317/#318 maintainer activity
- healthaudit: formal `Status` enum (registered/invoked/errored/skipped → typed)
- go-health: `Check.Status` extension (`unknown`/`skipped`) + classifier wiring
- PR preparation for whichever #317 direction the maintainer picks
- `.config/metadata.yaml` investigation (foreign modification)
- Everything carried from the previous report's "not started": rank-1 ecology triage,
  samber-do-auditlog fixes, FluffBall/KeyCountdown MISSING_DIR, CI run verification,
  `nix flake check`, v0.1.1 tag/release, AGENTS.md update, dprint pass

## d) TOTALLY FUCKED UP

No file destruction this round (lesson from the last one held: exact-match edits
only). But two quality incidents, both public:

1. **Filed a broken repro.** Three compile-level errors in a snippet published under
   the user's name on a 2.8k-star repo. A maintainer running it verbatim would have
   gotten compile errors — the exact "professional-looking but wrong" failure
   verify-before-filing exists to prevent. My gates verified the diagnosis but never
   executed the snippet. Fixed within the session (now compiled + run standalone),
   but the window was real.
2. **Declared a vacuous verification.** Printed `res["repro.check"] == <nil>` from a
   map where the real key was `main.check` — a missing key prints identically. Said
   "ran and verified" over a print that could not distinguish missing from present.
   The user's pushback ("ran and verified for real?") forced the redo; the redo's
   control experiment then exposed that lazy-unbuilt singletons ALSO report nil,
   which materially improved both issues. Recovery was better than the original
   claim; the original claim should never have been made.

## e) WHAT WE SHOULD IMPROVE

1. **Snippet gate:** every Go block in a filed issue gets compiled and run verbatim,
   full raw output captured — add a tiny check script over `docs/upstream/*.md`.
2. **Full-dump discipline:** verify via raw structures (whole map, whole diff),
   never via projections (indexed lookups) that conflate missing/zero.
3. **Voice-first drafting:** upstream text starts in the owner's first-person voice
   with concrete numbers; polish rounds should be for facts, not for de-robotizing.
4. **One canonical repro:** keep the upstream repro and the repo test in lockstep
   (same file, build-tagged) so they cannot drift.
5. **Response ownership:** decide who watches #317/#318 and in what timeframe;
   an unanswered verified issue decays fast.
6. **Close research threads:** the oops audience-split insight (public vs developer
   message — relevant to dashboards leaking internals) deserves a delivered answer
   or an explicit TODO, not an abandoned grep.

## f) UP TO 50 THINGS TO DO NEXT

**Upstream (1–8)**
1. Watch #317/#318; respond to maintainer questions within a day of activity
2. Prepare the #317 option-1 PR branch in advance (sentinel `ErrHealthCheckSkipped`) so filing a PR is a 5-minute act if invited
3. Add the #318 design notes only if a maintainer asks — do not dump unsolicited
4. Cross-link: comment on #317 pointing at the now-runtime-verified lazy lifecycle (nil → build → boom) as supporting evidence for #318
5. Check whether `ErrHealthCheckTimeout`/panic sentinel interplay needs a note in #317 (timeout must win over partial failures)
6. Search samber/do PRs (not just issues) for prior transient-healthcheck attempts
7. Decide whether go-health should pin a fork/patch if #317 stalls
8. Link #317/#318 from samber-linter README (credibility + upstream traceability)

**Runtime health model (9–16)**
9. healthaudit: introduce typed `Status` enum (Registered/Invoked/Errored/Skipped)
10. healthaudit: expose sweep results as outcomes (not just counters) for go-health consumption
11. go-health: extend `Check.Status` with `unknown`/`skipped`, fed by healthaudit outcomes
12. go-health: classifier lattice update (Unknown < Pass < Warn < Fail) + per-state dashboard colors in go-health-dashboard
13. go-health: startup latch already treats absence as not-started — document interplay with new statuses
14. aggregrate package: propagate per-source outcome states through merged keys
15. HW-1/3/4 → runtime mapping table in samber-linter docs ("what the dashboard should show once states exist")
16. Sample: wire go-health-dashboard to render `unknown`/`skipped` distinctly

**samber-linter (17–26)**
17. Compile-check all Go snippets under docs/upstream in CI (script + job step)
18. Keep upstream repro and `upstream_transient_test.go` in lockstep (single source)
19. v0.1.1: bump main.go version, tag, push release (CHANGELOG entry exists)
20. Watch first green CI run; fix lint job policy (fix vs rescope the 141)
21. `nix flake check` on committed tree
22. AGENTS.md: registration trap, workspace `all`, exit-2 load failures, map-index lesson, voice rules for upstream text
23. dprint pass over hand-edited markdown
24. Remove dead `Version` param in `toFinding` (or use it in findings)
25. Re-run the 43-project ecology sweep post-changes; store as repo script
26. HW-4 posture implementation once decided (opt-in vs on-by-default)

**Ecology triage (27–33)**
27. standard-bug-tracking-schema: begin rank-1 triage (4×HW-1, 63 unprotected)
28. samber-do-auditlog: fix 7×HW-1 + 5×HW-2 + 1×HW-3 (its HW-3 is the live #317 case)
29. Re-scan CV post-fixes; commit updated baseline (10%)
30. FluffBall / KeyCountdown MISSING_DIR investigation
31. Kernovia go.work (needs go ≥ 1.27): upstream courtesy note or local fix
32. ast-state-analyzer: `go mod tidy`, re-scan, close anomaly
33. Refresh pseudonymous triage doc with post-fix numbers

**Hygiene (34–40)**
34. Investigate `.config/metadata.yaml` foreign modification; commit or revert consciously
35. Clean stale /tmp artifacts (sl-test, sl-final, samber-linter binaries, repro dirs)
36. `git worktree prune` (worktrees removed, metadata may linger)
37. dprint/golines formatting sweep on remaining hand-edited files
38. TODO_LIST: fold this round's upstream items into the living sections (currently only a checklist line)
39. Verify no concurrent-session clobbering of output.go/driver.go since last check
40. Dependabot/renovate for plugin-module-register and golangci-lint version pins

**Bigger ideas (41–50)**
41. `samber-linter explain HW-N` subcommand (rule docs from code single-source)
42. Baseline v2: per-rule counts + schema version
43. `--check` silent under `--json`
44. SARIF: coverage as automation metric property
45. Rule severity table generated from `ruleMetaByRule` into README
46. Suppression age report (`--suppressions` listing directives + expiry)
47. Config auto-discovery (`samber-linter.yml`)
48. Benchmark analyzer on largest consumer (63 services) — load-time budget
49. Fuzz the allowlist config parser
50. Design note: HW-2's runtime analogue — bare checks vs sweep timeout (bridge doc between HW rules and go-health timeouts)

## g) QUESTIONS (cannot self-answer)

1. **HW-4 posture (still open from last report):** on-by-default `info`/Medium, or opt-in? The upstream #318 discussion makes "lazy = green until built" a *state* question rather than pure noise — does that change your answer?
2. **Upstream identity & follow-up:** #317/#318 are filed under your GitHub account. Do you want me to own responses (draft replies for your review within a day of activity), or do you want to handle maintainer dialogue personally?
3. **`.config/metadata.yaml`** is modified in the working tree and this session did not touch it. Yours? Another session's? Should I leave it strictly alone?

**Waiting for instructions.**
