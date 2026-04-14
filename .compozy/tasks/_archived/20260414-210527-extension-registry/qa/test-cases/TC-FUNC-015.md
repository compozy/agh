# TC-FUNC-015: Installer Stale Temp Directory Cleanup

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/installer.go` |

## Objective

Validate that the installer cleans up stale temporary directories (>1 hour old) before starting a new install.

## Preconditions

- Temp directory with `.agh-install-` prefix older than 1 hour exists in the install root.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create a `.agh-install-XXXXX` directory with mtime > 1 hour ago | **Expected:** Directory created. |
| 2 | Run `Install()` for a new extension | **Expected:** Stale temp directory removed before install proceeds. |
| 3 | Create a `.agh-install-XXXXX` directory with mtime < 1 hour ago | **Expected:** Directory is NOT removed (recent, possibly in-use). |

## Edge Cases

- Multiple stale directories: all should be cleaned.
- Permission denied on stale dir: should log warning and continue install.
