# TC-SEC-002: Symlink Rejection in Archive Extraction

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Security |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/extract.go` |
| **OWASP** | A01:2021 — Broken Access Control |

## Objective

Validate that symbolic links in tar.gz archives are rejected during extraction, preventing symlink-based file access attacks.

## Preconditions

- Crafted tar.gz archive containing symlink entries.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Archive with symlink `link.txt -> /etc/passwd` | **Expected:** Extraction fails or skips symlink with warning. No symlink created. |
| 2 | Archive with symlink `link.txt -> ../secret.key` | **Expected:** Rejected — symlink target escapes root. |
| 3 | Archive with hardlink pointing outside root | **Expected:** Rejected. |
| 4 | Verify no symlinks exist in extracted directory | **Expected:** `filepath.Walk` finds zero symlinks. |

## Edge Cases

- Symlink to directory: should be rejected.
- Chain of symlinks (A -> B -> C -> /etc/passwd): should be rejected.
