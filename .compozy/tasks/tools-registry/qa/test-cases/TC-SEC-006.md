# TC-SEC-006 — Remote MCP refresh attempts at most one refresh and never bootstraps a new login

- **Priority:** P0
- **Type:** Security / credential lifecycle
- **Trace:** Task 09, ADR-010, ADR-011, Safety Invariant 23

## Objective

Prove `MCPCallExecutor` calls `internal/mcp/auth.Service.Refresh` at most once before client creation or one retry after an outbound auth failure. It must never start a new login flow and must return deterministic redacted reason codes when refresh is impossible.

## Preconditions

- Fake remote HTTP MCP server.
- Fake OAuth issuer that fails `refresh_token` exchange.
- Stored token state is `expired` and `refreshable = true`.

## Test Steps

1. Invoke an MCP tool from the fake server.
   - **Expected:** Executor performs exactly one refresh attempt; refresh fails; call returns `mcp_auth_refresh_failed`.
2. Inspect daemon log for refresh attempts.
   - **Expected:** Exactly one `internal/mcp/auth.Service.Refresh` call; no `OAuth login` initiation; no `client.NewOAuthStreamableHttpClient`/`client.NewOAuthSSEClient`/`MemoryTokenStore` instantiation.
3. Repeat the call.
   - **Expected:** Still `mcp_auth_refresh_failed` (no infinite loop, no further refreshes).
4. Run `agh mcp auth status fake_http -o json`.
   - **Expected:** Status reflects `expired`/`invalid` truthfully.
5. Run `agh mcp auth status --refresh fake_http`.
   - **Expected:** Manual refresh path is the only recovery; registry never starts it implicitly.

## Edge Cases

- Token in `needs_login` state must NOT trigger any refresh attempt; call returns `mcp_auth_required` immediately.
- After successful manual `agh mcp auth login`, the next call refreshes the executor's cached client and proceeds.

## Automation

- **Target:** Integration
- **Status:** Existing partial; Missing zero-login-bootstrap assertion
- **Command/Spec:** `go test ./internal/mcp -run TestRemoteAuthRefreshBoundedOnce`
- **Notes:** Prevents regression toward library-managed OAuth flow.
