# TC-INT-009: GitHub Auth Token Increases Rate Limit

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/github/client.go` |

## Objective

Validate that the GitHub adapter sends the `Authorization` header when `GITHUB_TOKEN` is set.

## Preconditions

- Mock server that checks for Authorization header.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Set `GITHUB_TOKEN=test-token` and call `Info(ctx, "owner/repo")` | **Expected:** Request includes `Authorization: Bearer test-token` header. |
| 2 | Unset `GITHUB_TOKEN` and call `Info(ctx, "owner/repo")` | **Expected:** Request has no Authorization header. |
| 3 | Verify mock server returns different rate limits per auth state | **Expected:** Authenticated: 5000 limit. Unauthenticated: 60 limit. |

## Edge Cases

- `GITHUB_TOKEN` set to empty string: should behave as unset.
