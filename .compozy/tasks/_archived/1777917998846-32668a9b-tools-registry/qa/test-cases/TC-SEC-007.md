# TC-SEC-007 — MCP auth status maps to deterministic redacted reason codes

- **Priority:** P1
- **Type:** Security / diagnostics
- **Trace:** Task 09, ADR-010, TechSpec MCP auth-status table

## Objective

Prove the mapping between `internal/mcp/auth.StatusValue` values and registry availability reason codes is exact and never exposes token material.

## Preconditions

- Fake MCP servers configured to produce each status: `unconfigured`, `needs_login`, `authenticated`, `expired`, `invalid`.

## Test Steps

For each status:

1. `agh tool info mcp__<server>__<tool> -o json`.
   - **Expected mapping:**
     - `unconfigured` → `mcp_auth_unconfigured`
     - `needs_login` → `mcp_auth_required`
     - `authenticated` → no auth-related reason code
     - `expired` → `mcp_auth_expired`
     - `invalid` → `mcp_auth_invalid`
2. Confirm `MCPAuthStatus` view exposes `server_name`, `auth_type`, `client_id`, `scopes`, `expires_at`, `refreshable`, `token_present`, `diagnostic` and never access tokens, refresh tokens, OAuth codes, PKCE verifiers, client secrets, approval tokens, or hosted MCP bind nonces.
3. `GET /api/sessions/{id}/tools` — confirm session projections do not include `MCPAuthStatus`; tools are hidden or denied via reason code.

## Edge Cases

- For `authenticated`, transient backend failure on first call must NOT switch the displayed status to `invalid` until the auth subsystem records the change.
- `expired` + non-refreshable token must remain `mcp_auth_expired` (TC-SEC-006 covers refresh attempts).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestAuthStatusMapping`
