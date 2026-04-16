# TC-FUNC-003: CAS update rejects stale version

**Priority:** P0
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that CAS update semantics work correctly for multi-step version progression. After a record is created at version 1, an update with `ExpectedVersion=1` must succeed and bump to version 2. A subsequent update attempt using the now-stale `ExpectedVersion=1` must be rejected with a 409 conflict, preventing lost-update anomalies.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A valid `MutationActor` is configured.
- At least one resource kind (e.g., `tool`) is registered.

## Test Steps

1. Call `PutRaw` with `Kind="tool"`, `ID="versioned-tool"`, `ExpectedVersion=0`, and payload `{"name": "v1"}`.
   **Expected:** Returns successfully with `Version=1`.

2. Call `PutRaw` with `Kind="tool"`, `ID="versioned-tool"`, `ExpectedVersion=1`, and payload `{"name": "v2"}`.
   **Expected:** Returns successfully with `Version=2`. The payload is updated to `{"name": "v2"}`.

3. Call `PutRaw` with `Kind="tool"`, `ID="versioned-tool"`, `ExpectedVersion=1`, and payload `{"name": "v2-conflict"}`.
   **Expected:** Returns a version conflict error (409 semantics). The error indicates that `ExpectedVersion=1` does not match the current version `2`.

4. Call `Get` for `Kind="tool"`, `ID="versioned-tool"`.
   **Expected:** The record has `Version=2` with payload `{"name": "v2"}`, confirming the conflicting write from step 3 was completely rejected.

5. Call `PutRaw` with `Kind="tool"`, `ID="versioned-tool"`, `ExpectedVersion=2`, and payload `{"name": "v3"}`.
   **Expected:** Returns successfully with `Version=3`, confirming the version chain continues normally after a conflict rejection.

## Edge Cases

- Update with `ExpectedVersion=0` on an existing record is rejected (that is a create, not an update, and the record already exists).
- Update with `ExpectedVersion` set to a value higher than the current version (e.g., `ExpectedVersion=99`) is rejected, not silently accepted.
- Negative `ExpectedVersion` values (e.g., `-1`) are rejected at validation.
- Two concurrent updates both using `ExpectedVersion=1`: exactly one succeeds with `Version=2`, the other receives 409. The winner is determined by SQLite's serialization.
- Update that changes only metadata (not payload) still increments the version.
