# Status Report — go-output Integration Session

**Date:** 2026-09-10 03:00 CEST
**Scope:** This session only — adopting `github.com/larsartmann/go-output` into samber-linter, using `/home/lars/projects/cmdguard` as the reference implementation.
**Verdict:** Feature shipped and verified end-to-end. Process had three self-inflicted failures (two nearly declared false success). One pre-existing repo-wide quality gate is red and remains red.

---

## Session Timeline

1. Question "Are we using go-output?" → answered: **No** (not in go.mod, no references).
2. User: "Maybe we fucking should" + pointed at cmdguard.
3. Researched cmdguard: `output.go` / `cli_output.go` / `cli_errors_json.go` patterns (OutputConfig, ParseFormat, RenderTable, RegisteredTableMarshalFormats, UnsupportedFormatError).
4. Researched go-output: Format enum (16 formats), Table + AddRowChecked, registry dispatch, submodule layout.
5. **Mechanism discovery 1:** format registration is `init()`-based per submodule; the root module registers nothing. cmdguard imports only the root and still gets all 16 formats because `samber-do-auditlog` (its dep) imports the submodules transitively (proved via `go list -deps` + importer loop). samber-linter has no such transit → must import submodules explicitly.
6. **Mechanism discovery 2:** go-output v0.38.0 sibling tags are misaligned — markdown/markup@v0.38.0 compile against `escape@v0.38.0` (`escape.MarkdownCell`) but their published go.mod pins v0.37.0. Build failed with `undefined: escape.MarkdownCell` until escape was forced to v0.38.0 (cmdguard carries the same pin).
7. Implemented `--output <fmt>`: table/csv/tsv/markdown/html/xml/asciidoc. JSON-family and diagram formats deliberately banned (machine formats stay with `--json`/`--sarif` — one shape per consumer).
8. Tests (6 new), lint fixes (err113 → sentinel error, varnamelen renames), README (contract §Quick start + §6), AGENTS.md memory, flake.nix vendorHash.
9. Verified: go build/vet, full suite under CI parity env (`GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go test -count=1 ./...`), CLI smoke tests, dogfood on cmdguard (found a **real HW-2** in cmdguard `pkg/cmdguard/v4/scope.go:427`), hermetic `nix build` ✓, treefmt/format ✓, hermetic lint: **0 hits in files I touched**.

---

## Brutal Self-Review

### What did I forget?

1. **FEATURES.md / TODO_LIST.md / CHANGELOG.md** — I updated README and AGENTS.md but not the rest of the project documentation suite. The `--output` feature is absent from FEATURES.md; no changelog entry exists; follow-up tasks were not harvested into TODO_LIST.md.
2. **Proof for the "lint gate was already red" claim** — I asserted the hermetic lint failure pre-dates this change based on strong evidence (≈30 hits in files/lines I never touched), but I never **proved** it by running the hermetic lint on the pre-change commit (git worktree). Evidence ≠ proof; the distinction should have been stated or closed.
3. **GitHub Actions CI status** — if the hermetic lint (same `.golangci.yml`) is red, the `lint` job on master is probably red too. I never ran `gh run list` to confirm. The claim "CI on master may be red" is unverified.
4. **Plugin build check** — `.custom-gcl.yml` / `plugin/` builds its own module graph. Almost certainly unaffected (plugin imports only the analyzer), but I never verified a custom-golangci build against the new go.sum.
5. **Dogfood of `--output` in CI** — the `dogfood` job doesn't exercise the new flag, so it can rot silently.

### What is something stupid we do anyway?

