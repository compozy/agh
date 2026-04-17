# TC-INT-016: Boot rebuild reconstructs bridge desired state

**Priority:** P0
**Type:** Integration
**Package:** internal/daemon, internal/bridges
**Related Tasks:** 11

## Objective

Validate that persisted `bridge.instance` resource records are correctly loaded during daemon boot and reconstructed into the bridge registry's desired state. After boot, bridges must be registered and ready without any runtime re-declaration.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Pre-populated resource records: 2 `bridge.instance` records with valid bridge configurations
- Daemon boot sequence that includes bridge projector reconciliation
- Bridge registry/runtime initialized as part of boot

## Test Steps

1. Persist 2 `bridge.instance` resource records directly into the SQLite database:
   - `bridge-ws-1`: WebSocket bridge with endpoint and auth configuration
   - `bridge-http-1`: HTTP bridge with URL and header configuration
   **Expected:** Both records present in `resource_records`.

2. Boot the daemon (or the relevant subsystem that performs bridge boot rebuild).
   **Expected:** Boot completes without error. Bridge projector triggered.

3. Query the bridge registry for registered bridge instances.
   **Expected:** Both `bridge-ws-1` and `bridge-http-1` are registered.

4. Verify `bridge-ws-1` has the correct desired state (endpoint, auth config).
   **Expected:** Configuration matches persisted record data.

5. Verify `bridge-http-1` has the correct desired state (URL, headers).
   **Expected:** Configuration matches persisted record data.

6. Verify the bridge registry's desired state matches the persisted records exactly — no extra bridges, no missing bridges.
   **Expected:** Registry contains exactly 2 bridges, both matching their resource records.

7. Add a third `bridge.instance` record and trigger reconciliation (without restart).
   **Expected:** Bridge registry now contains 3 bridges. The runtime update path works post-boot as well.

## Edge Cases

- Boot with zero bridge records — registry starts empty, no error
- Boot with a bridge record referencing an unavailable bridge type — record loaded, bridge marked as degraded or error logged
- Boot with duplicate bridge IDs — should not occur due to unique constraint, but if it does, deterministic resolution
- Bridge record with missing required config fields — projector skips or logs warning, other bridges unaffected
- Concurrent boot and write — boot rebuild picks up whatever is committed at query time
- Registry state after removing a bridge record and reconciling — bridge deregistered cleanly
