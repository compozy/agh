# TC-FUNC-002: MultiRegistry Search Handles Partial Source Failures

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `MultiRegistry.Search()` returns results from healthy sources when one or more sources fail.

## Preconditions

- Two stub sources: one returns results, one returns an error.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Configure source A to return `[{slug: "a/pkg"}]` and source B to return `error("timeout")` | **Expected:** Stubs configured. |
| 2 | Call `Search(ctx, "query", SearchOpts{})` | **Expected:** Returns `[{slug: "a/pkg"}]` from source A. No panic. |
| 3 | Check that the error from source B is logged but not returned as a blocking error | **Expected:** Function returns results, not an error. |

## Edge Cases

- All sources fail: returns an error (no results to return).
- Context cancelled mid-search: all goroutines exit cleanly.
