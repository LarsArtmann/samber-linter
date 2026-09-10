# Ecology triage queue — pseudonymous (2026-09-10)

First ecology scan of all local samber/do v2 consumers (43 analyzed, 2 missing).
Real project names live ONLY in a local keyfile outside any repository
(`~/backups/ecology/keyfile.json`); this document is safe to commit.

[2026-09-10 docs-health] The queue below is the LIVE triage backlog — unmarked rows are open
by design. Execution is tracked in `TODO_LIST.md` (samber-do-auditlog + rank-1
standard-bug-tracking-schema); CV self-fixed post-scan (baseline reads 10% vs the 8% committed).

Scoring: `unprotected = registered − checked` (services whose sweep row is an
unconditional green "pass"). Triage states per finding: **fix** · **suppress
with reason** · **accept into baseline**. Re-scan after each batch; the
baseline ratchet only moves down.

## Priority queue

| rank | project | registered | checked | unprotected | findings | HW-1 | HW-2 | HW-3 | HW-4 | first action                             |
| ---- | ------- | ---------- | ------- | ----------- | -------- | ---- | ---- | ---- | ---- | ---------------------------------------- |
| 1    | p-4766  | 63         | 0       | 63          | 4        | 4    | 0    | 0    | 0    | implement checks / fix registration kind |
| 2    | p-a32b  | 61         | 5       | 56          | 9        | 7    | 0    | 0    | 2    | implement checks / fix registration kind |
| 3    | p-c49f  | 19         | 0       | 19          | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 4    | p-d9a9  | 20         | 5       | 15          | 18       | 7    | 5    | 1    | 5    | implement checks / fix registration kind |
| 5    | p-b344  | 15         | 1       | 14          | 2        | 2    | 0    | 0    | 0    | implement checks / fix registration kind |
| 6    | p-6d7b  | 16         | 3       | 13          | 3        | 0    | 0    | 0    | 3    | HW-4: eager-or-suppress per service      |
| 7    | p-b4bd  | 10         | 1       | 9           | 1        | 0    | 0    | 0    | 1    | HW-4: eager-or-suppress per service      |
| 8    | p-6c47  | 8          | 0       | 8           | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 9    | p-1507  | 7          | 0       | 7           | 1        | 1    | 0    | 0    | 0    | implement checks / fix registration kind |
| 10   | p-57ed  | 13         | 8       | 5           | 8        | 5    | 0    | 0    | 3    | implement checks / fix registration kind |
| 11   | p-01be  | 6          | 1       | 5           | 4        | 2    | 1    | 0    | 1    | implement checks / fix registration kind |
| 12   | p-63c5  | 7          | 3       | 4           | 6        | 0    | 3    | 0    | 3    | add ctx variant                          |
| 13   | p-a2d9  | 5          | 1       | 4           | 2        | 0    | 1    | 0    | 1    | add ctx variant                          |
| 14   | p-e62c  | 7          | 5       | 2           | 5        | 0    | 0    | 0    | 5    | HW-4: eager-or-suppress per service      |
| 15   | p-46ae  | 3          | 1       | 2           | 1        | 0    | 1    | 0    | 0    | add ctx variant                          |

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

43 projects analyzed · 30 clean · 12 with findings · 66 findings total
(HW-1 ×30, HW-2 ×11, HW-3 ×1, HW-4 ×24, HW-5 ×0) · 2 load errors under
investigation · HW-5 never fired in the wild.
