# TC-SEC-007: File Permission Handling in Extracted Archives

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/extract.go` |
| **OWASP** | A01:2021 — Broken Access Control |

## Objective

Validate that extracted files have safe permissions and that setuid/setgid bits are stripped.

## Preconditions

- Archive with files having various permission modes.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Archive entry with mode 0777 (world-writable executable) | **Expected:** Extracted with safe mode (e.g., 0755 for dirs, 0644 for files). |
| 2 | Archive entry with setuid bit (mode 04755) | **Expected:** Setuid bit stripped. File mode is 0755 or similar. |
| 3 | Archive entry with setgid bit (mode 02755) | **Expected:** Setgid bit stripped. |
| 4 | Archive entry with sticky bit | **Expected:** Handled safely. |
| 5 | Directory with mode 0000 (no permissions) | **Expected:** Created with at least read+execute for owner. |

## Edge Cases

- Device files in archive: should be rejected.
- FIFO/named pipe entries: should be rejected.
