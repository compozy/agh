# TC-SEC-002: SQL injection prevented in store layer queries

**Priority:** P0 (Critical)
**Type:** Security
**Component:** `internal/store/globaldb/global_db_bundles.go`

## Objective

Validate that all store layer queries use parameterized statements, preventing SQL injection.

## Preconditions

- SQLite database with bundle tables

## Test Steps

1. Create activation with ID containing SQL: `"'; DROP TABLE bundle_activations; --"`
   **Expected:** ID stored as literal string, no SQL execution. Query returns the activation.

2. Get activation with malicious ID
   **Expected:** Returns ErrActivationNotFound (no match), no SQL injection

3. List activations with extension_name containing SQL injection payload
   **Expected:** Data stored as-is, queries use `?` placeholders

4. Delete activation with malicious ID
   **Expected:** Returns ErrActivationNotFound, table intact

5. Verify all ExecContext/QueryContext calls use `?` placeholders
   **Expected:** Code review confirms no string interpolation in SQL

## Edge Cases

- Unicode in activation fields → stored correctly as UTF-8
- Very long string values → stored without truncation (SQLite TEXT has no limit)
- NULL workspace_id stored via NullableString → retrieved correctly
