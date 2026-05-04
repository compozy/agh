# TC-UI-003 — Web does not render unsupported login/approval/invoke controls

- **Priority:** P1
- **Type:** UI / truthfulness
- **Trace:** Task 13, ADR-006

## Objective

Prove web UI never invents controls that the daemon does not support.

## Test Steps

1. Render full operator tools page; capture all clickable elements.
   - **Expected:** No "Log in" button for MCP servers (MCP login remains `agh mcp auth login` CLI).
   - **Expected:** No "Approve once" button for individual tool calls (approval flow uses CLI/HTTP/UDS approval-token issuance or the hosted MCP approval bridge).
   - **Expected:** No "Invoke" button outside dedicated diagnostic views that explicitly call documented daemon endpoints.
2. Each call-to-action maps to a documented daemon contract from `internal/api/contract`.

## Automation

- **Target:** E2E + Manual
- **Status:** Existing partial; Manual reviewer signs off on copy
- **Command/Spec:** `make test-e2e-web`; manual review during Task 16
