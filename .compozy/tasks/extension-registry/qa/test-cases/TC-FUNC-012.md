# TC-FUNC-012: Installer Enforces File Count Limit

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/extract.go` |

## Objective

Validate that extraction rejects archives containing more than the maximum file count (default 10,000 entries).

## Preconditions

- Archive crafted with > 10,000 entries.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create tar.gz with 10,001 files | **Expected:** Archive created. |
| 2 | Call `ExtractArchive()` | **Expected:** Returns error mentioning file count limit exceeded. |
| 3 | Archive with exactly 10,000 files | **Expected:** Extraction succeeds. |

## Edge Cases

- Archive with mix of files and directories: both count toward limit.
