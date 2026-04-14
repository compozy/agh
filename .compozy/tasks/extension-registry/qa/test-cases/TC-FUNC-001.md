# TC-FUNC-001: MultiRegistry Concurrent Search Aggregation

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `MultiRegistry.Search()` queries all registered sources concurrently and merges results with priority-based deduplication.

## Preconditions

- Two or more `RegistrySource` stubs configured with overlapping result sets.
- Stubs configured with different priorities.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create `MultiRegistry` with 2 stub sources (priority 1 and priority 2), both returning a listing with slug `test/pkg` | **Expected:** MultiRegistry created without error. |
| 2 | Call `Search(ctx, "test", SearchOpts{})` | **Expected:** Returns exactly 1 listing for `test/pkg` from the higher-priority source. |
| 3 | Verify both sources were queried (check call counts) | **Expected:** Both sources received the Search call. |
| 4 | Verify results are deduplicated by slug | **Expected:** No duplicate slugs in results. |

## Edge Cases

- One source returns error: other source results still returned.
- All sources return errors: returns aggregated error.
- Source with `Capabilities().Search == false`: skipped silently.
- Empty query string: handled per source behavior.

## Related Tests

- TC-FUNC-002 (partial failure), TC-FUNC-003 (non-searchable skip)
