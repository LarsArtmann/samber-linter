# Status: branching-flow DO/RO rule review + migration pro/contra (2026-10-05 15:01)

Session type: **read-only exploration + decision support**. No code changed, no
commits, repo tree clean. Working dir: samber-linter; explored:
`/home/lars/projects/branching-flow` (`pkg/doanalyzerv2`, `pkg/roleak`).

## What the session produced

1. **"What can we learn" analysis** of `doanalyzerv2` (DO-1..DO-8, AST walker)
   and `roleak` (RO-1..RO-5, samber/ro): findings-UX enrichment ideas
   (root-cause grouping, consistency outliers, type-in-suggestion), precision
   machinery (accessor suppression, structural provider-closure recognition),
   and confirmation that DO-9 *delegates* to our `healthwash.New()`.
2. **PRO/CONTRA on migrating both rule families into samber-linter**, with the
   decisive finding: branching-flow **is** module `github.com/larsartmann/go-design-smells`
   and **depends on samber-linter v0.4.0** (`go.mod:25`; `doanalyzerv2/analyzer_healthwash.go:34`
   imports `pkg/healthwash`). A straight import migration is a **module cycle**;
   only a port (reimplement here, delete there, consume back) is viable.
   Recommendation delivered: port DO-1..DO-8, leave roleak, cheapest de-risk =
   extract shared precision/enrichment layer into samber-linter first.

---

## a) FULLY DONE

- Read the complete `doanalyzerv2` analyzer surface: `analyzer.go`,
  `analyzer_calls.go` (all 470 lines incl. `isAccessorFunction`,
  `isInsideProvideClosure`, `extractInvokeType`), `analyzer_global.go`,
  `analyzer_shutdown.go`, `analyzer_structural.go` (incl. `groupRootCauses`,
  `flagConsistencyOutliers`, composition-root allowlists), `analyzer_healthwash.go`
  (DO-9 delegation), `types.go`, `types_enum.go` (go-enum), `to_findings.go`, `doc.go`.
- Read 10 of 11 recent (Oct 5) precision test files: accessor suppression,
  provider closure, composition-root allowlist, consistency note, generic skip,
  invoke suggestion, registration maps, root-cause grouping, test-file hint,
  precision regression (storbi/BuildFlow/timesheets).
- Read `roleak` `doc.go`, `types.go`, `analyzer.go` lines 1-200 (RO-1, RO-2 in
  full; RO-3 constructor map started).
- Verified the dependency direction both ways via `go.mod` greps: go-design-smells
  → samber-linter v0.4.0; samber-linter does NOT depend on go-design-smells.
- Located how both analyzers are wired into branching-flow (`pkg/analysis/analysis.go:127,131`,
  `pkg/stats/collect_*.go`, `cmd/branching-flow/*_output.go` column specs).
- Checked samber-linter's own state for comparison (`pkg/healthwash/rules.go`
  eval-site precedence, file listing; grep confirmed no Suggestion field in healthwash).
- Delivered both answers with file:line citations and a concrete recommendation
  + offered next actions (prototype root-cause grouping; price the port).

## b) PARTIALLY DONE

- **Verification depth**: all conclusions are from static reading. Neither
  branching-flow's test suites nor the DO-9 delegation analysistest were RUN.
  The claim "DO-9 delegation validates our architecture" rests on the doc
  comment + test file existing, not on a green run.
- **roleak coverage**: RO-3/RO-4/RO-5 implementations (analyzer.go lines 200+)
  unread; rule semantics taken from `doc.go` only.
- **"Findings UX is our weakest area" claim**: evidenced (healthwash emits
  `message` only), but I never inspected `go-finding.Finding`'s data model to
  confirm a suggestion/fix field even exists to carry enrichment.
- **Prior-art check**: branching-flow's `docs/`, `TODO_LIST.md`, AGENTS.md were
  NOT searched for an existing migration decision/ADR. My recommendation could
  be a split brain with something already decided over there. Also: the Oct-5
  test files were written ~2h before this session (12:58-12:59) — likely
  another session's work; I did not read its status report for rationale.

## c) NOT STARTED

