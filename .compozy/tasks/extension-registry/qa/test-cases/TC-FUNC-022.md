# TC-FUNC-022: CLI Extension Install With --asset Flag

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `--asset` flag selects a specific tar.gz asset from a GitHub release with multiple assets.

## Preconditions

- GitHub release with multiple `.tar.gz` assets.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install owner/repo --asset my-ext-linux-amd64.tar.gz` | **Expected:** Downloads the specified asset. |
| 2 | Run `agh extension install owner/repo` (no --asset, multiple tar.gz) | **Expected:** Error: multiple assets found, please specify `--asset`. |
| 3 | Run `agh extension install owner/repo --asset nonexistent.tar.gz` | **Expected:** Error: asset "nonexistent.tar.gz" not found in release. |

## Edge Cases

- Single tar.gz asset: should auto-select without `--asset`.
- Asset is not a tar.gz: should fail with format error.
