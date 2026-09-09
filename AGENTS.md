# AGENTS.md — samber-linter

## Repo status: pre-implementation (verified 2026-09-09)

This repository contains **only `README.md`** — a complete, verified design spec.
No `go.mod`, no source code, no `flake.nix`, no CI, no tests exist yet. If you
see Go source here, it was added after this file was written; re-verify the
status claims below against the tree.

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

None exists yet. When scaffolding: this is a LarsArtmann project — build and
task automation belongs in **`flake.nix`** (`nix build`, `nix flake check`,
`nix run .#test`, `nix run .#lint`). Never create a Makefile or justfile.

## Ecosystem references (local, on this machine)

| Artifact | Relevance |
| -------- | --------- |
| `~/projects/branching-flow/pkg/doanalyzerv2` | DO-1..DO-8 usage-shape rules; HW-* backports there as DO-9 family once stable |
| `samber-do-auditlog` | Wrapping pattern the phase-3 runtime companion is modeled on |
| `~/.config/crush/skills/samber-do-best-practices/SKILL.md` §6.3 | The skill rule this linter mechanizes |
| `samber/do v2.1.0` module cache | Source of all mechanism pins in README §2 |
