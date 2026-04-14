# TC-FUNC-011: Installer Enforces Decompressed Size Limit

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/extract.go` |

## Objective

Validate that the extraction pipeline rejects archives whose decompressed content exceeds the limit (default 500MB), preventing decompression bombs.

## Preconditions

- Archive crafted to decompress to more than 500MB (highly compressible data).
- Installer with default size limits.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create tar.gz with repeated zero bytes that compress well (compressed ~1MB, decompressed ~600MB) | **Expected:** Archive created. |
| 2 | Call `Install()` or `ExtractArchive()` with this archive | **Expected:** Returns error mentioning decompressed size limit exceeded. |
| 3 | Verify partial extraction cleaned up | **Expected:** No files remain in target or temp directory. |

## Edge Cases

- Archive with many small files totaling over limit: should trigger the same protection.
- Decompression stops mid-file: should not leave corrupt partial file.
