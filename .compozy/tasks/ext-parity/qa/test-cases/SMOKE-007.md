# SMOKE-007: Tool Publication via Resource Snapshot

**Priority:** P0
**Type:** Smoke
**Package:** internal/tools
**Related Tasks:** 08

## Objective

Validate that an extension can publish tool definitions via the resources/snapshot mechanism, that published tools appear in the canonical tool store, and that the legacy provide_tools handshake method is no longer required. This confirms the migration from the imperative tool registration protocol to the declarative resource-based approach.

## Preconditions

- Resource store initialized with the tool.definition kind codec registered
- Reconcile driver configured with the tools projector
- An extension session established with granted_resource_kinds including "tool.definition"
- Canonical tool store wired to read from reconciled tool projections

## Test Steps

1. **Publish tool records via resources/snapshot** from the extension, sending a snapshot containing two tool.definition resources:
   - Tool A: name="search_files", description="Search files by pattern", input_schema with a "pattern" string field
   - Tool B: name="read_file", description="Read file contents", input_schema with a "path" string field
   **Expected:** Snapshot is accepted. Both tool.definition resources are persisted with version=1 and owner_kind/owner_id matching the extension session.

2. **Trigger reconciliation** so the tools projector processes the new tool definitions.
   **Expected:** Reconciliation completes without error.

3. **Query the canonical tool store** for available tools.
   **Expected:** Both "search_files" and "read_file" appear in the tool list with correct names, descriptions, and input schemas.

4. **Verify tool metadata** includes the source extension identity.
   **Expected:** Each tool record carries owner_kind="extension" and owner_id matching the publishing extension's ID.

5. **Publish an updated snapshot** with only Tool A (Tool B removed) and Tool A's description changed to "Search files recursively".
   **Expected:** Snapshot is accepted. After reconciliation, the tool store contains only "search_files" with the updated description. "read_file" is removed (snapshot semantics = full replacement).

6. **Verify provide_tools is not required** by starting a new extension session that does NOT send a provide_tools capability, but does publish tool resources via snapshot.
   **Expected:** Tools appear in the canonical store regardless of provide_tools presence. No error about missing provide_tools.

## Edge Cases

- A snapshot with zero tool definitions removes all tools previously owned by that extension
- A snapshot with a tool whose input_schema is invalid JSON fails validation at persist time
- Two extensions publishing tools with the same name: both appear, disambiguated by owner (no silent overwrite)
- A tool.definition with a missing name field fails validation
- Publishing a snapshot for a kind not in granted_resource_kinds is rejected
- Rapid sequential snapshots (publish, then immediately publish again) resolve to the last snapshot state after reconciliation
- A tool record with an extremely large input_schema (many fields) persists and round-trips correctly
