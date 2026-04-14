# TC-FUNC-030: Old Marketplace Package Fully Removed

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 2 min |
| **Module** | Migration Verification |

## Objective

Validate that the old `internal/skills/marketplace/` package is completely removed and no code references it.

## Preconditions

- ext-registry branch with all 5 tasks merged.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Check if `internal/skills/marketplace/` directory exists | **Expected:** Directory does not exist (fully deleted). |
| 2 | Grep codebase for `"skills/marketplace"` import path | **Expected:** Zero matches. |
| 3 | Grep codebase for `marketplace.Client` or `marketplace.New` (old types) | **Expected:** Zero matches outside test stubs. |
| 4 | Run `make build` | **Expected:** Compiles without "undefined" errors from missing marketplace package. |

## Edge Cases

- None — this is a binary check: the old package either exists or it doesn't.
