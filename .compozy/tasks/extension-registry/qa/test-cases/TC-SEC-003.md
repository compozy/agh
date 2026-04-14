# TC-SEC-003: Decompression Bomb Protection

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/extract.go` |
| **OWASP** | A05:2021 — Security Misconfiguration |

## Objective

Validate that the extraction pipeline detects and aborts decompression bombs (small compressed archives that expand to massive size).

## Preconditions

- Crafted archive: ~1KB compressed, decompresses to >500MB.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create archive of repeated zero bytes (extreme compression ratio) | **Expected:** Archive created, very small file size. |
| 2 | Call `ExtractArchive()` with this archive | **Expected:** Extraction aborts when counting writer exceeds 500MB. Error message references decompression limit. |
| 3 | Verify no files remain in temp/target directory | **Expected:** Cleanup complete — no partial extraction. |
| 4 | Verify memory usage did not spike to 500MB+ | **Expected:** Stream processing kept memory usage bounded. |

## Edge Cases

- Archive just under limit (499MB decompressed): should succeed.
- Multiple files each under limit but total over: should trigger limit.
