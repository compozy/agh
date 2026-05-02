## TC-SEC-001: MCP OAuth PKCE Lifecycle And Redaction

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 55 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that remote MCP servers authenticate through OAuth 2.1 authorization code with PKCE, persist refreshable tokens durably, refresh/logout correctly, and never expose token material through config, CLI, API, settings, logs, fixtures, or docs examples.

### Traceability

- Task: task_05, MCP Auth and Skill Security.
- TechSpec: issue 27; Testing Approach OAuth PKCE generation, callback verification, refresh, redaction, and logout.
- ADR: ADR-003 first-class MCP OAuth auth subsystem.
- Surfaces: `internal/mcp/auth`, `internal/store/globaldb`, `internal/config`, `internal/api/contract/settings.go`, `internal/api/core/settings.go`, `internal/cli`, web settings MCP surfaces, site MCP auth docs.

### Preconditions

- Local mock OAuth authorization server supports metadata, authorization, token, refresh, and optional revocation endpoints.
- Remote MCP config uses token-free auth metadata and `client_secret_ref` only when needed.
- Sentinel values exist for access token, refresh token, authorization code, PKCE verifier, and client secret.

### Test Steps

1. Start OAuth login for a configured remote MCP server.
   - **Expected:** Authorization URL includes state and S256 PKCE challenge; verifier remains internal and is not printed or logged.

2. Complete callback exchange with the mock OAuth server.
   - **Expected:** State mismatch is rejected, valid callback stores token material durably, and CLI reports redacted authenticated status.

3. Force token refresh.
   - **Expected:** Durable token storage updates expiry/token metadata while CLI/API/settings surfaces continue to expose only redacted auth status.

4. Run `agh mcp auth status`, settings API, and config show/list output in human and JSON modes.
   - **Expected:** No access token, refresh token, authorization code, PKCE verifier, or client secret value appears.

5. Run logout.
   - **Expected:** Token is revoked when supported or deleted locally, and status reports unauthenticated without leaving stale token material.

6. Review web fixtures and site docs.
   - **Expected:** Web settings treats remote rows as auth-managed and token-free; docs state tokens are outside `mcp.json` and redacted from operator outputs.

### Evidence To Capture

- `qa/logs/TC-SEC-001/go-test-mcp-auth.log`
- `qa/logs/TC-SEC-001/mcp-auth-status.json`
- `qa/logs/TC-SEC-001/settings-mcp.json`
- `qa/logs/TC-SEC-001/redaction-grep.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| State mismatch | Wrong OAuth state | Exchange rejected and no token saved |
| Metadata lacks S256 | Unsupported challenge methods | Login rejected |
| Expired token | Refreshable token set | Refresh updates durable state |
| Secret in config/log fixture | Sentinel values | Redacted everywhere except approved auth store |

### Related Test Cases

- TC-UI-002: Web settings MCP/auth redaction.
- TC-REG-002: MCP docs consistency.
