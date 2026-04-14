# TC-SEC-001: Path Traversal Prevention in Archive Extraction

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/extract.go` |
| **OWASP** | A01:2021 — Broken Access Control |

## Objective

Validate that archive entries with path traversal sequences (`../`) are rejected during extraction, preventing writes outside the target directory.

## Preconditions

- Crafted tar.gz archive with path traversal entries.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Archive entry: `../../../etc/passwd` | **Expected:** Extraction fails with path traversal error. No file written outside target. |
| 2 | Archive entry: `foo/../../bar.txt` | **Expected:** Rejected — resolved path escapes target root. |
| 3 | Archive entry: `./safe/file.txt` | **Expected:** Accepted — stays within target. |
| 4 | Archive entry: absolute path `/etc/passwd` | **Expected:** Rejected — absolute paths not allowed. |
| 5 | Verify `CleanArchiveEntryPath()` strips leading `./` and `../` | **Expected:** Returns cleaned path or error for traversal. |
| 6 | Verify `PathWithinRoot()` catches all traversal variants | **Expected:** Returns false for any path escaping root. |

## Edge Cases

- URL-encoded traversal: `%2e%2e%2f` — should be decoded and caught.
- Unicode traversal: should be normalized and caught.
- Null byte in path: should be rejected.
- Very deeply nested valid path: should be accepted.
