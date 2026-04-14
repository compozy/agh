# TC-INT-005: GitHub Info Parses Release Metadata

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/github/client.go` |

## Objective

Validate that the GitHub adapter correctly parses release API responses into `Detail` structs.

## Preconditions

- Mock server returning canned GitHub release JSON for `owner/repo`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Info(ctx, "owner/repo")` | **Expected:** Returns `Detail` with latest version, readme (from release body), available versions list. |
| 2 | Verify `Detail.Versions` contains all non-draft, non-prerelease tags | **Expected:** Sorted list of version strings. |
| 3 | Verify `Detail.Listing.Source` is "github" | **Expected:** Source correctly set. |
| 4 | Call `Info(ctx, "nonexistent/repo")` | **Expected:** Returns 404-based error. |

## Edge Cases

- Repo with no releases: returns appropriate error.
- Release body is empty: readme field is empty string, not nil.
- Very long release body: truncated or handled within memory limits.
