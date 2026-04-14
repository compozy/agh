# TC-INT-001: ClawHub Search Returns Valid Listings

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/clawhub/client.go` |

## Objective

Validate that the ClawHub adapter correctly parses API search responses into `Listing` structs.

## Preconditions

- `httptest.Server` returning canned ClawHub search JSON.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Start mock server returning valid search response with 3 skills | **Expected:** Server running. |
| 2 | Create ClawHub client pointing at mock server | **Expected:** Client created. |
| 3 | Call `Search(ctx, "test", SearchOpts{Limit: 10})` | **Expected:** Returns 3 `Listing` structs with populated slug, name, version, author, downloads. |
| 4 | Verify `Listing.Source` is "clawhub" | **Expected:** Source field correctly set. |
| 5 | Verify `Listing.Type` is `PackageTypeSkill` | **Expected:** ClawHub returns skills only. |

## Edge Cases

- Empty search response: returns empty slice, not error.
- Malformed JSON: returns parse error.
- HTTP 500: returns error with status code context.
