# AGENTS.md — samber-linter

## Repo status: implemented v0.1.0 (updated 2026-09-10)

HW-1..HW-6, driver, golangci plugin, and runtime companion are implemented
(`cmd/`, `internal/`, `pkg/`, `plugin/`), with CI in `.github/workflows/ci.yml`
(test, drift-matrix, lint, dogfood, upstream-snippets). `--output` (go-output
findings table: table/csv/tsv/markdown/html/xml/asciidoc) added 2026-09-10;
`--json`/`--sarif` stay the only machine formats. The phasing section below is
historical.

**Quality-gate reality (2026-09-10):** every CI run to date is red —
`test`/`dogfood` failed on missing `GOEXPERIMENT=jsonv2` (fixed in the
workflow same day, unverified until the next push), and the `lint` job fails
twice over (golangci-lint-action's `latest` binary is built with go1.24 <
go.mod 1.26.7, plus ~141 pre-existing findings — see TODO_LIST). Only
`drift-matrix` has ever been green. Do not claim "CI green" in any doc until
a run proves it.

**`README.md` is the contract.** Read it fully before writing any code. Its
claims carry a verification ledger (§11) with file:line pins into
`samber/do v2.1.0` source; treat it as ground truth, not marketing copy.

## What this is

A static analyzer that detects **health-washing** in `samber/do` v2 DI
containers: services that render green `pass` on health dashboards but cannot
actually fail. Analyzer package name: `healthwash`. Rule IDs: `HW-1` … `HW-6`
(README §3 has the full rule table with severities).

## Binding implementation constraints (easy to get wrong)

These come straight from the spec; violating any of them invalidates the analyzer:

- **Type-checking based, not AST-based.** Must build on
  `golang.org/x/tools/go/analysis` — interface satisfaction cannot be decided
  from the AST alone (README §4).
- **Registration matching by package path, not identifier text** — the analyzer
  must be resilient to dot-imports and renames of `samber/do`.
- **Type resolution rules** (README §4 step 2): `Provide*` resolves the provider
  closure's return type (drop error return); `ProvideValue*` resolves the value
  argument's type. For interface-typed registrations, analyze the **concrete
  returned type** — the runtime sweep asserts the stored instance, not the
  registration interface. Unresolvable registrations are silent by default,
  `HW-unresolved` only under `--strict`.
- **Compute method sets for `T` and `*T` separately** — the pointer-receiver
  trap (HW-5) exists precisely because the sweep type-asserts the stored value.
- **Findings attribute to the registration call site**, with the service type
  name in the message — that is where the fix lands.
- **Suppressions require a reason**:
  `//samber-linter:allow hw-1 <reason>`. A suppression without reason text is
  itself a finding (`hw-suppression-reason-missing`).
- **HW-6 is a ratchet, not a lint**: coverage ratio against a committed baseline
  file; `--set-baseline` locks gains. Modeled on CV's `any-count` ratchet.
- **samber/do v1 is out of scope** (README §10 non-goals).

## Driver contract (v0.1.1)

- Exit codes: `0` clean, `1` findings/gate failure, `2` load failure or
  triage-only (`--check` advisory forces `0` in every case).
- Empty `--output` value means plain-text default — `ParseOutputFormat` must
  accept the zero value (regression: a parallel session's `--output` flag
  broke every plain invocation; guarded by `output_test.go`).
- Allowlist entries with an empty `pathPattern` apply project-wide (not a
  silent no-op); entries with no rules warn once and stay inert.
- Suppression directives are honored anywhere inside a multi-line
  registration call; orphaned malformed directives surface as `HW-0`.
- go.work consumers are analyzed with the pattern `all`, not `./...` —
  `./...` silently skips workspace roots' sibling modules.
- golangci-lint integration requires the custom build
  (`.custom-gcl.yml` + `plugin/plugin_integration_test.go` locks registration
  and settings pass-through); a stock golangci-lint cannot load the plugin.

## Upstream engagement

- [samber/do#317](https://github.com/samber/do/issues/317) (transient
  healthcheck never dispatched, sentinel option) and
  [#318](https://github.com/samber/do/issues/318) (explicit sweep outcome
  states) filed 2026-09-10 under the user's account; watch for responses,
  do not dump unsolicited design notes.
- Upstream text rules: first person, concise, concrete numbers, real links,
  no offer-speak ("do not suggest. Just explain the problem!"), AI
  disclaimer in a small `<sub>` footer (`GLM-5.3-Flash via Crush`).
