# SMOKE-001: Resource CRUD Round-Trip

**Priority:** P0
**Type:** Smoke
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that the raw persistence kernel correctly implements the full CRUD lifecycle for resource records. Each operation (PutRaw, GetRaw, ListRaw, DeleteRaw) must return the expected shapes with correct version, scope, owner, and source stamps. This is the foundational layer all typed stores build on, so any breakage here cascades everywhere.

## Preconditions

- SQLite database initialized via `t.TempDir()` with the resource schema applied
- A raw resource store instance created with default options
- No pre-existing records in the store

## Test Steps

1. **PutRaw a new resource record** with kind="test.widget", scope="workspace", owner_kind="session", owner_id="sess-001", source="manual", and a valid JSON spec payload.
   **Expected:** Returns a record with version=1, matching kind/scope/owner fields, non-zero created_at and updated_at timestamps, and the spec payload preserved byte-for-byte.

2. **GetRaw the record** by its returned ID and kind.
   **Expected:** Returns the identical record from step 1 with all fields matching, including version=1, scope, owner_kind, owner_id, source, and spec payload.

3. **PutRaw an update** to the same record with expected_version=1 and a modified spec payload.
   **Expected:** Returns the record with version=2, updated_at strictly greater than the original, and the new spec payload. All other fields (kind, scope, owner_kind, owner_id, source) remain unchanged.

4. **ListRaw with kind filter** set to "test.widget".
   **Expected:** Returns a slice containing exactly one record matching the updated version=2 state from step 3.

5. **ListRaw with scope filter** set to "workspace".
   **Expected:** Returns a non-empty slice that includes the test record.

6. **ListRaw with owner filter** set to owner_kind="session", owner_id="sess-001".
   **Expected:** Returns a slice containing the test record.

7. **DeleteRaw the record** by ID and kind with expected_version=2.
   **Expected:** Returns no error. The record is removed from the store.

8. **GetRaw the deleted record** by its original ID and kind.
   **Expected:** Returns a not-found error (or nil record), confirming the delete was effective.

## Edge Cases

- PutRaw with expected_version=0 on an already-existing ID returns a conflict/version-mismatch error
- PutRaw with expected_version=5 on a record at version=2 returns a conflict/version-mismatch error
- DeleteRaw with a stale expected_version returns a conflict/version-mismatch error
- GetRaw with a non-existent ID returns a not-found error, not a panic or empty record
- ListRaw with no matching filters returns an empty slice, not nil
- PutRaw with an empty spec payload succeeds (spec is allowed to be empty JSON `{}`)
- PutRaw with very large spec payload (64KB+) succeeds and round-trips correctly
- Concurrent PutRaw calls on the same record serialize correctly via version checks
