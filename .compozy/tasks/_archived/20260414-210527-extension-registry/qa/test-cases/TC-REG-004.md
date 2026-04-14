# TC-REG-004: Database Migrations Preserve Existing Extension Data

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Regression |
| **Estimated Time** | 3 min |
| **Module** | `internal/store/globaldb/global_db.go` |
| **Changed In** | Task 04 — Schema Changes |

## Objective

Validate that the 3 new nullable columns (`registry_slug`, `registry_name`, `remote_version`) are added without losing existing extension data.

## Preconditions

- Existing SQLite database with pre-migration extensions table containing rows.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Open pre-migration database with existing extension rows | **Expected:** Database opens without error. |
| 2 | Run schema migration (ALTER TABLE ADD COLUMN) | **Expected:** Columns added. Existing rows have NULL for new columns. |
| 3 | Query existing extension data | **Expected:** All pre-existing fields intact (name, path, source, etc.). |
| 4 | Insert new extension with registry metadata | **Expected:** New row has all columns populated including new ones. |

## Regression Risk

High — ALTER TABLE on SQLite with existing data must preserve all existing rows.
