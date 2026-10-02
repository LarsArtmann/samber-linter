# Status: honest retrospective of the 2026-10-02 pin-alignment / harness / fixtures session

**Session:** 2026-10-02 ~11:55–13:51 CEST. Followed the
`2026-10-02_11-42` report's NEXT queue (§f items 1, 2, 11–14, 18, 20–23;
judged §f.24 moot). This report is written AFTER the work, at the user's
request, as a self-critical audit — not a marketing summary.

## a) FULLY DONE (verified this session)

| # | Item | Evidence |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- ------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | golangci-lint pins aligned to v2.14.0 (CI action + `.custom-gcl.yml`); nixpkgs already 2.14.0 — AGENTS "one version everywhere" holds again | `nix eval nixpkgs#golangci-lint.version` = 2.14.0; plugin custom-build integration green on the new pin (suite, twice) |
| 2 | Flake drift guard `checks.golangci-version-drift`: nixpkgs ≠ `.custom-gcl.yml` ≠ CI pin fails loudly with the policy message; negative-proven by temporarily regressing the pin (failed with the message), restored, green | `flake.nix`; log captured in session |
| 3 | CI lint job step cross-checking the two in-repo pins (CI runners cannot see nixpkgs) | `.github/workflows/ci.yml` lint job — NOTE: never executed (unpushed), see d)1 |
| 4 | CLI exit-code harness `TestExitCodeContract` (14 cases): help spellings incl. trailing + `help` subcommand + `completion bash`, `--version`, unknown flag 1, invalid `--output` 2, clean 0 / findings 1 / load failure 2 / `--check`→0 | `cmd/samber-linter/exitcode_test.go`; runs main's own wiring via extracted `newCLI`/`buildDriverOptions` against offline scaffold modules |
| 5 | Bug fix: bare `-help` exited 1 (pflag shorthand-cluster trap); now rewritten to `--help` (exact standalone token only), exits 0 with help | real binary before (exit 1, ERROR) / after (exit 0); `TestNormalizeHelpFlag` |
| 6 | Bug fix: `--check` returned 2 on load failure, violating README's "always exit 0"; now 0 with the advisory note on stderr | `internal/driver/driver.go` load-error path; `TestCheckForcesZeroOnLoadFailure`; real binary smoke |
| 7 | `TestBuildDriverOptions`: pattern fallback, empty-`--baseline` → `driver.DefaultBaselinePath`, output mapping + error, min-confidence, version (§f.18) | `cmd/samber-linter/options_test.go` |
| 8 | `--output` help ≡ `driver.SupportedOutputFormatNames()` exact ordered equality (§f.23; was one-directional substring) | `flags_test.go` |
| 9 | Resolver fixtures (§f.11–14): `testdata/src/diverging` (diverging returns silent), `foreignprov/lib`+`main` (cross-package provider body invisible), `hwwrap` extension (wrapper × provider-body COMPOSES: HW-4+HW-7), `hw7cross/main` extension (HW-7 fact × provider-body composes) | all in `TestGoldenCorpus`, green |
| 10 | Docs: CHANGELOG `[Unreleased]` (harness, guards, both fixes), AGENTS.md (version policy v2.14.0 + guards, `-help` trap, `--check`-on-load-failure, harness pointer, fixture-pinned false-negative classes), TODO_LIST.md (consolidated leftovers), session report `2026-10-02_13-00` | the four files |
| 11 | Final gates: full suite 6/6 ok, `nix build` green, `nix flake check` ALL checks passed (dprint, treefmt, hermetic lint 0 issues, drift guard), dogfood `./...` clean exit 0 | command outputs in session; flake-check exit 0 |

## b) PARTIALLY DONE

| # | Item                                      | State                                                                                                                                                                                                                                            |
| - | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | CI validation of everything workflow-side | local equivalents green; the actual workflow (new pin step + step) never ran — repo not pushed (policy: never push unprompted)                                                                                                                   |
| 2 | `-help` legacy-contract verification      | the fix works and is tested, but I never built the PRE-cmdguard binary (365bb4a) to prove the old CLI accepted `-help`; the "regression" framing rests on the previous session's (demonstrably unreliable) smoke table + stdlib-flag conventions |
| 3 | Harness output assertions                 | exit codes pinned everywhere; message text pinned only for 4 of 14 cases (help, version, findings); the unknown-flag and invalid-output styled errors have no content assertions                                                                 |

