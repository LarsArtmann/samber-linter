# Status Report — TODO-queue execution: wrapper detection, rule table, toolchain (2026-09-20 18:40)

Session scope: execute the six-item short-term queue from `TODO_LIST.md`
(consumer ratchet wiring, CV runbook, HW-9 renumber, defaultRules single
source, wrapper-indirection false negative, ecology triage refresh) — plus
everything that fight turned up. Local gate at close:
**`nix flake check` all green** (build, full test suite, treefmt, dprint,
hermetic golangci-lint v2.13.2 at floor 1.27.1), dogfood run exit 0,
upstream-snippets gate PASS. GitHub CI has **not** validated this session's
work (nothing pushed since 15:29 — see d).

## a) FULLY DONE

1. **Item 3 — stale-directive renumbered to HW-9.** `FEATURES.md` renumbered;
   the dangling "design notes in docs/FP-BUDGETS.md" pointer made true by
   writing the design + FP budget (budget 0: syntactic fact; ships only on
   explicit go-ahead).
2. **Item 4 — single-source rule table.** New `pkg/healthwash/rulemeta.go`:
   `RuleTable` (ID, slug, severity, confidence, summary, DefaultEnabled) is
   now the one definition consumed by the driver (`toFinding` lookup,
   hand-map deleted), the golangci plugin (generated `BuildDocs`), and both
   README drift tests (registry + new slug-drift assertion). Posture flips
   are one line via `DefaultDisabledRules`, wired into driver AND plugin.
   New tests: table-covers-constants guard, well-formedness, empty-defaults
   posture pin, lookup, `TestToFindingHonorsRuleTable` at the driver.