- Any code change in either repo (none was requested).
- Port pricing (files/LOC/deps/test-rewrite estimate) — offered, not executed.
- Root-cause-grouping prototype for healthwash.
- AGENTS.md / ecosystem-table update with this session's durable facts
  (dep direction, v0.4.0 consumer pin, DO-9 delegation status, cycle blocker).
- samber-linter latest-tag check (is the consumer's v0.4.0 pin stale vs our tags?).

## d) TOTALLY FUCKED UP

Nothing destructive — the session was read-only and the tree is clean
(`git status` empty). Two **overclaims** to retract/downgrade, though:

1. "DO-9 delegation confirmed working" → should be "delegation code + test
   exist; not executed this session."
2. "branching-flow" vs "go-design-smells" naming presented as a settled fact
   from one `go.mod` line; the rename/module history was never checked, so the
   framing "the branching-flow repo IS go-design-smells" is single-sourced.

## e) WHAT WE SHOULD IMPROVE (about this session's work)

- **Run, don't read**: for a consumer-integration claim, one
  `go test ./pkg/doanalyzerv2/...` in branching-flow costs seconds and upgrades
  the strongest claim from "code exists" to "verified green".
- **Always grep for prior decisions** in the other repo (docs/, TODO_LIST,
  AGENTS.md, docs/status/) before issuing a recommendation — my pro/contra
   could duplicate or contradict an existing ADR.
- **Verify the data model before proposing UX features** (go-finding.Finding
  fields) — otherwise the proposal may be unrepresentable as described.
- **Memory discipline violated**: durable discoveries (cycle blocker, consumer
  pin v0.4.0, Oct-5 precision work in the consumer) should have gone into
  samber-linter's AGENTS.md ecosystem table immediately, per the aggressive
  update protocol. Deferred because the session was analysis-only; that is an
  excuse, not a reason.
- Single-source facts (module identity) should be labeled as such or
  cross-checked (git remote, git log) in the same turn.

## f) NEXT: up to 50 things (ranked, session-scoped)

**Decision & scoping**
1. Decide Q1 below: umbrella (port DO-1..DO-8) vs stay health-washing-only. Everything else keys off this.
2. Search branching-flow docs/TODO_LIST/AGENTS.md + docs/status/ for a prior migration ADR or the Oct-5 session report.
3. If porting: full inventory of what moves (8 DO rules, 11 test files, 3 cmd column specs, enum codegen, severity maps).
4. Price the port: LOC, dependency deltas (astutil/core/finding → go-finding/go-linter-sdk), test-rewrite count.
5. Confirm go-design-smells' visibility (public?) and release cadence; check if inverting the dep (it consumes DO from us) is acceptable to it.
6. Decide roleak disposition: stay in go-design-smells, or become its own samber/ro linter. Do NOT fold into samber-linter.
7. Check samber-linter's max `git tag` vs the consumer's v0.4.0 pin; if stale, note consumer-upgrade nudge (their decision).
8. Verify the branching-flow→go-design-smells module history (git log / remote) to kill the single-source naming fact.

**Verification of this session's claims**
9. Run `go test ./pkg/doanalyzerv2/...` and `./pkg/roleak/...` in branching-flow.
10. Run the DO-9 delegation analysistest specifically (`TestDO9Delegation`).
11. Inspect `go-finding` Finding struct: does it carry suggestion/fix/how-to-fix fields at all?
12. Feasibility check for root-cause grouping in healthwash: can evalSite reports be correlated by resolved service type at emit time (facts/report aggregation point)?
13. Read roleak analyzer.go lines 200-300 (RO-3/4/5 impls) before ever reusing its patterns.
14. Confirm healthwash suppression model vs branching-flow's `//nolint:branching-flow:do` — decide if file-scope ignore comments are worth adopting (we already require reasons; they don't).

**UX enrichment (if pursued)**
15. Prototype type-in-suggestion for HW findings (service type already in message; check go-finding capacity first — see 11).
16. Prototype root-cause grouping: HW-3/HW-4 wrapper finding citing the underlying HW-1 on the same stored type.
17. Prototype consistency outliers: package registers healthchecks one way, one site diverges.
18. Golden-pair fixtures for each enrichment (NotFlagged + StillFlagged), mirroring branching-flow's test discipline.
19. If new fields enter output: check --json/SARIF schema stability and go-output column specs; extend `SupportedOutputFormats` tests if formats change.
20. Keep enrichment OUT of machine formats if it's advisory-only (precedent: --check advisory line rule).

