# TC-INT-007: GitHub Excludes Pre-Release and Draft Releases

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/github/client.go` |

## Objective

Validate that the GitHub adapter filters out pre-release and draft releases when resolving latest version.

## Preconditions

- Mock server with releases: `v3.0.0-beta` (prerelease), `v2.0.0` (draft), `v1.0.0` (stable).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Info(ctx, "owner/repo")` | **Expected:** Latest version is `v1.0.0` (only stable release). |
| 2 | Call `Download(ctx, "owner/repo", DownloadOpts{})` (latest) | **Expected:** Downloads assets from `v1.0.0` release. |
| 3 | Call `Download(ctx, "owner/repo", DownloadOpts{Version: "v3.0.0-beta"})` (explicit) | **Expected:** Downloads pre-release when explicitly requested. |

## Edge Cases

- All releases are pre-release: `Info()` should return error or empty latest.
- All releases are drafts: same behavior.
