# TC-FUNC-005: MultiRegistry Download Delegates to Resolved Source

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `MultiRegistry.Download()` correctly delegates the download to the appropriate source and returns a valid `DownloadResult`.

## Preconditions

- Stub source configured to return a `DownloadResult` with a reader, version, and checksum.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Download(ctx, "test/pkg", DownloadOpts{Version: "1.0.0"})` | **Expected:** Returns `DownloadResult` with non-nil Reader, correct Version, and ContentSize > 0. |
| 2 | Read from the result's Reader | **Expected:** Data matches expected archive content. |
| 3 | Call with non-existent slug | **Expected:** Returns error, nil result. |

## Edge Cases

- Download with empty version: should resolve to latest.
- Download with `--asset` flag: should pass through to source.
