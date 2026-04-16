# TC-INT-014: Boot rebuild rehydrates tool and MCP state

**Priority:** P1
**Type:** Integration
**Package:** internal/daemon
**Related Tasks:** 08

## Objective

Validate that persisted `tool` and `mcp_server` resource records survive a daemon restart and are correctly rehydrated into the runtime state. After boot, tools and MCP servers must be available as if they had just been published, without any extension re-publishing.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Pre-populated resource records: 3 `tool` records and 2 `mcp_server` records from various sources
- Daemon boot sequence that reads resource records and rebuilds runtime state
- Projectors for `tool` and `mcp_server` kinds wired into the daemon

## Test Steps

1. Persist 3 `tool` resource records (`boot-tool-1`, `boot-tool-2`, `boot-tool-3`) and 2 `mcp_server` records (`mcp-srv-1`, `mcp-srv-2`) directly into the SQLite database.
   **Expected:** All 5 records present in the database.

2. Boot the daemon (or the relevant subsystem that performs boot rebuild).
   **Expected:** Boot completes without error. Projectors for `tool` and `mcp_server` are triggered.

3. Query the tool runtime (not the database — the in-memory/projected state) for available tools.
   **Expected:** All 3 tools are available: `boot-tool-1`, `boot-tool-2`, `boot-tool-3`.

4. Query the MCP server runtime for available servers.
   **Expected:** Both MCP servers are available: `mcp-srv-1`, `mcp-srv-2`.

5. Verify the rehydrated tools have correct data (descriptions, schemas, etc.).
   **Expected:** Data matches what was persisted in step 1. No data loss during rehydration.

6. Verify the rehydrated MCP servers have correct connection configurations.
   **Expected:** Server configurations match persisted records.

7. Trigger a tool call against one of the rehydrated tools.
   **Expected:** Tool is callable (or at least dispatches correctly — actual execution depends on the tool handler being available).

## Edge Cases

- Boot with zero resource records — runtime starts cleanly with empty tool/MCP catalogs
- Boot with orphaned records (source no longer exists) — records still loaded, no error
- Boot with corrupted data in one record — other records still load, corrupted one logged and skipped
- Boot rebuild is idempotent — calling rebuild twice produces same state
- Records modified between boot start and projector run — projector picks up latest state
