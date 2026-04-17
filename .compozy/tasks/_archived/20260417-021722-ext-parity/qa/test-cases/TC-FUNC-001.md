# TC-FUNC-001: CAS create rejects duplicate version

**Priority:** P0
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that the CAS (Compare-And-Swap) store enforces optimistic concurrency on creation. A `PutRaw` call with `ExpectedVersion=0` must succeed exactly once for a given `(kind, id)` pair. A second `PutRaw` with `ExpectedVersion=0` for the same `(kind, id)` must be rejected with a 409 conflict error, preventing silent overwrites of existing records.

## Preconditions

- A fresh in-memory or `t.TempDir()`-backed SQLite resource store is initialized.
- The store schema has been applied (tables and indexes exist).
- A valid `MutationActor` is configured with appropriate owner and source fields.
- At least one resource kind (e.g., `tool`) is registered.

## Test Steps

1. Call `PutRaw` with `Kind="tool"`, `ID="my-tool"`, `ExpectedVersion=0`, and a valid JSON payload.
   **Expected:** The call returns successfully. The returned record has `Version=1`, `Kind="tool"`, `ID="my-tool"`, and the payload matches what was submitted.

2. Call `PutRaw` again with `Kind="tool"`, `ID="my-tool"`, `ExpectedVersion=0`, and any valid JSON payload.
   **Expected:** The call returns an error. The error is a version conflict (409 semantics). The error message or type indicates the expected version did not match the current version. The original record at `Version=1` remains unchanged.

3. Call `Get` for `Kind="tool"`, `ID="my-tool"`.
   **Expected:** The returned record still has `Version=1` and contains the payload from step 1, confirming the second write had no side effects.

## Edge Cases

- Two goroutines race `PutRaw` with `ExpectedVersion=0` for the same `(kind, id)` simultaneously: exactly one succeeds, the other receives 409.
- `PutRaw` with `ExpectedVersion=0` for a different `ID` (e.g., `"my-tool-2"`) succeeds independently, proving the conflict check is scoped to `(kind, id)`.
- `PutRaw` with `ExpectedVersion=0` for a different `Kind` but same `ID` (e.g., `Kind="agent"`, `ID="my-tool"`) succeeds, confirming the uniqueness constraint spans both kind and id.
- `PutRaw` with `ExpectedVersion=0` and an empty or nil payload is rejected at validation, not at the CAS layer.
