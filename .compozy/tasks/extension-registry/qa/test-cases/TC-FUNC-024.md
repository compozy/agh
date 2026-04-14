# TC-FUNC-024: CLI Extension Remove With DB Rollback on Failure

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `agh extension remove` cleans both filesystem and DB, and rolls back DB changes if filesystem deletion fails.

## Preconditions

- Extension installed with both filesystem and DB entries.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension remove <name>` (normal case) | **Expected:** Directory deleted, DB entry removed. Success message. |
| 2 | Make extension directory read-only, then run remove | **Expected:** Filesystem deletion fails. DB entry should NOT be removed (rollback). Error message displayed. |
| 3 | Remove a non-existent extension | **Expected:** Error: extension "<name>" not found. |

## Edge Cases

- Extension in DB but directory already missing: should remove DB entry and warn.
- Extension directory exists but not in DB: should report "not found" (DB is source of truth).
