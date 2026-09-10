# False-Positive Budgets

Per-rule statement of where healthwash *could* over-report, why each residual
risk is tolerated, and the empirical evidence from the 2026-09-10 ecology scan
(43 samber/do v2 consumers analyzed, 66 findings, **0 confirmed false
positives**). A rule that exceeds its budget is a bug: fix the rule, don't
grow the budget.

## How FPs are prevented structurally

1. **Type-based detection only.** Every rule keys on interface satisfaction of
   the *stored instance* — exactly what the samber/do sweep type-asserts
   (`service_eager.go:88,94`, `service_lazy.go:136`). No name heuristics, no
   comment guessing, no flow analysis that can misfire.
2. **Interface-satisfaction, not method-name matching.** All four Shutdowner
   variants share the method name `Shutdown`; satisfaction of the resolved
   `*types.Interface` is the only sound test (`resolve.go`).
3. **Registration-function matching by package path** (`github.com/samber/do/v2`),
   never identifier text — dot-imports and renames cannot spoof or hide sites.
4. **Compile-gated fixtures + discrimination proofs.** Every rule has a fixture
   that a mutant analyzer (rule disabled) fails to reproduce, so a rule whose
   tests test nothing cannot ship (`TestDiscriminationProofs`).
5. **Drift matrix.** README §2 mechanism claims are asserted against the real
   samber/do v2.0.0 and v2.1.0 sources in CI (`TestDriftV200`, `TestDriftV210`);
   an upstream behavior change fails CI before it can produce wrong findings.

## Budgets

| Rule | Findings (ecology) | FP class | Budget | Mitigation when exceeded |
|------|-------------------|----------|--------|--------------------------|
| HW-1 | 30 | "Service can't meaningfully fail" disagreement | **0** — the finding is a type fact: Shutdowner without any Healthchecker renders an unconditional pass. Whether a check is *worth writing* is the user's call, expressed via a reasoned suppression, not an FP. | allowlist entry or `//samber-linter:allow hw-1 <reason>` |
| HW-2 | 11 | "This bare check never blocks" | **0** — the sweep cannot cancel a bare check; the degradation is factual regardless of the check's current body. | reasoned suppression |
| HW-3 | 1 | none known | **0** — the transient healthcheck is an upstream TODO that always returns nil; a check on a transient *never executes*. | reasoned suppression |
| HW-4 | 24 | **Judgment calls only.** A lazily-built service that is in fact constructed during boot reports green "until first resolution" for microseconds; HW-4 still flags it. | **Soft ceiling ≈ 20% of findings per project.** HW-4 is `info`/Medium (never fails CI by default) precisely because the "when is it actually built" answer is not statically knowable. | `--disable HW-4`, allowlist, or register boot-critical services eagerly |
| HW-5 | 0 | none known | **0** — pointer-receiver/value-registration is a compile-time fact. | reasoned suppression |
| HW-0 | 0 | none known | **0** — malformed directive is a syntactic fact; also fires orphaned (no matching finding) since v0.1.1. | write the reason |
| HW-unresolved | 0 (off) | Interface-typed closure results are statically unknowable — reporting them as violations would be the FP. | **0 while off**; in `--strict` the finding is explicitly "unresolved", never a guessed rule. | keep `--strict` off, or resolve the concrete type |

## Known boundary cases (documented, not bugs)

- **HW-5 with Shutdowner on T:** when a value-registered type carries its check
  on `*T` only, HW-5 reports and HW-1 is skipped — both share the single remedy
  (register the pointer). One finding per fixable cause.
- **Closure providers returning interfaces** resolve to `Unresolved`, never to
  a guessed HW-N, because the sweep type-asserts the stored concrete instance.
- **Transients with a check** report HW-3 and never HW-2/HW-4: the check can
  never execute, so wrapper-level nuance is moot.
- **Duplicate registrations** each report at their own site (each instance is
  separately sweep-visible) but count once in the HW-6 coverage denominator.

## Ecology evidence

Scan: 45 local consumers (43 analyzable; 2 broken by their own toolchains —
a `go.work` requiring go ≥ 1.27 and a stale `go.mod`). Distribution:
HW-1×30, HW-2×11, HW-3×1, HW-4×24, HW-5×0, HW-0×0. Spot-checked findings in
CV (7× HW-1), samber-do-auditlog (7× HW-1, 5× HW-2, 1× HW-3), and PapDashboard
(3× HW-2, 3× HW-4) were all real mechanism facts; zero confirmed FPs.