## c) NOT STARTED (deliberately deferred, recorded in TODO_LIST.md)

1. §f.7/15/16: ecology re-survey (`scripts/ecology-scan.sh`) as the
   post-resolver-change regression proof — needs the go ≥ 1.27.1 scanner
   toolchain dance; heavy.
2. §f.19: old-vs-new binary parity re-run on a findings-bearing module for
   the FINAL binary.
3. §f.25–29: upstream filings (tagalign order disagreement, cmdguard
   feedback, webphone follow-up) — external actions under the user's
   accounts.
4. §f.30: README §5/§11 naming cmdguard as the CLI framework — README is
   the contract; that is a user-level editorial decision.
5. §f.35: `nix flake check --all-systems` (aarch64 unverified).
6. §f.36: treefmt vs `golangci fmt` split (golines/tagalign only in the
   latter) — I hit this personally (see d)4) and still did not document or
   fix the split.

## d) TOTALLY FUCKED UP (defects and missteps of THIS session)

1. **The CI workflow edits are unexecuted.** I wrote a new lint-job step and
   moved the pin but never validated the YAML (no actionlint/yamllint, no
   push). A typo ships red CI on next push. The whole point of item 3 in §a
   is enforcement, and its own correctness is unenforced.
2. **My first-pass code failed hermetic lint with 10 findings** (wsl ×2,
   golines, lll, paralleltest ×2, predeclared ×2, thelper, wrapcheck) and it
   took four fix iterations plus a formatter run. The LSP had surfaced some
   of these live; I dismissed the project-warning wall as fixture noise
   (correct for testdata, wrong for my cmd files). Lesson not yet encoded:
   run `nix run .#lint` after each file batch, not once at the end.
3. **Introduced a comment-vs-code lie.** `TestNormalizeHelpFlag`'s doc says
   the rewrite applies "everywhere it appears", but `normalizeHelpFlag`
   returns after the FIRST `-help` token; a second `-help` stays unrewritten.
   Nonsense input, lying doc — both wrong. Unfixed at report time (user
   ordered report-then-wait).
4. **Repeated the treefmt/golines trap from §f.36 instead of fixing it.**
   `nix run .#fmt` reported "0 changed" while golines still wanted changes;
   I hand-wrapped lines twice before running the actual tool
   (`golangci-lint fmt`). The known split-brain cost me three iterations.
5. **fd leak in the capture helper.** `capturedStd.release()` closes the
   write ends but never the read ends; each subtest leaks two fds until
   process exit. Harmless at this scale, sloppy forever after.
6. **User-visible error message changed without flagging it.** The wrapcheck
   fix prefixed the invalid `--output` error with `parsing --output "…":` —
   byte-parity with the v0.4.0 binary for that path is now broken, and the
   CHANGELOG entry does not mention it. The harness never asserted the old
   text, so nothing caught the delta.
7. **Report-before-gates process smell.** I wrote the 13-00 status report
   claiming partial green, then the gates went red (dprint + the 10 lint
   findings), then I patched the report. Point-in-time reports should be cut
   after gates, or clearly marked pre-gate.
8. **Wrote two status reports for one session** (13-00 mid-session, this one
   now) — the 13-00 one is now a mutated hybrid of snapshot and final state.

## e) WHAT WE SHOULD IMPROVE (systemic, from this session's evidence)

1. **Pin per-batch lint in the working loop** (d)2): `nix run .#lint` after
   each new/edited Go file batch. The end-of-session single lint pass is how
   10 findings accumulate invisibly.
2. **Kill the two-formatter split (§f.36)**: add golines/tagalign to treefmt
   or script a `fmt-all` app that runs treefmt + `golangci fmt`; the current
   state taxes every session that writes Go.
3. **Subprocess-level CLI tests for the true entry path**: the harness calls
   `ExecuteWithArgs(normalizeHelpFlag(args))`, mirroring main — but if main
   ever drops the `normalizeHelpFlag` call, tests stay green. A tiny
   build-and-run test of the real binary pins main() itself.