- **Pipeline-masked exit codes.** Twice this session I ran `cmd | head` / `cmd | grep && echo OK` and read a green banner while the underlying command had failed (`$?` after `head`, `tail` swallowing `nix flake check`'s failure). AGENTS.md _literally documents this exact trap_ ("verify the raw summaries, not the filtered tail") and I still walked into it. Both were caught on re-check, but only by luck of re-verification habit.
- **`go mod tidy` before imports exist** — tidy silently removed the freshly-added dep; wasted a cycle. Correct order is code-first-then-tidy.
- **The `nix hash path vendor/` shortcut** — I flagged it as risky in my own reasoning and used it anyway. It produced a hash that doesn't match the goModules fetcher layout; wasted a full nix build cycle. The canonical fakeHash→`got:` loop is the only correct method.

### What could I have done better?

1. **The broken edit.** A careless find/replace in `output_test.go` deleted an `if` body and left the file syntactically broken until the next edit. The edit tool is exact-match by design; I constructed a wrong `old_string`/`new_string` pair. Whole-function edits belong in `lsp_replace_symbol`, not hand-rolled truncations.
2. **Format-matrix testing.** cmdguard tests all 16 formats in a table-driven loop; I test 3 supported formats + 9 bans. A loop over all 7 supported formats would have cost nothing more.
3. **Checked the daemon before worrying** — I initially feared the auto-commit daemon had committed my broken intermediate test file. Verification showed both commits of that file contain the fixed version (the broken state lived only in the working tree). I should verify-then-worry by default, not worry-then-verify.
4. **Absolute paths in the table** — the dogfood table shows `/home/lars/projects/cmdguard/...` in every row. Not a regression (text mode has the same paths) but the table makes it loud; a relative-path option was worth considering during design instead of after.

### What could I still improve?

- **Exit-code discipline:** never read `$?` after a pipe; run verification commands bare or with explicit status capture. Consider a personal verification wrapper.
- **Docs-suite checklist at feature completion:** FEATURES/TODO_LIST/CHANGELOG alongside README/AGENTS, every time.
- **Baseline proof:** when claiming "pre-existing failure", immediately produce the before-commit run (worktree + hermetic check) instead of arguing from evidence.
- **Dogfood new flags in CI** the same session they ship.

### Did I lie to you?

No. But two claims were softer than they sounded and are now stated precisely: (1) "lint gate already red" is **strong evidence, not proof**; (2) "hermetic lint: 0 hits in my files" is true, while the **check as a whole still exits 1** — do not read my summary as a green lint gate.

### Ghost systems / split brains / scope creep

- **Ghost systems:** none. The output path is wired flag→driver→renderer→tests end-to-end.
- **Split brains (small, accepted):** two error types for "unsupported format" — `output.UnsupportedFormatError` (dispatch layer) vs `driver.ErrUnsupportedOutputFormat` (policy layer). Intentional layering; both documented. `--output csv` + plain-text coverage/summary lines means CSV output is not machine-parseable end-to-end — documented tradeoff, listed as improvement.
- **Removed something useful?** Nothing removed. Default text output untouched.
- **Scope creep?** Held the line on the machine-format ban and the untouched default; resisted adding yaml/toml/diagrams "because we can".

---

## a) FULLY DONE

| #  | Item                                                                                                                  | Evidence                                                                                                  |
| -- | --------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| 1  | go-output v0.38.0 integrated (root + table, delimited, markdown, markup; escape pinned v0.38.0)                       | `go.mod:10-14,34`                                                                                         |
| 2  | `--output <fmt>` end-to-end: flag → ParseOutputFormat validation → findings table render; default text mode untouched | `internal/driver/output.go`, `internal/driver/driver.go:157-166`, `cmd/samber-linter/main.go:23-25,50-56` |
| 3  | 6 new tests, all passing; full suite green under CI-parity env                                                        | `internal/driver/output_test.go`                                                                          |
| 4  | escape sibling-tag misalignment diagnosed, pinned, documented                                                         | `go.mod:34`, AGENTS.md                                                                                    |
| 5  | Format-registration mechanism documented (init()-per-submodule, auditlog transit)                                     | AGENTS.md                                                                                                 |
| 6  | flake.nix vendorHash updated; hermetic `nix build` (incl. full test suite checkPhase) green                           | `flake.nix:29`                                                                                            |
| 7  | treefmt/format checks green for all touched files                                                                     | `nix build .#checks.x86_64-linux.{treefmt,format}` exit 0                                                 |
| 8  | README (contract) §Quick start + §6 CLI surface updated                                                               | README.md                                                                                                 |
| 9  | AGENTS.md memory updated (repo status + 2 new gotcha bullets)                                                         | AGENTS.md                                                                                                 |
| 10 | My lint hits zeroed (err113 → `ErrUnsupportedOutputFormat` sentinel; varnamelen renames)                              | hermetic lint log: 0 hits in touched files                                                                |

## b) PARTIALLY DONE

| # | Item                     | Gap                                                                                                         |
| - | ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| ~~1~~ | ~~Docs-suite update~~ done — FEATURES/TODO_LIST/CHANGELOG updated 2026-09-10 (docs-health audit; CHANGELOG [Unreleased]) | ~~FEATURES.md / TODO_LIST.md / CHANGELOG.md missing the feature + follow-ups~~ |
| 2 | Hermetic lint gate       | My files clean; gate still exits 1 on ~30 pre-existing hits; no fix-or-scope decision made                  |
| 3 | CSV output composability | Coverage/summary lines stay plain text → CSV not end-to-end machine-parseable (documented, not solved)      |
| 4 | Table UX                 | Absolute filesystem paths in Location column; long messages make very wide tables (no wrap/trim check done) |
| ~~5~~ | ~~Plugin compatibility~~ done — `plugin/plugin_integration_test.go` build-and-fire proof `836be4c` | ~~Assumed unaffected; never ran a custom-gcl build~~ |

## c) NOT STARTED

| # | Item                                                                                                       |
| - | ---------------------------------------------------------------------------------------------------------- |
| ~~1~~ | ~~HW-1..HW-6 backport to `branching-flow/pkg/doanalyzerv2` as DO-9 family~~ done — shipped — branching-flow `analyzer_healthwash.go` delegates DO-9a–e |
| ~~2~~ | ~~Upstream conversation with samber/do (transient HealthCheck returning nil is an upstream TODO)~~ done — filed as samber/do#317 + #318 `ae77908` (both OPEN) |
| 3 | `--output` smoke step in `.github/workflows/ci.yml` dogfood job                                            |
| ~~4~~ | ~~Confirm/cite GitHub Actions lint job status on master (`gh run list`)~~ done — 2026-09-10 — all runs red except drift-matrix; GOEXPERIMENT fix `17732a4`, lint debt in TODO_LIST |
| ~~5~~ | ~~Release engineering for the new flag (tag, `go run @latest` smoke — README quick-start advertises @latest)~~ done — tracked — v0.1.1 release line in TODO_LIST (pending user go-ahead) |
| 6 | Runtime-companion metrics export polish (phase 3 stretch beyond current healthaudit)                       |

## d) TOTALLY FUCKED UP

Nothing shipped is broken — the feature works and is verified. What _is_ fucked up:

| # | Item                                                                                                                                                                                                                                                                                                                                                                                                                                             | Status            |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------- |
| ~~1~~ | ~~**Repo-wide hermetic lint gate is red** (pre-existing: varnamelen ×22, wrapcheck ×3, tparallel, unparam, forbidigo, mnd, predeclared, gochecknoglobals, cyclop, nestif, lll across `driver.go`, `main.go`, `healthwash/`, tests). `nix flake check` therefore **cannot serve as a merge gate** — it fails on every tree, clean or not. Pre-dates this session (evidence: hits in untouched files); unproven against the exact pre-change commit.~~ done — still red 2026-09-10 — root causes diagnosed (golangci toolchain mismatch + ~141 findings); tracked as TODO_LIST quality gate | ~~Red, unowned~~ |
| 2 | **Two false-green moments this session** (masked exit codes). Caught, but the process that allowed them is the same one documented in AGENTS.md as a known trap.                                                                                                                                                                                                                                                                                 | Process debt      |
| 3 | **Broken intermediate edit** sat in the working tree (never committed — verified). Cost: one wasted round trip + an apology-grade mistake in exact-match editing.                                                                                                                                                                                                                                                                                | Fixed, historical |

## e) WHAT WE SHOULD IMPROVE

1. **Verification discipline:** bare exit-code checks (`cmd; echo exit=$?`), never `$?` after pipes; explicit `|| echo FAILED` on gated pipelines.
2. **Edit safety:** whole-function edits via `lsp_replace_symbol`; exact-match `edit` only for small, fully-quoted blocks.
3. **Feature docs suite:** README + AGENTS + FEATURES + CHANGELOG + TODO_LIST in one pass at feature completion.
4. **Claim hygiene:** separate "verified", "strong evidence", "assumed" in every report (this report does).
5. **Dogfood-in-CI:** every new flag gets one line in the dogfood job the day it ships.
6. **Upstream-first for ecosystem bugs:** file the escape-pin misalignment and the cmdguard HW-2 finding (with verify-before-filing discipline) instead of only pinning/documenting locally.
7. **Lint policy decision** (see question 1) so `nix flake check` can become a real gate again.

## f) TOP 50 NEXT THINGS (brainstorm, sorted by impact; most are ROADMAP fuel)

**Quality gate & CI**

1. Decide + execute lint-gate policy: fix all ~30 pre-existing hits **or** scope the gate (see question 1)
2. Confirm GitHub Actions `lint` job status on master (`gh run list`) and align expectations
3. Pin CI golangci-lint version to the exact hermetic version (three-tool drift caused confusion)
4. Add `--output markdown` smoke line to ci.yml dogfood job
5. Add `meta.description` to flake apps (kills the recurring flake-check warnings)
6. Gate the auto-commit daemon on `go build` passing (crush hook) so red intermediates can't land
7. Add table-driven test rendering **all 7** supported formats (cmdguard-style matrix)
8. Add `errors.Is(err, ErrUnsupportedOutputFormat)` contract test
9. Test typo case (`--output tabe`) surfaces the full enum list
10. Test `NO_COLOR`/`CI` env behavior of table rendering
11. Test HW-0 / HW-unresolved rows render correctly in table mode
12. Perf sanity: render 10k-finding table (bench)
13. `go vet ./...` + drift-matrix already in CI — add `--output` + `--json` combined-run test (composability contract)

**Product / UX**
14. Relative-path option for Location (`--paths=rel` or trim module root)
15. `--no-summary` (or output-aware summaries) so CSV/markdown are end-to-end parseable
16. Decide direction: keep plain-text default vs cmdguard-style table default (see question 2)
17. ~~Document plugin/golangci path explicitly: `--output` is driver-only by design~~ done (README §6 documents the machine-format ban and driver-only scope)
18. README: rendered example block of `--output markdown`
19. README: exit-code table (usage errors are 2 by flag convention; triage is also 2 — clarify)
20. Long-message wrapping/width check for table renderer; consider Message column truncation
21. `-o` short alias? (consistency question with cmdguard)
22. docs/DOMAIN_LANGUAGE.md: "presentation format vs machine format" vocabulary entry
23. Consider `SAMBER_LINTER_OUTPUT` env for CI ergonomics (YAGNI candidate, keep last)

**Docs & release**
24. ~~FEATURES.md: add `--output` (DONE)~~ done (FEATURES.md lists `--output` under Driver CLI (2026-09-10 audit))
25. ~~CHANGELOG.md: unreleased entry for the flag + escape pin~~ done (CHANGELOG [Unreleased] carries the flag + escape pin (2026-09-10))
26. ~~TODO_LIST.md: harvest this report's actionable items (docs-health HARVEST)~~ done (docs-health pass 2026-09-10)
27. Release: tag v0.1.1, `go run ...@latest` smoke (go-release flow)
28. AGENTS.md: document "never `nix hash path vendor/` for vendorHash — use fakeHash loop"
29. ~~Annotate this report's claims when later revisited (docs-health ANNOTATE mode)~~ done (docs-health pass 2026-09-10)

**Ecosystem / upstream**
30. Verify + file go-output issue: markdown/markup@v0.38.0 mis-pin escape v0.37.0 (verify-before-filing)
31. Verify + file/report the real HW-2 in cmdguard `scope.go:427` (user owns cmdguard)
32. Watch go-output releases; bump when sibling tags realign; re-test matrix
33. Propose go-output CI check that fails when sibling module pins drift from the release tag
34. ~~HW-* backport to branching-flow doanalyzerv2 (DO-9 family) — P3 per AGENTS.md~~ done (shipped — branching-flow delegation DO-9a–e (`analyzer_healthwash.go`))
35. ~~Upstream samber/do conversation: transient HealthCheck returns nil (upstream TODO)~~ done (filed samber/do#317 + #318 `ae77908`)
36. ~~Dogfood samber-linter on branching-flow; collect real-world finding quality data~~ done (clean (verified in dogfood, docs/status/2026-09-10_01-17 §a.23))
37. ~~Verify `.custom-gcl.yml` plugin build against the new go.sum~~ done (`plugin/plugin_integration_test.go` `836be4c`)
38. Dep sweep: check go-finding / go-linter-sdk / go-atomic-write for newer versions (go-ecosystem-upgrade flow)
39. Drift-matrix: prepare samber/do v2.2.x extension point for when it releases

**Architecture / robustness (lower priority)**
40. Multi-module (go.work) target test for packages.Load path
41. Consider extracting `formatNames`/helpers into a tiny internal/outputapi if driver grows more output code (YAGNI today)
42. Revisit `reportCoverage` cyclomatic complexity 21 (pre-existing; split baseline/threshold paths)
43. Revisit `gochecknoglobals` on `VerifiedDover`/`ruleMetaByRule` (pre-existing; options pattern or keep-with-nolint)
44. Exit-code contract: consider distinct exit code for usage errors vs triage (contract change — needs care)
45. Fuzz `ParseOutputFormat` inputs (cheap, low value — enum space is tiny)
46. Add `--output` to `nix run .#test` dogfood? (covered by CI item 4 — pick one place)
47. Table renderer: test CJK/unicode message width handling (go-output runewidth dep exists)
48. SARIF export test hardening: validate against SARIF 2.1 schema in CI
49. Consider `--output` + `--strict` interplay test (HW-unresolved appears in table)
50. Write the one-paragraph "why machine formats are banned from --output" ADR in docs/adr/

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Lint-gate policy:** Should I fix all ~30 pre-existing lint hits (hours of mechanical churn: renames, globals restructuring, complexity splits) so `nix flake check` becomes a true green gate, or do we scope the gate (e.g. new-code-only / config exclusions) and leave legacy as-is? This decides whether the repo's quality gate is real today.
2. **Default output direction:** Is plain `file:line:col` text the permanent default (CI-log compatibility), or is the endgame a cmdguard-style styled table default with text opt-out? This determines how much polish (colors, width handling, relative paths) `--output table` deserves.
3. ~~**Upstream & release appetite:** Should I (a) verify + file the cmdguard HW-2 finding upstream, (b) file the go-output escape-pin issue upstream, and (c) cut a v0.1.1 release so the README's `@latest` quick start serves the new flag — now, or after more bake time?~~ done (partially — #317/#318 filed `ae77908`; the v0.1.1 release call remains open (TODO_LIST))

---

_Point-in-time snapshot 2026-09-10 03:00 CEST. Verification commands this session: `go build ./...`, `go vet ./...`, `GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go test -count=1 ./...`, `golangci-lint run`, `nix build`, `nix build .#checks.x86_64-linux.{build,treefmt,format,lint}`, CLI smoke + cmdguard dogfood._
