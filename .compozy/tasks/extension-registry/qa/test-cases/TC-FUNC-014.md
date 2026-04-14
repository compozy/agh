# TC-FUNC-014: Installer MoveInstalledDir Atomic Replace With Backup

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/installer.go` |

## Objective

Validate that `MoveInstalledDir()` atomically replaces existing installations with backup-on-replace behavior.

## Preconditions

- Existing extension installed at target path.
- New extracted directory ready to move.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `MoveInstalledDir(src, dst)` where `dst` already exists | **Expected:** Old directory backed up, new directory placed at `dst`. |
| 2 | Verify old directory contents are in backup location | **Expected:** Backup exists with original files. |
| 3 | Call `MoveInstalledDir(src, dst)` where `dst` does not exist | **Expected:** Directory moved to `dst` directly, no backup created. |

## Edge Cases

- Source directory empty: should still move (empty dir is valid).
- Target path has special characters: should handle correctly.
- Move across filesystem boundaries: should fall back to copy+delete if rename fails.
