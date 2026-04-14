# TC-FUNC-016: Archive Extraction Handles Single Root Directory

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/extract.go` |

## Objective

Validate that when a tar.gz archive contains a single root directory (common with GitHub auto-generated archives), extraction correctly walks into it and places contents at the target root.

## Preconditions

- Archive with structure: `repo-v1.0.0/extension.toml`, `repo-v1.0.0/src/main.go`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Extract archive with single root dir `repo-v1.0.0/` | **Expected:** Files extracted as `extension.toml`, `src/main.go` (root dir stripped). |
| 2 | Extract archive with no root dir (files directly at root) | **Expected:** Files extracted as-is. |
| 3 | Extract archive with multiple root dirs | **Expected:** Files extracted as-is (no stripping). |

## Edge Cases

- Root dir name with special characters: handled correctly.
- Root dir is empty (only contains subdirectories): should still strip.