- Snippet gate: every Go block in `docs/upstream/*.md` is compiled and run
  verbatim by `scripts/check-upstream-snippets.sh` (CI job
  `upstream-snippets`). Blocks quoting upstream source must be marked
  ` ```go snippet-skip `; unclassified blocks fail the check.
- Verification lessons (encoded after two incidents): execute snippets
  verbatim before filing; print whole structures, never map-index lookups
  (a missing key and a nil value print identically — `res["x"] == nil`
  cannot distinguish missing from present).

## Mechanism facts are version-pinned

Every behavioral claim about samber/do (non-implementers return nil; lazy
unbuilt returns nil; transient healthcheck is an upstream TODO returning nil;
wrapper asserts stored instance) is pinned to **samber/do v2.1.0** with
file:line references (README §2, §11). Consequences:

- Pin the analyzed samber/do version in CI and re-run the mechanism assertions
  against new releases — upstream drift must fail the build loudly, not
  silently invalidate rules.
- If rules misfire, first check whether samber/do changed underneath you.

## Testing discipline (required before shipping P0)

- Golden-case corpus from the CV production incident (README §9 table): each
  fixture has an expected outcome (e.g. `graphrag.Store` → HW-1; handler struct
  with no lifecycle interfaces → clean).
- **Discrimination proof**: every golden case must be shown to **fail** on a
  mutant analyzer (rule inverted/removed) in a **scratch copy** — never by
  mutating the shared tree.
- Test build must pass before tests run.

## Phasing

P0 = analyzer skeleton + HW-1 + HW-5. P1 = HW-2/3/4. P2 = HW-6 ratchet +
`--json` + baseline file. P3 = runtime companion (`healthaudit` metrics),
doanalyzerv2 backport as DO-9 family, upstream conversation. See README §9.

## Build/task automation

`flake.nix` uses the `go-standard` module from `github.com/LarsArtmann/go-nix-helpers`
(3 inputs: nixpkgs, flake-parts, go-nix-helpers). Commands: `nix build`,
`nix flake check` (build + full test suite + treefmt + hermetic golangci-lint),
`nix run .#test`, `nix run .#lint`, `nix run .#fmt`. Never create a Makefile
or justfile.

Non-obvious, easy to break:

- **vendor/ was removed 2026-09-10** — build fetches via proxy.golang.org with
  a pinned `vendorHash`. All four `larsartmann/*` deps (go-atomic-write,
  go-finding, go-linter-sdk, go-error-family) are **public** (verified via
  GitHub 2026-09-10), so no `deps`/`GOPRIVATE`/mkPreparedSource wiring exists
  or should be added unless a genuinely private dep appears.
- **`subPackages = [ "./cmd/samber-linter" ]`** — the repo root has no Go
  files; the module default `"."` fails the build.
- **`extraBuildAttrs.checkPhase` overrides buildGoModule's default**, which
  only tests built subPackages (cmd has no test files) — without the override
  the golden corpus silently stops gating `nix build`.
- **`apps.test` is overridden with `lib.mkForce`** — the module default runs
  `go test -race`, which needs a C compiler and no GOEXPERIMENT; ours matches
  CI (`GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go test -count=1 ./...`).
- **GOEXPERIMENT=jsonv2 is a HARD build requirement** since go-finding v1.9.2
  started importing `encoding/json/v2` (earlier claim "nothing imports it"
  is obsolete). Without the experiment the toolchain excludes those files and
  every package build fails — this is exactly what red-lined all 2026-09-10
  CI runs; the workflow now sets `env: GOEXPERIMENT: jsonv2`. Any new Go
  entry point (scripts, snippet builds, future CI jobs) must inherit it.
- **nixpkgs `go_1_26` is exactly 1.26.7 = go.mod floor** (verified 2026-09-10).
  Bumping the go.mod floor past nixpkgs' toolchain requires
  `goTarballVersion`/`goTarballHash` in go-standard.
- **go-output v0.38.0 sibling tags are misaligned**: markdown/markup@v0.38.0
  compile against `escape@v0.38.0` (`escape.MarkdownCell`) but their published
  go.mod pins v0.37.0. go.mod must keep the explicit
  `go-output/escape v0.38.0` require; `go mod tidy` alone re-breaks the build
  with `undefined: escape.MarkdownCell`. cmdguard carries the same pin.
- **go-output format registration is init()-based per submodule** — the root
  module registers nothing. cmdguard imports only the root and still gets all
  formats because samber-do-auditlog (its dep) imports the submodules
  transitively. samber-linter has no such transit, so
  `internal/driver/output.go` blank-imports table/delimited/markdown/markup;
  adding a format = add the blank import + extend the SupportedOutputFormats
  tests (json/yaml/toml/jsonl and diagram formats are deliberately banned
  there — one shape per consumer, machine formats stay with --json/--sarif).
- Formatting: treefmt (gofumpt + goimports + nixfmt) for go/nix, dprint for
  json/yaml/markdown — the two tools own disjoint file sets.

## Ecosystem references (local, on this machine)

| Artifact                                                        | Relevance                                                                     |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `~/projects/branching-flow/pkg/doanalyzerv2`                    | DO-1..DO-8 usage-shape rules + DO-9a–e delegating to this repo's healthwash  |
| `samber-do-auditlog`                                            | Wrapping pattern the runtime companion is modeled on; biggest ecology offender (7×HW-1 + 5×HW-2 + 1×HW-3)                                     |
| `~/.config/crush/skills/samber-do-best-practices/SKILL.md` §6.3 | The skill rule this linter mechanizes                                         |
| `samber/do v2.1.0` module cache                                 | Source of all mechanism pins in README §2                                     |
