# TC-INT-008: Second session invalidates older nonce

**Priority:** P1
**Type:** Integration
**Package:** internal/extension
**Related Tasks:** 05

## Objective

Validate nonce rotation: when an extension is initialized a second time (e.g., new session for the same source), the older nonce becomes invalid. Any snapshot call using the stale nonce must be rejected, ensuring only the most recent session can mutate resources.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables and source state
- Extension host initialized
- Two extension sessions can be started sequentially for the same source identifier

## Test Steps

1. Start extension session A for `source=ext-alpha`. Capture `nonce_A` from the initialize response.
   **Expected:** Session A initialized. `nonce_A` is non-empty and persisted in `resource_source_state`.

2. Session A issues `resources/snapshot` with `nonce_A` and 2 tool records.
   **Expected:** Snapshot accepted. Records persisted.

3. Start extension session B for the same `source=ext-alpha`. Capture `nonce_B` from the initialize response.
   **Expected:** Session B initialized. `nonce_B` differs from `nonce_A`. `resource_source_state` updated with `nonce_B`.

4. Session A issues `resources/snapshot` with `nonce_A` (the old nonce).
   **Expected:** Snapshot rejected with a stale nonce error. Records from step 2 remain unchanged.

5. Session B issues `resources/snapshot` with `nonce_B` and 3 tool records.
   **Expected:** Snapshot accepted. Store now contains Session B's 3 records.

6. Verify Session A's original 2 records have been replaced by Session B's 3 records (since the snapshot is full-replace for source+kind).
   **Expected:** Only Session B's 3 tool records exist for `source=ext-alpha`.

## Edge Cases

- Session A and B started nearly simultaneously — nonce assignment is serialized, no race
- Session B crashes before publishing any snapshot — Session A's records remain until B publishes or source is reset
- Three sequential sessions — only the latest nonce is valid
- Stale nonce error includes enough context to diagnose (e.g., which source, expected vs provided)
- Extension retries with stale nonce after receiving error — still rejected (no retry exemption)
