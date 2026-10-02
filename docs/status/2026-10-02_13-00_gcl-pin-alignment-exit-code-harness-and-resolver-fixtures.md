# Status: golangci pin alignment, CLI exit-code harness, resolver fixtures

**Session:** 2026-10-02 ~12:00–13:00 CEST. Executed the actionable top of the
previous session's NEXT queue
(`docs/status/2026-10-02_11-42_cmdguard-cli-adoption-and-stale-fixture-completion.md`
§f): golangci pin alignment (§f.1), flake drift guard (§f.2), the CLI
testing-gap batch (§f.18, 20–23), and the resolver-fixture batch (§f.11–14).

## a) FULLY DONE (verified)

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | golangci-lint pins aligned to **v2.14.0** everywhere (CI action `version:`, `.custom-gcl.yml`); nixpkgs already shipped 2.14.0 — the AGENTS "one version everywhere" policy holds again | both files; `nix eval nixpkgs#golangci-lint.version` = 2.14.0; plugin custom-build integration green on the new pin |
| 2 | Flake drift guard `checks.golangci-version-drift` (nixpkgs ≠ `.custom-gcl.yml` ≠ CI pin → loud failure with policy message) | `flake.nix`; negative-proven by temporarily regressing the pin (failed with the drift message), restored, green |
| 3 | CI lint job step cross-checking the two in-repo pins (CI cannot see nixpkgs) | `.github/workflows/ci.yml` lint job |
| 4 | CLI exit-code contract harness `TestExitCodeContract`: help spellings (`--help`, `-h`, `-help`, trailing, `help`, `completion bash`), `--version`, unknown flag 1, invalid `--output` 2, clean 0 / findings 1 / load failure 2, `--check`→0 (incl. on load failure) — through main's REAL wiring (`newCLI`, extracted for this) with offline scaffold modules | `cmd/samber-linter/exitcode_test.go`; sequential (fd capture + chdir) |
| 5 | `buildDriverOptions` extracted + unit-pinned: pattern fallback, empty-`--baseline` fallback to `driver.DefaultBaselinePath` (§f.18), output mapping, min-confidence | `cmd/samber-linter/options_test.go` |
| 6 | `-help` and `-h` variants no longer manual-smoke-only (§f.20) + `normalizeHelpFlag` exact-rewrite unit test | exitcode_test + TestNormalizeHelpFlag |
| 7 | cobra auto help/completion subcommands pinned invocable, never reaching the linter (§f.22) | harness cases "help subcommand", "completion bash" |
| 8 | `dynamicOutputHelp` ≡ `driver.SupportedOutputFormatNames()` exact ordered equality (§f.23; was one-directional substring) | flags_test.go |
| 9 | **Bug fixed: bare `-help` exited 1** (styled error, unknown shorthand `e`) — the adoption session's smoke table had recorded it as passing; stdlib-flag legacy contract restored via exact-token rewrite | real binary before (exit 1) / after (exit 0, help); TestNormalizeHelpFlag |
| 10 | **Bug fixed: `--check` did not force 0 on load failure** — README says "always exit 0"; driver returned 2 before consulting the flag | driver.go load-error path; TestCheckForcesZeroOnLoadFailure; real binary smoke |
| 11 | Resolver fixtures (§f.11–14): `testdata/src/diverging` (diverging returns stay unresolvable, silent), `foreignprov/lib`+`main` (cross-package provider body invisible, silent), `hwwrap` extension (wrapper × provider-body COMPOSES: HW-4+HW-7 at the wrapper call site), `hw7cross/main` extension (HW-7 cross-package fact × provider-body composes at one site) | all registered in TestGoldenCorpus, green |
| 12 | CHANGELOG [Unreleased] records the harness, guard, pins, and both fixes; AGENTS.md updated (version policy v2.14.0 + guards, `-help` trap, `--check`-on-load-failure, harness pointer, false-negative classes now fixture-pinned); TODO_LIST updated | the three files |
| 13 | Full suite green after all changes (incl. plugin custom-GCL on v2.14.0, 6/6 ok) | `GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go test -count=1 ./...` |

## b) PARTIALLY DONE

| # | Item | State |
| - | ---- | ----- |
| 1 | Final gates | `nix build` / `nix flake check` / dogfood re-run in progress at report time; suite + hermetic lint + drift-guard check already green individually |
| 2 | §f.24 (in-repo `testdata/smoke` module) | judged MOOT: the harness scaffolds offline modules inline (`t.TempDir`), no /tmp or network dependency to remove |

## c) NOT STARTED (deliberately, from §f)

Ecology re-survey (§f.7/15/16), old-vs-new final-binary parity (§f.19),
upstream filings (§f.25–29), README cmdguard naming (§f.30), docs
annotation sweep (§f.17) — recorded in TODO_LIST.md with pointers.

## d) LESSONS

1. **A manual smoke table is testimony, not a contract.** The adoption
   session recorded `-help` as passing; it exited 1 in reality. The harness
   caught it on its first run — that is the entire argument for §f.21.
2. **The README is the contract even against code.** `--check`'s "always
   exit 0" was violated by an early return older than the cmdguard
   adoption; nobody noticed because no test drove a load failure through
   `--check`.
3. **fd-capture tests need explicit release semantics.** The first capture
   helper raced (assert before the drain goroutine scheduled) — intermittent
   empty outputs. `release()` (close write ends, join drainers) made it
   deterministic.

## e) NEXT

1. Push and watch CI on the pin bump + harness (drift-guard step runs in
   the lint job; plugin test exercises custom-GCL 2.14.0).
2. The §f leftovers consolidated in TODO_LIST.md.
3. Release decision for the two behavior fixes (`-help`, `--check`): ship
   with the next batch (v0.4.1) or hold — CHANGELOG [Unreleased] is ready
   either way.

_Point-in-time snapshot 2026-10-02 ~13:00 CEST. Baseline suite green at
session start (6/6) on commit 7d76464 + the uncommitted status-doc
formatter diff (untouched, not mine)._
