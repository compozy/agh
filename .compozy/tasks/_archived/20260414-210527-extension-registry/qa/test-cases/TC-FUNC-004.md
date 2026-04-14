# TC-FUNC-004: MultiRegistry Info Resolves From Highest Priority Source

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `MultiRegistry.Info()` returns detail from the highest-priority source that can resolve the slug.

## Preconditions

- Two sources both capable of returning info for slug `test/pkg`.
- Source A at priority 1, Source B at priority 2.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Info(ctx, "test/pkg")` | **Expected:** Returns Detail from Source B (higher priority). |
| 2 | Configure Source B to return `ErrNotSupported` for the slug | **Expected:** Falls back to Source A. |
| 3 | Both sources return errors | **Expected:** Returns error indicating package not found. |

## Edge Cases

- Slug not found in any source: clear "not found" error.
