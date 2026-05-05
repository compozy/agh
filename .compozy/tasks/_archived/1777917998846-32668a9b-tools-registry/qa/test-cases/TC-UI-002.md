# TC-UI-002 — Tool detail view shows redacted MCP auth diagnostics

- **Priority:** P1
- **Type:** UI / security
- **Trace:** Task 13, ADR-010, Safety Invariant 20

## Objective

Prove the detail view for an MCP tool shows redacted `auth_status` mirroring `/api/settings/mcp-servers` without rendering token material.

## Test Steps

1. Open MCP tool with `authenticated` status.
   - **Expected:** Status badge says `authenticated`; details show `client_id`, `scopes`, `expires_at`, `refreshable`. No raw tokens in DOM, no tokens in Redux/state, no tokens in network response visible to web client.
2. Open MCP tool with `expired`.
   - **Expected:** Reason `mcp_auth_expired`; helper copy points to `agh mcp auth status --refresh <server>` (existing CLI path), not an inline login control.
3. Inspect `network` tab and DOM for sentinel `mcp:test:bearer:OAUTHTOKEN_v1`.
   - **Expected:** Zero matches.

## Automation

- **Target:** E2E
- **Status:** Existing
- **Command/Spec:** `make test-e2e-web` (`tools-detail` spec); sentinel scan in Task 16
