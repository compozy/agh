# TC-FUNC-013: Snapshot rejects non-active session nonce

**Priority:** P0
**Type:** Functional
**Package:** internal/extension
**Related Tasks:** 05

## Objective

Validate that the extension session nonce mechanism prevents stale sessions from submitting snapshots. When a new session is started for the same source (extension), the previous session's nonce is invalidated. Any attempt to submit a resource snapshot using the old nonce must be rejected. This ensures that only the most recent active session for a given source can modify resource state, preventing split-brain scenarios.

## Preconditions

- The extension runtime is initialized and ready to accept session registrations.
- A resource store is available and connected to the extension runtime.
- Extension source `ext-A` is registered and has a valid manifest.

## Test Steps

1. Start session A for source `ext-A`. Capture the returned session nonce (`nonce-A`).
   **Expected:** Session A is active. `nonce-A` is a non-empty, cryptographically random or unique token.

2. Using `nonce-A`, submit a resource snapshot for source `ext-A` containing `[tool/t1, tool/t2]`.
   **Expected:** The snapshot is accepted. Both `tool/t1` and `tool/t2` are persisted in the store.

3. Start session B for source `ext-A`. Capture the returned session nonce (`nonce-B`).
   **Expected:** Session B is active. `nonce-B` is different from `nonce-A`. Session A's nonce is invalidated (session A is no longer the active session for `ext-A`).

4. Using `nonce-A` (now stale), attempt to submit a resource snapshot for source `ext-A` containing `[tool/t1-updated]`.
   **Expected:** The snapshot is rejected with an error indicating the nonce is invalid or the session is no longer active. The error is distinct from other failure modes (e.g., not a version conflict or permission error). Records `tool/t1` and `tool/t2` remain unchanged.

5. Using `nonce-B`, submit a resource snapshot for source `ext-A` containing `[tool/t1-v2, tool/t3]`.
   **Expected:** The snapshot is accepted. The store now contains `tool/t1-v2` and `tool/t3`. `tool/t2` may be removed if the snapshot is a full replacement, or retained if it is additive.

6. Verify records in the store.
   **Expected:** The store state reflects only the snapshots from valid (active) sessions. No data from the rejected step 4 snapshot is present.

## Edge Cases

- Starting a third session C for the same source invalidates `nonce-B`, and session B can no longer submit snapshots.
- A nonce from a source that has been completely deregistered is rejected.
- Race condition: sessions A and B start nearly simultaneously for the same source. Exactly one wins the active slot; the other's nonce is immediately invalid.
- The nonce is not guessable: incrementing or modifying `nonce-A` does not produce a valid nonce.
- Session start for a different source (`ext-B`) does not affect `ext-A`'s active nonce.
- Submitting a `PutRaw` (single record) with a stale nonce is also rejected, not just snapshots.
