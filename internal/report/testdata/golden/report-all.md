# cdd check

Root: `/projects/shop`

## `src/checkout.ts` (typescript)

| Unit | Kind | Location | ICPs | Limit | Status |
| --- | --- | --- | --- | --- | --- |
| `greet` | function | 1:1 | 2.5 | 10 | ok |
| **`CheckoutService`** | class | 12:3 | **14.5** | 10 | **over limit** |

Metrics:

- `greet` — code_branch 2×1=2, external_coupling 1×0.5=0.5
- `CheckoutService` — code_branch 6×1=6, condition 3×1=3, external_coupling 11×0.5=5.5

Warnings:

- unsupported syntax at 42:7, unit skipped

## `src/money.ts` (typescript)

| Unit | Kind | Location | ICPs | Limit | Status |
| --- | --- | --- | --- | --- | --- |
| `formatMoney` | function | 3:1 | 2 | 10 | ok |

Metrics:

- `formatMoney` — code_branch 1×1=1, condition 1×1=1

## Warnings

- the legacy mode is reported, not enforced yet

## Summary

3 units analyzed, 1 over limit, elapsed 1.234s.
