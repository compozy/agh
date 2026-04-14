# TC-SEC-006: Archive Compressed Size Enforcement

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/installer.go` |
| **OWASP** | A05:2021 — Security Misconfiguration |

## Objective

Validate that `io.LimitReader` enforces the 50MB compressed archive size limit, preventing disk exhaustion from oversized downloads.

## Preconditions

- Stub downloader returning a stream > 50MB.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Download stream of exactly 50MB | **Expected:** Install proceeds (at limit). |
| 2 | Download stream of 50MB + 1 byte | **Expected:** Read truncated by LimitReader. Extraction fails or produces corrupt archive error. |
| 3 | Download stream of 100MB | **Expected:** Only 50MB read from stream. Clear error about size limit. |

## Edge Cases

- ContentSize header says 30MB but actual stream is 60MB: LimitReader should still enforce.
- ContentSize header is 0 (unknown): LimitReader still wraps the reader.
