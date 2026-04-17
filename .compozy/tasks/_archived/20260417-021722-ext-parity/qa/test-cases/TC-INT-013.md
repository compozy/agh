# TC-INT-013: Dynamic snapshot replaces provide_tools

**Priority:** P1
**Type:** Integration
**Package:** internal/extension
**Related Tasks:** 08

## Objective

Validate that an extension can dynamically add, update, and remove tool records via `resources/snapshot` without using the legacy `provide_tools` path. The snapshot mechanism is the new canonical way for extensions to manage their tool catalog at runtime.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized
- Extension initialized with valid nonce and `resource_kinds=["tool"]` grant
- No legacy `provide_tools` handler registered (or if present, not invoked)

## Test Steps

1. Extension issues `resources/snapshot` with 2 tool records: `dyn-tool-1` and `dyn-tool-2`.
   **Expected:** Snapshot accepted. 2 tool records in the resource store.

2. Query the resource store for `kind=tool` from the extension's source.
   **Expected:** Exactly `dyn-tool-1` and `dyn-tool-2` present.

3. Extension issues a second `resources/snapshot` adding `dyn-tool-3` and updating `dyn-tool-1` (new description).
   **Expected:** Snapshot accepted. Store contains `dyn-tool-1` (updated), `dyn-tool-2`, `dyn-tool-3`.

4. Verify `dyn-tool-1`'s data payload reflects the updated description.
   **Expected:** Description matches the second snapshot's value, not the original.

5. Extension issues a third `resources/snapshot` containing only `dyn-tool-3` (omitting `dyn-tool-1` and `dyn-tool-2`).
   **Expected:** Snapshot accepted. `dyn-tool-1` and `dyn-tool-2` removed. Only `dyn-tool-3` remains.

6. Verify `dyn-tool-1` and `dyn-tool-2` are no longer in the resource store.
   **Expected:** Queries for these IDs return no results.

7. Verify that no `provide_tools` RPC call was made during any of these operations.
   **Expected:** The legacy path was not invoked. All tool management happened through `resources/snapshot`.

## Edge Cases

- Snapshot with 0 tools — removes all tools for this source, effectively a "clear"
- Rapid sequential snapshots — each fully replaces the previous, no merge
- Snapshot during active tool execution — in-flight tool call completes with old definition, new definition applies to subsequent calls
- Extension without tool grant tries to snapshot tools — rejected with permission error
- Very large tool catalog (100+ tools in one snapshot) — accepted within reasonable time
