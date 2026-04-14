# SMOKE-004: Extension Remove Cleans Filesystem and DB

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Smoke |
| **Estimated Time** | 3 min |
| **Module** | CLI / Extension Management |

## Objective

Validate that `agh extension remove <name>` deletes the extension directory and removes the database entry.

## Preconditions

- An extension previously installed via `agh extension install`.
- Extension directory and DB entry exist.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension remove <name>` | **Expected:** Success message. Exit code 0. |
| 2 | Check `~/.agh/extensions/<name>/` | **Expected:** Directory no longer exists. |
| 3 | Query `extensions` table for the removed name | **Expected:** No row found. |
| 4 | Run `agh extension remove <name>` again | **Expected:** Error message: extension not found. Non-zero exit code. |

## Edge Cases

- Remove extension while daemon is running: should succeed (Phase 1 does not notify daemon).
- Remove extension with corrupted directory (missing files): should still clean DB entry.
