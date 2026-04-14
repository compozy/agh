# TC-FUNC-003: MultiRegistry Skips Non-Searchable Sources

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 2 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that sources with `Capabilities().Search == false` (e.g., GitHub) are skipped during search operations without errors.

## Preconditions

- Stub source with `Search` capability set to `false`.
- Another stub source with `Search` capability set to `true`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create MultiRegistry with both sources | **Expected:** Created without error. |
| 2 | Call `Search(ctx, "query", SearchOpts{})` | **Expected:** Only the searchable source is queried. Non-searchable source's Search method is never called. |
| 3 | Verify results come only from the searchable source | **Expected:** Results match searchable source output only. |

## Edge Cases

- All sources non-searchable: returns empty results, not an error.
