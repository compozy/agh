# TC-INT-006: GitHub Download Selects Correct Asset

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Integration |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/github/client.go` |

## Objective

Validate GitHub adapter's asset selection logic for releases with single and multiple tar.gz assets.

## Preconditions

- Mock server with canned release responses.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Release with single `repo-v1.0.0.tar.gz` asset. Call `Download(ctx, "owner/repo", DownloadOpts{})` | **Expected:** Auto-selects the single tar.gz asset. |
| 2 | Release with 3 tar.gz assets. Call `Download(ctx, "owner/repo", DownloadOpts{})` | **Expected:** Error: multiple assets, specify `--asset`. |
| 3 | Release with 3 tar.gz assets. Call `Download(ctx, "owner/repo", DownloadOpts{Asset: "repo-linux-amd64.tar.gz"})` | **Expected:** Selects the named asset. |
| 4 | Release with no tar.gz assets. Call `Download(ctx, "owner/repo", DownloadOpts{})` | **Expected:** Falls back to GitHub auto-generated source tarball. |

## Edge Cases

- Asset URL requires redirect (GitHub CDN): should follow redirects.
- Asset larger than 50MB: should be caught by installer's LimitReader.