**Precision layer (de-risk path, no cycle)**
21. Diff branching-flow's `isInsideProvideClosure`/`isProviderClosureSignature` against healthwash `wrapper.go`; extract any recognizer we lack.
22. Evaluate accessor-function recognizer applicability to our wrapper/unresolved classes (may shrink the "still invisible" list in AGENTS.md).
23. Run healthwash over the storbi/BuildFlow/timesheets regression corpus as an FP probe (with permission; pseudonymize per ecology rules).
24. Consider structural (signature-based) provider recognition as a second resolution channel in healthwash (composes with wrapper + provider-body channels).
25. Pin each new recognizer with fixtures (`testdata/src/...`) before shipping.

**If the port is greenlit**
26. Choose rule-ID scheme: keep stable `DO_*` output names (consumer suppression compatibility) while living under samber-linter.
27. Re-home DO rules as `pkg/dousage` (or similar) on go/analysis; AST-decidable rules may use a lightweight pass.Analyzer with no types fact dependency.
28. Port the composition-root allowlists (type names, package names, cleanup-field heuristic) as CONFIG, not hardcoded maps, if feasible.
29. Port severity maps into samber-linter's severity model (go-finding severities; check mapping table).
30. Unify suppression story: DO rules must adopt `//samber-linter:allow <rule> <reason>` semantics — breaking change for branching-flow users; needs a migration note.
31. Wire new rules into `plugin/` golangci plugin + extend `plugin_integration_test.go` settings pass-through.
32. Extend cmdguard CLI: rule-family enable/disable flags without breaking the 12-flag surface pinned by `flags_test.go` and `exitcode_test.go`.
33. Update README (the contract): rule table, scope statement, §11 ledger additions.
34. Update `.custom-gcl.yml` version pin policy if plugin surface grows.
35. Re-run `nix run .#lint` + full CI matrix incl. dogfood baseline impact.
36. GOEXPERIMENT=jsonv2 now binds ported code — confirm acceptable to go-design-smells consumers.
37. Plan release: version bump (minor at least), README §12 FIRST, then tag (drift test enforces order).
38. Delete originals in go-design-smells only after it consumes the ported rules (avoid dual-maintenance window).
39. Ecology re-scan (`scripts/ecology-scan.sh`) as post-change regression proof.

**Docs & memory**
40. Update samber-linter AGENTS.md ecosystem table: go-design-smells module identity, dep direction, v0.4.0 consumer pin, DO-9 delegation, cycle blocker.
41. Record the "port, don't import (cycle)" lesson in the project where it lands (only becomes cross-project if the port happens).
42. Add chosen next steps to TODO_LIST.md with status (once Q1 decided).
43. If port: CHANGELOG entry + FEATURES.md PLANNED→DONE tracking.
44. Ask branching-flow side (out of repo) to link the DO-9 mapping table to our README §3 so IDs stay synchronized.

**Housekeeping**
45. Keep this report as the decision record for the migration question; link it from whichever ADR/TODO item survives.
46. Re-verify git status clean at session end (done: clean).
47. No baseline/dogfood impact this session (no code changed) — nothing to re-cut.

## g) Questions I cannot answer myself

1. **Scope decision**: do you WANT samber-linter to become the umbrella
   samber/do linter (port DO-1..DO-8, accept the scope rename), or must it stay
   strictly the health-washing analyzer? This is a product call; both are defensible.
2. **roleak's fate**: should a samber/ro linter eventually exist as its own
   project (like this one), or is it fine living forever inside
   go-design-smells as an internal rule family?
3. **AGENTS.md now or later**: should I write this session's durable facts
   (cycle blocker, v0.4.0 consumer pin, DO-9 delegation) into samber-linter's
   AGENTS.md ecosystem table immediately, or hold until Q1 is decided so the
   entry reflects the outcome rather than the open question?

— Session ended with repo tree clean; awaiting instructions.