4. **Workflow validation**: run actionlint (or at least a YAML parse) on
   `.github/workflows/ci.yml` before commit; it currently has zero local
   gate.
5. **Assert message text, not just exit codes**, for the misuse paths — the
   styled errors are part of the UX contract (fang rendering).
6. **Deduplicate the baseline fallback**: `buildDriverOptions` and
   `driver.Run` both map empty → `DefaultBaselinePath`. Harmless belt-and-
   suspenders today, but two owners of one default is how drift starts.
7. **FEATURES.md not touched** — the harness, drift guard, and two fixes
   belong in the feature inventory; I forgot the file exists.

## f) NEXT (ordered by impact)

1. Fix `normalizeHelpFlag` doc-vs-code mismatch (rewrite-all or fix comment).
2. Close the read-end fds in `capturedStd.release()`.
3. Build the 365bb4a (pre-cmdguard) binary; verify `-help` behavior there;
   correct the CHANGELOG framing if the old CLI also rejected it.
4. Decide + document the invalid-`--output` message delta (keep prefix or
   restore parity + `//nolint:wrapcheck` with reason).
5. Validate `.github/workflows/ci.yml` (actionlint/yaml parse); push and
   watch the first CI run on v2.14.0 + the new pin-consistency step.
6. Subprocess test pinning main()'s arg path (normalizeHelpFlag actually
   applied by the real binary).
7. Add message-text assertions for unknown-flag and invalid-output errors.
8. Add FEATURES.md entries (exit-code harness, drift guard, `-help` fix,
   `--check`-on-load-failure fix).
9. Extend `TestBuildDriverOptions` to the untested mapping fields (Strict,
   JSON, SARIF, Check, SetBaseline, ConfigPath, DisableRules).
10. Strict-mode variant of `foreignprov` (HW-unresolved for cross-package
    providers under `--strict`).
11. Encode the per-batch lint rule into AGENTS.md working discipline.
12. Resolve §f.36 (treefmt vs golines/tagalign) structurally.
13. Ecology re-survey post-resolver-change (§f.7) — the deferred regression
    proof.
14. Old-vs-new final-binary parity on a findings module (§f.19).
15. `nix flake check --all-systems` (aarch64).
16. Upstream tagalign filing (§f.25, verify-before-filing first).
17. cmdguard feedback items (§f.27/28).
18. Webphone follow-up (§f.29).
19. README §5/§11 cmdguard naming decision (§f.30 — user).
20. Release decision: v0.4.1 with the two CLI fixes (see g)1).
21. Annotate 2026-09-20 reports touched by the resolver change (§f.17).
22. Review 365bb4a's `driver.go` +61 lines (§f.10) — still unread by me.
23. Re-survey `samber-do-auditlog` row under the new resolver (§f.15).
24. Two-sessions-one-tree protocol into AGENTS.md (§f.37 — see g)3).
25. Promote the parity loop to `scripts/cli-parity.sh` (§f.38).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release timing:** cut v0.4.1 now with the two behavior fixes
   (`-help` exit 0, `--check`-forces-0 on load failure) plus the drift
   guard, or hold and batch with the resolver-era leftovers? Consumers gate
   on exit codes; the fixes make the tool match its README, but a release is
   a fleet-wide event I should not trigger unilaterally.
2. **Error-message parity policy:** for the invalid `--output` path, do you
   want byte-parity with the v0.4.0 binary preserved (revert my wrapcheck
   prefix, add a reasoned nolint), or is the wrapped, more actionable
   message the better contract going forward?
3. **Is another agent session active on this tree right now?** I observed
   only the auto-commit daemon, but I cannot see other sessions. If one is
   mid-flight on the resolver family, my fixture shapes (`diverging`,
   `foreignprov`, the hwwrap/hw7cross extensions) may collide with its next
   steps — same question the 11-42 session asked, now about MY shapes.

_Point-in-time snapshot 2026-10-02 13:51 CEST. All local gates green at
report time (suite 6/6, `nix build`, `nix flake check` incl. hermetic lint +
drift guard, dogfood clean); CI unvalidated (unpushed)._
