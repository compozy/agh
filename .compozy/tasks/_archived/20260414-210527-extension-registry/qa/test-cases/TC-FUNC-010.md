# TC-FUNC-010: Installer Enforces Compressed Archive Size Limit

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/installer.go` |

## Objective

Validate that the installer rejects archives exceeding the compressed size limit (default 50MB).

## Preconditions

- Stub downloader returning a stream with `ContentSize` exceeding 50MB.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create downloader returning ContentSize of 60MB | **Expected:** Downloader configured. |
| 2 | Call `Install(ctx, "test/ext", dlOpts, targetDir)` | **Expected:** Returns error mentioning archive size exceeds limit. No files extracted. |
| 3 | Verify no temp directories remain | **Expected:** Cleanup occurred despite failure. |

## Edge Cases

- Archive exactly at limit (50MB): should be accepted.
- Archive at limit + 1 byte: should be rejected.
- ContentSize unknown (0 or -1): should apply `io.LimitReader` during read instead.