3. **Item 5 — wrapper-indirection false negative FIXED.**
   `pkg/healthwash/wrapper.go`: one-level, package-local wrapper resolution —
   a package-level function (generic or plain) whose body holds exactly ONE
   `do.*` registration with a bare parameter in the provider slot maps that
   parameter to the call-site argument (object-identity binding). Rules fire
   and suppressions bind at the wrapper CALL SITE; the parameterized body
   call is never reported or counted (kills the old phantom
   HW-unresolved records). Fixture `testdata/src/hwwrap` (generic + plain +
   eager wrappers, explicit instantiation, suppression-at-callsite, clean
   negative); discrimination-proof row `HW-1 via hwwrap`. Docs: README §4,
   FP-BUDGETS (structural guard #6 + two boundary bullets), CHANGELOG,
   FEATURES, AGENTS.md.
4. **Item 6 — pseudonymous triage doc refreshed** to the 2026-09-20 scan:
   8 findings-bearing projects queued (19 findings), load-error wave called
   out as environmental (32 consumers require go ≥ 1.27.1), deltas vs
   2026-09-10 documented, aggregate updated.
5. **Item 1 — consumer ratchet gates wired @v0.2.2** (jobs did not exist;
   both AGENTS.md said "pending"): `healthwash` job added to
   samber-do-auditlog and standard-bug-tracking-schema `ci.yml`, baseline
   path trigger added (schema repo). Both verified **green at v0.2.2
   locally** (60% / 3%, baseline acknowledged, exit 0). Bonus: auditlog's
   own go-version drift guard was red (go.mod 1.27.1 vs pins 1.26.7) —
   fixed across ci.yml / flake.nix / .golangci.yml per its documented
   procedure; guard now passes. Both AGENTS.md wiring notes updated.
6. **Item 2 — CV runbook + executed bump.**
   `docs/operations/healthwash-gate-runbook.md` (pin-bump paragraph of
   record: edit → `-version` build-gate → gate run → adjudicate; never
   `--set` during a bump; never pin back to silence). Then performed it:
   `LINTER_REF` v0.2.1→v0.2.2, provenance header updated, `-version` gates
   to `v0.2.2`, gate PASS at 10/31 = 32% = committed floor.
7. **Toolchain realignment (unplanned, blocking):** go.mod floor had been
   auto-committed to 1.27.1 leaving nix/CI at 1.26 — flake now pins
   `goPkgAttr = "go_1_27"` (locked nixpkgs ships exactly 1.27.1),
   `apps.test` runtimeInputs bumped, CI pins `go-version: "1.27"` ×5, stale
   comments fixed. Local builds were silently broken before this.
8. **treefmt sandbox trap fixed:** goimports shells out to `go`; in the
   no-network check sandbox any go older than the floor dies with
   "go: downloading goX" DNS errors — and treefmt-nix registers the check
   TWICE (`checks.treefmt` + go-standard's `checks.format`). Fixed locally
   (`hermeticTreefmtCheck` override: go_1_27 + GOTOOLCHAIN=local, mkForce
   on both attrs) AND at the root in go-nix-helpers' go-standard module
   (unpushed — see b).
9. **Bookkeeping:** TODO_LIST rewritten (completed items deleted →
   CHANGELOG), CHANGELOG Unreleased (wrapper channel, HW-9 reservation,
   rule table, toolchain, ecosystem wiring), FEATURES DONE entries +
   date-stamp, AGENTS.md memory updated (wrapper boundary classes,
   toolchain policy, treefmt gotcha), dprint formatting fixed repo-wide
   (incl. a concurrent session's unformatted file).

## b) PARTIALLY DONE

- **go-nix-helpers module fix — committed locally, UNPUSHED.** samber-linter
  therefore carries a documented local override that duplicates it: a
  temporary split brain that must be collapsed (push → `nix flake lock
  --update-input go-nix-helpers` → drop the local copy). I do not push
  without permission.
- **Wrapper detection is v1 by design:** one level, package-level functions
  only. Still invisible: wrapper chains, cross-package wrappers, wrapper
  methods, closures, multi-registration bodies. Documented in three places;
  no negative fixture for chains yet (no real consumer shape observed).
- **Ecology impact of the wrapper channel unmeasured:** the 2026-09-20 scan
  predates the fix; a re-scan needs a go ≥ 1.27.1 scanner toolchain (the
  same wave that blocks 32 consumers).
- **Consumer CI validation:** jobs verified green locally with the exact
  pinned command; their GitHub CI runs only after the repos are pushed.

## c) NOT STARTED

- The four open user decisions (left in TODO_LIST): `--min-confidence`
  default flip (needs one release of FP data), GitHub `.crush` history
  purge, upstream ownership for samber/do#317/#318, HW-9 go-ahead.
- v0.3.0 release cut (all content sits in Unreleased; README §12 correctly
  still claims v0.2.2).
- Anything requiring pushes: go-nix-helpers, samber-linter master, both
  consumer repos, CV.

## d) TOTALLY FUCKED UP (honest)

1. **Broke flake.nix syntax with a blind edit**: my first
   `checks.format` override landed after the `in`, producing two `in`s.
   Caught by viewing the file immediately — but a view-before-edit would
   have avoided it entirely.
2. **Three wasted full `nix flake check` cycles on a blind spot:** I
   overrode `checks.format`, verified that ONE attr builds, and assumed
   victory; the full check kept failing because treefmt-nix ALSO registers
   `checks.treefmt`. I should have run `nix flake show` and enumerated the
   checks before fighting logs. Lesson recorded in AGENTS.md.
3. **Guessed a stdlib API**: `types.Var.IsParam()` does not exist
   (compile error). The identity-match against `sig.Params()` was already
   sufficient — the check was cargo-culted. Verify APIs before writing.
4. **Fixture type error on first compile** (`provideNamed[CheckedStore]` vs
   provider returning `*CheckedStore`) — re-reading the fake do's generic
   shape before writing would have caught it.
5. **9 lint findings in MY new code, found in two late rounds**
   (nonamedreturns, varnamelen, wsl_v5 ×4, nlreturn, cyclop, gocognit).
   Root cause: wrote ~500 lines against a 108-linter config and ran the
   hermetic lint only after formatting rounds. Running
   `nix build .#checks.x86_64-linux.lint` right after the first compile
   would have surfaced all of them in one pass.
6. **Remote CI never saw this session's work.** The daemon commits locally
   but nothing has been pushed since 15:29; "CI green" currently describes
   only the last pushed run (TODO_LIST's own rule). My verification is
   local-nix-only; that distinction was not stated loudly enough at wrap-up.
7. **Concurrent-session log confusion:** another agent's mid-flight commits
   produced lint/nix logs against snapshots that no longer existed; I
   chased one ghost log (`writeBaseline` findings) before realizing the drv
   hash didn't match my eval. Pin derivation paths when triaging.

## e) WHAT WE SHOULD IMPROVE (session-derived)

- **Enumerate before you fight**: `nix flake show --json` first when a
  check fails; overrides are cheap, blind spots are not.
- **Lint early, lint targeted**: hermetic lint check is directly buildable
  per-attr (`nix build .#checks.x86_64-linux.lint`) — run it immediately
  after each new Go file stabilizes.
- **Collapse the temporary treefmt split brain**: push go-nix-helpers,
  update samber-linter's input lock, delete the local override.
- **Push cadence**: local-green ≠ CI-green; decide who pushes and when
  (daemon does not push). TODO_LIST's quality-gate note should get a
  "last locally-verified" companion line.
- **API verification discipline** for stdlib/x-tools surfaces (IsParam
  class of error).
- **CV AGENTS.md does not link the new runbook** — one-line docs wiring
  missed; add the pointer in CV's next session.
- **Wrapper boundaries deserve fixture coverage as shapes appear** (chains,
  methods, closures) so each v2 widening is discrimination-proven, not
  prose-only.

## f) Up to 50 next things (harvest pool — most are ROADMAP fuel; the
bounded few already live in TODO_LIST)

1. Cut **v0.3.0** (wrapper channel + HW-8 + rule table): README §12 line
   FIRST, then tag (drift test enforces order).
2. Push go-nix-helpers; update samber-linter's input lock; drop the local
   `hermeticTreefmtCheck` override.
3. Push samber-linter master; confirm all five CI jobs green on the new
   floor.
4. Push both consumer repos; watch their first `healthwash` job runs.
5. Push CV; its healthwash gate now pins v0.2.2.
6. Re-run the ecology scan on a go1.27.1 toolchain (unblocks the
   32-consumer load-error wave) and measure wrapper-channel impact.
7. Add wrapper-chain negative fixture when a real shape appears.
8. Consider wrapper methods (`r.provide(...)`) as wrapper v2 — needs a
   receiver-aware calleeIdent.
9. Consider closure wrappers (`var wrap = func(...)` bodies) as v3.
10. Add HW-9 implementation once the user gives the go-ahead (design + FP
    budget already recorded).
11. Collect one release of max-recall FP data (`--min-confidence 0.5
    --strict` runs in consumer CI) → feed the default-flip decision.
12. Add the CV runbook pointer to CV's AGENTS.md.
13. Add a README §11 verification-ledger row for the wrapper channel
    (fixture + discrimination proof as evidence links).
14. Snapshot-test the plugin's generated `BuildDocs` string against the
    table (cheap drift guard, currently only indirectly covered).
15. Driver e2e test: a stub module using a wrapper (currently only
    analyzer-level fixtures cover the channel).
16. Check golangci-lint version policy after the 1.27.1 floor: v2.13.2
    still builds ≥ floor (verified by green lint check) — re-pin when
    nixpkgs moves.
17. Consider `--enable` flag design so a future default-off rule can be
    re-enabled per-run (prerequisite for any posture flip).
18. Export `RuleTable` through `pkg/sdk` if BuildFlow wants rule metadata.
19. Docs: README §6 mention that the plugin honors the same default
    posture table (one sentence).
20. Audit remaining `docs/status/**.md` Go blocks for the snippet gate
    after any future doc edits (gate exists; habit doesn't).
21. CV: chip at the 21 unchecked services (their runbook process).
22. standard-bug-tracking-schema: 59 unprotected-but-rule-silent services
    — evaluate whether a rule candidate exists there or precision is right.
23. samber-do-auditlog: raise the 60% floor after next real checks land.
24. Add `last locally verified` timestamps to TODO_LIST's quality gate.
25. go-nix-helpers: fix its own pre-existing `--no-build` eval quirk
    (`templ-committed` attr; verified pre-existing at HEAD 5574810).
26. go-nix-helpers: CI for the module change (its own flake check).
27. Consider `nix flake check --all-systems` viability (aarch64 currently
    omitted).
28. Tag the treefmt-check fix in go-nix-helpers CHANGELOG if it keeps one.
29. Regression-test `DefaultDisabledRules` merge behavior in the driver
    (unit test with a synthetic table consumer is impossible today — the
    table is a package global; consider an injection point if v2 posture
    work starts).
30. Review whether `innerSkips` should also skip suppress-directive
    collection inside wrapper bodies (currently directives there are inert
    by position — probably right, but unstated).

## g) Questions I can NOT figure out myself

1. **Push authority**: may I push (go-nix-helpers, samber-linter, the two
   consumers, CV), or is pushing exclusively yours? The go-nix-helpers fix
   cannot propagate to samber-linter's lock without a push, and none of
   today's green gates are CI-proven until something is.
2. **v0.3.0 timing**: cut it now from Unreleased (HW-8 + wrapper channel +
   rule table), or hold it for the ecology re-scan / any pending decision?
   The wrapper channel may surface new findings in consumers — ship-then-
   surface, or measure-first?
3. **HW-9 go-ahead now or parked?** Design + FP budget are recorded;
   implementation is ~a day with fixtures. Your call per the standing
   decision.

---
*Generated by Crush (GLM-5.3) 2026-09-20 18:40 CEST from this session's
work only. Format note: written as Markdown at the user's explicit
`.md` request — the status-report skill's canonical format is HTML.*
