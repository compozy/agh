# TC-FUNC-019: CLI Extension Search With --limit Flag

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 2 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `--limit N` flag restricts the number of search results returned.

## Preconditions

- Query that returns multiple results.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension search "test" --limit 1` | **Expected:** At most 1 result displayed. |
| 2 | Run `agh extension search "test" --limit 0` | **Expected:** Error or zero results (validate behavior). |
| 3 | Run `agh extension search "test" --limit 100` | **Expected:** All available results (up to 100). |

## Edge Cases

- Negative limit: should return error.
- Non-numeric limit: should return parse error.
