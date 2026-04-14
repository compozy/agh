# TC-INT-002: ClawHub Download Returns Valid Archive Stream

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/clawhub/client.go` |

## Objective

Validate that ClawHub Download returns a readable tar.gz stream with correct metadata.

## Preconditions

- Mock server returning a valid tar.gz response with Content-Length header.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Download(ctx, "test-skill", DownloadOpts{})` (latest) | **Expected:** Returns `DownloadResult` with non-nil Reader and ContentSize matching Content-Length. |
| 2 | Read full stream and verify it's valid gzip | **Expected:** `gzip.NewReader()` succeeds on the data. |
| 3 | Call `Download(ctx, "test-skill", DownloadOpts{Version: "1.0.0"})` | **Expected:** Uses versioned endpoint if available. |

## Edge Cases

- Version not found: returns 404, mapped to clear error.
- Connection reset during download: reader returns io.ErrUnexpectedEOF.
