# TC-FUNC-039 — Upstream `notifications/tools/list_changed` is cache invalidation only

- **Priority:** P2
- **Type:** Functional / MCP cache
- **Trace:** Task 09, TechSpec MCP Library Adoption

## Objective

Prove that an upstream MCP server emitting `notifications/tools/list_changed` during an active session is treated only as a cache-invalidation hint; it must not mutate registry structure directly. MVP refreshes descriptors on demand.

## Test Steps

1. Configure fake remote MCP server that pushes `notifications/tools/list_changed` mid-session.
2. Inspect registry state immediately after the notification.
   - **Expected:** No mutation; cache flag set.
3. Trigger a projection rebuild or call.
   - **Expected:** Daemon issues fresh `tools/list`; updated descriptors land.
4. Confirm AGH does not maintain standalone notification subscriptions outside an active client session.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestRemoteListChangedHint`
