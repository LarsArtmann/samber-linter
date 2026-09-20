# Ecology triage queue — pseudonymous (2026-09-10, refreshed 2026-09-20)

Ecology scan of all local samber/do v2 consumers. Real project names live
ONLY in a local keyfile outside any repository
(`~/backups/ecology/keyfile.json`); this document is safe to commit.

[2026-09-10 docs-health] Execution is tracked in `TODO_LIST.md`
(samber-do-auditlog + rank-1 standard-bug-tracking-schema); CV self-fixed
post-scan (baseline reads 10% vs the 8% committed).

[2026-09-20 refresh] Numbers below now come from
`docs/ecology/2026-09-20-scan.txt`, which supersedes BOTH 2026-09-16 scans
(the mid-scan driver fixes — per-module go.work scanning and baseline
validation — landed in v0.2.2 and changed the counts). Discovery widened from
45 to 72 projects, and **32 of them now LOAD_ERROR because their go.mod or
go.work requires go ≥ 1.27.1 while the scan ran under GOTOOLCHAIN=local
go1.26.7** — an environmental wave, never readable as clean. Load-error
projects drop out of this queue until the scanner runs on a 1.27 toolchain
(see the AGENTS.md survey-traps note for the exact fix).

Scoring: `unprotected = registered − checked` (services whose sweep row is an
unconditional green "pass"). Triage states per finding: **fix** · **suppress
with reason** · **accept into baseline**. Re-scan after each batch; the
baseline ratchet only moves down.

## Priority queue (2026-09-20, findings-bearing projects only)

| rank | project | registered | checked | unprotected | findings | HW-1 | HW-2 | HW-3 | HW-4 | first action                             |
| ---- | ------- | ---------- | ------- | ----------- | -------- | ---- | ---- | ---- | ---- | ---------------------------------------- |
| 1    | p-c49f  | 19         | 0       | 19          | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 2    | p-0a41  | 20         | 3       | 17         | 10       | 3    | 7    | 0    | 0    | add ctx variant + implement checks       |
| 3    | p-1507  | 7          | 0       | 7          | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 4    | p-6c47  | 6          | 0       | 6          | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 5    | p-1fba  | 5          | 0       | 5          | 2        | 2    | 0    | 0    | 0    | implement checks / fix registration kind |
| 6    | p-b4bd  | 6          | 1       | 5          | 1        | 0    | 0    | 0    | 1    | HW-4: eager-or-suppress per service      |
| 7    | p-a2d9  | 5          | 1       | 4          | 2        | 0    | 1    | 0    | 1    | add ctx variant                          |
| 8    | p-46ae  | 3          | 1       | 2          | 1        | 0    | 1    | 0    | 0    | add ctx variant                          |

## What changed since 2026-09-10 (post-v0.2.2 driver fixes)

- Both former top offenders self-fixed or dropped out: the 2026-09-10 rank-1
  project now reports 0 findings at 61 registered (59 still unprotected —
  services with no lifecycle interface at all are outside rule scope by
  design), and the rank-2 project reports 0 findings at 31 registered with
  checked up from 5 to 10.
- Six of the fifteen 2026-09-10 queue rows are in the load-error wave and
  left the queue (including the auditlog offender — its committed v2
  baseline, not this scan, is its current floor).
- p-0a41 enters the queue as the largest open batch (10 findings); the
  remaining rows shrank (fix rounds landed between scans).
- HW-7/HW-8: zero findings in the wild across all 40 analyzed projects.

## Per-rule playbook

| Rule | Meaning                          | Systematic fix                                                                                                           |
| ---- | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| HW-1 | Shutdowner without Healthchecker | Implement `HealthCheck(context.Context) error` (ping what you hold); third-party types → wrap, else suppress with reason |
| HW-2 | Bare `HealthCheck()`             | Add the ctx variant (mechanical); a hung check cannot be cancelled                                                       |
| HW-3 | Transient with a check           | The check can never run: register as singleton or delete the dead code                                                   |
| HW-4 | Lazy + check                     | Boot-critical → register eagerly; otherwise suppress with reason                                                         |
| HW-0 | Suppression without reason       | Add the reason text                                                                                                      |

## Process

1. Per project: `samber-linter --set-baseline ./...` → commit baseline (floor = today).
2. Fix/suppress findings top-down per the queue.
3. Re-run; commit the improved baseline. The floor never drops.
4. Add the analyzer to the project's CI (plugin or `go run`).

## Aggregate (names stripped, safe to publish)

2026-09-20: 72 discovered · 40 analyzed · 32 load errors (go ≥ 1.27.1 wave) ·
8 with findings · 19 findings total (HW-1 ×8, HW-2 ×9, HW-3 ×0, HW-4 ×2,
HW-5 ×0, HW-7 ×0, HW-8 ×0). Not directly comparable to 2026-09-10 (43
analyzed, 66 findings): the driver fixed two false-green/false-negative
defects in between, discovery widened, and a third of the population is
temporarily unloadable.

2026-09-10 baseline for reference: 43 analyzed · 30 clean · 12 with findings ·
66 findings (HW-1 ×30, HW-2 ×11, HW-3 ×1, HW-4 ×24, HW-5 ×0) · HW-5 never
fired in the wild.
