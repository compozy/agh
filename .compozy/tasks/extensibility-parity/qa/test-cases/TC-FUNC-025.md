# TC-FUNC-025: Bridge projector Build does not open speculative connections

**Priority:** P0
**Type:** Functional
**Package:** internal/bridges
**Related Tasks:** 11

## Objective

Validate that the bridge projector's Build phase computes a delta plan (which bridge connections need to be added, removed, or reconfigured) purely from resource record data without opening any live network connections. Connection establishment must only occur during Apply, which atomically swaps the bridge registry. This prevents speculative connections from consuming resources, leaking sockets, or creating authentication side effects during the planning phase.

## Preconditions

- Bridge projector is instantiated with an initial registry containing one active bridge connection: "ext-github" connected to a GitHub MCP server.
- The resource store contains the corresponding bridge.instance record for "ext-github".
- New bridge.instance resource records are prepared: "ext-slack" (Slack bridge) and an updated "ext-github" (changed endpoint URL).
- A network monitor or mock transport is in place to detect any outgoing connection attempts.

## Test Steps

1. Snapshot the live bridge registry: list all active connections and their states.
   **Expected:** One entry: "ext-github" with status connected and the original endpoint URL.

2. Insert the new bridge.instance resource record "ext-slack" into the store.
   **Expected:** Record persisted.

3. Update the bridge.instance resource record "ext-github" with a new endpoint URL.
   **Expected:** Record updated, version incremented.

4. Reset the network monitor's connection log to zero.
   **Expected:** Monitor shows no outgoing connections recorded.

5. Call the bridge projector's Build method.
   **Expected:** Build returns a delta plan: add "ext-slack", update "ext-github" (endpoint change). No errors.

6. Check the network monitor's connection log immediately after Build.
   **Expected:** Zero outgoing connection attempts. No TCP SYN packets, no TLS handshakes, no authentication requests were made during Build.

7. Query the live bridge registry.
   **Expected:** Registry is unchanged: still only "ext-github" with the original endpoint URL. "ext-slack" is not present.

8. Call the bridge projector's Apply method with the Build result.
   **Expected:** Apply completes. The network monitor now shows connection attempts to the new "ext-slack" endpoint and the updated "ext-github" endpoint.

9. Query the live bridge registry after Apply.
   **Expected:** Two entries: "ext-github" with the new endpoint URL (reconnected), "ext-slack" with status connected. The registry swap was atomic — no intermediate state was observable.

## Edge Cases

- Build with a bridge.instance that has invalid connection parameters (e.g., malformed URL): Build should still succeed (it only computes the plan). The error surfaces during Apply when the connection attempt fails.
- Build with 50+ bridge.instance records: Build completes in bounded time without proportional network I/O.
- Apply failure for one bridge (e.g., "ext-slack" unreachable): partial Apply behavior — verify whether the entire Apply rolls back or only the failed bridge is marked degraded (see TC-FUNC-026).
- Build after removing a bridge.instance record: delta plan includes a removal. Apply closes the live connection.
- Concurrent Build calls: no duplicate connection attempts; builds are serialized or only the latest takes effect.
