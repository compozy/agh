# TC-SEC-004: Extension Cannot Call Direct Put or Delete

**Priority:** P0
**Type:** Security
**Package:** internal/extension
**Related Tasks:** 05

## Objective

Validate that extensions are restricted to the snapshot-based write path and cannot invoke `resources/put` or `resources/delete` Host API methods directly. These methods must either not be registered in the extension-facing API surface or return 403 if called.

## Preconditions

- Extension `ext-rogue` is registered and has an active session with a valid nonce.
- The resource runtime is operational with at least one record persisted.
- The Host API method registry is initialized for the extension session.

## Test Steps

1. As `ext-rogue`, send a JSON-RPC request for method `resources/put` with a valid record payload.
   **Expected:** The request is rejected. Either the method is not found (JSON-RPC method not found error, code -32601) or returns 403 Forbidden. No record is created or modified.

2. As `ext-rogue`, send a JSON-RPC request for method `resources/delete` targeting an existing record owned by `ext-rogue` itself.
   **Expected:** The request is rejected with the same error class as step 1. Even deleting own records via the direct API is not permitted for extensions.

3. Verify the targeted record from step 2 still exists and is unmodified.
   **Expected:** The record is intact with its original content and metadata.

4. As `ext-rogue`, submit a valid snapshot that removes the record (by omitting it from the snapshot).
   **Expected:** The snapshot succeeds and the record is removed through the authorized snapshot reconciliation path.

5. As `ext-rogue`, attempt to call `resources/put` by varying the method name casing (e.g., `Resources/Put`, `RESOURCES/PUT`) or using path traversal patterns.
   **Expected:** All variations are rejected. Method dispatch is case-sensitive and does not normalize input in a way that could bypass restrictions.

## Edge Cases

- Extension constructs a raw JSON-RPC request outside the SDK to bypass any client-side method filtering.
- Extension calls a batch JSON-RPC request containing both allowed methods (e.g., `resources/list`) and disallowed methods (`resources/put`) to see if the batch partially succeeds.
- Extension attempts `resources/put` with the `source` field set to a different extension's identifier.
- Extension attempts to invoke internal/admin-only methods by guessing method names (e.g., `resources/admin/put`, `resources/internal/put`).

## Threat Model

This test prevents **bypass of the snapshot reconciliation model**. The snapshot path provides atomic, auditable, source-scoped writes with conflict detection. If extensions could call `resources/put` or `resources/delete` directly, they could perform targeted mutations outside the reconciliation model -- modifying individual records without the full-snapshot consistency guarantees, bypassing conflict detection, and potentially racing with other extensions' snapshots to create inconsistent state.
